package server

import (
	"context"
	"github.com/spf13/viper"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strconv"
)

/*
Database An abstraction of an active mongodb database connection. The same connection is re-used across
all SDK operations to ensure that we don't exceed the connection pool limit
*/
type Database struct {
	// options - A structure containing options used for connecting the MongoDB Client
	options *options.ClientOptions

	// defaultDatabase - The default database that MongoDB should connect to
	defaultDatabase string

	// Client - A pointer to the MongoDB client that facilitates a connection to the database
	Client *mongo.Client

	// Database - A pointer to the MongoDB database that the API interacts with
	Database *mongo.Database
}

/*
NewDatabase - Instantiate a new database object. Does not connect automatically, this needs to be
done with Database.Connect()
*/
func NewDatabase(hostname string, port int, defaultDatabase string) *Database {
	hosts := hostname + ":" + strconv.Itoa(port)

	clientOpts := options.Client().
		SetHosts([]string{hosts}).
		SetDirect(true).
		SetServerSelectionTimeout(30 * time.Second).
		SetTimeout(30 * time.Second)

	return &Database{
		options:         clientOpts,
		defaultDatabase: defaultDatabase,
	}
}

/*
NewDatabaseFromConfig - Instantiate a new database from viper config values
*/
func NewDatabaseFromConfig() *Database {
	database := NewDatabase(
		viper.GetString("mongo.hostname"),
		viper.GetInt("mongo.port"),
		viper.GetString("mongo.default_database"))

	database.SetSCRAMAuthentication(
		viper.GetString("mongo.username"),
		viper.GetString("mongo.password"))

	return database
}

/*
SetSCRAMAuthentication - Set the credentials for the database if they are needed
*/
func (d *Database) SetSCRAMAuthentication(username string, password string) {
	credentials := options.Credential{
		AuthMechanism: "SCRAM-SHA-256",
		AuthSource:    "admin",
		Username:      username,
		Password:      password,
	}

	d.options.SetAuth(credentials)
}

/*
Connect to the MongoDB instance defined in the Database object
*/
func (d *Database) Connect() error {
	client, err := mongo.Connect(context.Background(), d.options)
	if err != nil {
		return err
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		return err
	}

	d.Client = client
	d.Database = client.Database(d.defaultDatabase)

	return nil
}

/*
Disconnect Gracefully disconnect from your active MongoDB connection
*/
func (d *Database) Disconnect() error {
	err := d.Client.Disconnect(context.Background())
	if err != nil {
		return err
	}

	return nil
}

/*
Find a single document from the MongoDB instance and unmarshal it into the interface
passed in the 'model' parameter
*/
func (d *Database) Find(collection string, query bson.M, model interface{}) bool {
	coll := d.Database.Collection(collection)

	slog.Debug("FindOne Query", "collection", collection, "query", query)
	err := coll.FindOne(context.TODO(), query).Decode(model)
	if err != nil {
		slog.Error("Error during FineOne Query", "collection", collection, "query", query, "err", err)
		return false
	}

	return true
}

/*
FindMultiple - Find multiple documents from within a collection
*/
func (d *Database) FindMultiple(collection string, key string, value []string, model interface{}) bool {
	coll := d.Database.Collection(collection)

	slog.Debug("FindMultiple Query", "collection", collection, "key", key, "value", value)
	query := bson.M{key: bson.M{"$in": value}}
	cur, err := coll.Find(context.TODO(), query)
	if err != nil {
		slog.Error("Error during FindMultiple Query", "collection", collection, "key", key, "value", value, "err", err)
		return false
	}

	err = cur.All(context.TODO(), model)
	if err != nil {
		slog.Error("Error decoding FindMultiple Query", "collection", collection, "key", key, "value", value, "err", err)
		return false
	}

	return true
}

/*
Replace a single document from the MongoDB instance and unmarshal it into the interface
passed in the 'model' parameter
*/
func (d *Database) Replace(collection string, query bson.M, model interface{}) (*mongo.UpdateResult, bool) {
	coll := d.Database.Collection(collection)

	slog.Debug("ReplaceOne Query", "collection", collection, "query", query)
	result, err := coll.ReplaceOne(context.TODO(), query, model)
	if err != nil {
		return nil, false
	}

	return result, true
}

