package server

import (
	"context"
	"log/slog"

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
	Client   *mongo.Client
	Database *mongo.Database
}

/*
Connect to the MongoDB instance defined in the Database object
*/
func (d *Database) Connect(uri string) {
	opts := options.Client()

	opts.ApplyURI(uri)

	slog.Info("Connecting to mongoDB")
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		slog.Error("Failed to connect to MongoDB", "uri", uri)
		panic(1) // panic here as this is a fatal error
	}

	d.Database = client.Database("mtgjson")
	d.Client = client
}

/*
Disconnect Gracefully disconnect from your active MongoDB connection
*/
func (d *Database) Disconnect() {
	d.Health() // this will throw a fatal error when

	slog.Info("Disconnecting from MongoDB")
	err := d.Client.Disconnect(context.Background())
	if err != nil {
		slog.Error("Failed to disconnect from MongoDB", "err", err.Error())
		panic(1)
	}
}

/*
Health Ping the MongoDB database and panic if we don't get a response
*/
func (d *Database) Health() {
	err := d.Client.Ping(context.TODO(), nil)
	if err != nil {
		slog.Error("Failed to ping MongoDB for health", "err", err.Error())
		panic(1)
	}
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
BuildDatabaseURI Build a MongoDB connection URI using the values that are stored within our database object
*/
func BuildDatabaseURI(ipAddress string, port int, username string, password string) string {
	return "mongodb://" + username + ":" + password + "@" + ipAddress + ":" + strconv.Itoa(port)
}