/*
Delete a single document from the MongoDB instance
*/
func (d *Database) Delete(collection string, query bson.M) (*mongo.DeleteResult, bool) {
	coll := d.Database.Collection(collection)

	slog.Debug("DeleteOne Query", "collection", collection, "query", query)
	result, err := coll.DeleteOne(context.TODO(), query)
	if err != nil { // includes ErrNoDocuments
		slog.Error("Error during DeleteOne query", "collection", collection, "query", query, "err", err)
		return nil, false
	}

	if result.DeletedCount < 1 {
		return result, false
	}

	return result, true
}

/*
Insert the interface represented in the 'model' parameter into the MongoDB
instance
*/
func (d *Database) Insert(collection string, model interface{}) (*mongo.InsertOneResult, bool) {
	coll := d.Database.Collection(collection)

	slog.Debug("InsertOne Query", "collection", collection)
	result, err := coll.InsertOne(context.TODO(), model)
	if err != nil {
		slog.Debug("Error during InsertOne Query", "collection", collection, "err", err)
		return nil, false
	}

	return result, true
}

/*
Index Return all documents in a collection and unmarshal them into the interface passed
in the 'model' parameter
*/
func (d *Database) Index(collection string, limit int64, model interface{}) bool {
	opts := options.Find().SetLimit(limit)
	coll := d.Database.Collection(collection)

	slog.Debug("Index Collection Query", "collection", collection)
	cur, err := coll.Find(context.TODO(), bson.M{}, opts)
	if err != nil {
		slog.Error("Error during Indexing Collection", "collection", collection, "limit", limit, "err", err)
		return false
	}

	err = cur.All(context.TODO(), model)
	if err != nil { // includes ErrNoDocuments
		slog.Error("Error during Marshaling index results", "collection", collection, "limit", limit, "err", err)
		return false
	}

	return true
}

/*
SetField Update a single field in a requested document in the Mongo Database
*/
func (d *Database) SetField(collection string, query bson.M, fields bson.M) (*mongo.UpdateResult, bool) {
	coll := d.Database.Collection(collection)

	slog.Debug("SetField Query", "collection", collection, "query", query, "fields", fields)
	results, err := coll.UpdateOne(context.TODO(), query, bson.M{"$set": fields})
	if err != nil {
		slog.Error("Error during SetField Operation", "collection", collection, "query", query, "fields", fields, "err", err)
		return nil, false
	}

	return results, true
}

/*
AppendField Append an item to a field in a single document in the Mongo Database
*/
func (d *Database) AppendField(collection string, query bson.M, fields bson.M) (*mongo.UpdateResult, bool) {
	coll := d.Database.Collection(collection)

	slog.Debug("AppendField Query", "collection", collection, "query", query, "fields", fields)
	results, err := coll.UpdateOne(context.TODO(), query, bson.M{"$push": fields})
	if err != nil {
		slog.Error("Error during AppendField Operation", "collection", collection, "query", query, "fields", fields, "err", err)
		return nil, false
	}

	return results, true
}

/*
PullField Remove all instances of an object from an array in a single document
*/
func (d *Database) PullField(collection string, query bson.M, fields bson.M) (*mongo.UpdateResult, bool) {
	coll := d.Database.Collection(collection)

	slog.Debug("PullField Query", "collection", collection, "query", query, "fields", fields)
	results, err := coll.UpdateOne(context.TODO(), query, bson.M{"$pull": fields})
	if err != nil {
		slog.Error("Error during PullField Operation", "collection", collection, "query", query, "fields", fields, "err", err)
		return nil, false
	}

	return results, true
}

/*
IncrementField Increment a single field in a document
*/
func (d *Database) IncrementField(collection string, query bson.M, fields bson.M) (*mongo.UpdateResult, bool) {
	coll := d.Database.Collection(collection)

	slog.Debug("IncrementField Query", "collection", collection, "query", query, "fields", fields)
	results, err := coll.UpdateOne(context.TODO(), query, bson.M{"$inc": fields})
	if err != nil {
		slog.Error("Error during IncrementField Operation", "collection", collection, "query", query, "fields", fields, "err", err)
		return nil, false
	}

	return results, true

}
