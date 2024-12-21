package server

import (
	"context"
	"log/slog"

	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*
Database An abstraction of an active mongodb database connection. The same connection is re-used across
all SDK operations to ensure that we don't exceed the connection pool limit
*/
type Database struct {
	IPAddress string
	Port      int
	Username  string
	Password  string

	Client   *mongo.Client
	Database *mongo.Database
}

/*
BuildUri Build a MongoDB connection URI using the values that are stored within our database object
*/
func (d *Database) BuildUri() string {
	s := []string{"mongodb://", d.Username, ":", d.Password, "@", d.IPAddress, ":", strconv.Itoa(d.Port)}
	return strings.Join(s, "")
}

/*
Connect to the MongoDB instance defined in the Database object
*/
func (d *Database) Connect() {
	opts := options.Client()
	uri := d.BuildUri()

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
func (d Database) Disconnect() {
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
func (d Database) Health() {
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
func (d Database) Find(collection string, query bson.M, model interface{}) any {
	coll := d.Database.Collection(collection)

	slog.Debug("Find Query", "collection", collection, "query", query)
	err := coll.FindOne(context.TODO(), query).Decode(model)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil // log here
		}
	}

	return model
}

/*
Replace a single document from the MongoDB instance and unmarshal it into the interface
passed in the 'model' parameter
*/
func (d Database) Replace(collection string, query bson.M, model interface{}) any {
	coll := d.Database.Collection(collection)

	slog.Debug("ReplaceOne Query", "collection", collection, "query", query)
	result, err := coll.ReplaceOne(context.TODO(), query, model)
	if err == mongo.ErrNoDocuments {
		return nil
	}

	return result
}

/*
Delete a single document from the MongoDB instance
*/
func (d Database) Delete(collection string, query bson.M) *mongo.DeleteResult {
	coll := d.Database.Collection(collection)

	slog.Debug("DeleteOne Query", "collection", collection, "query", query)
	result, err := coll.DeleteOne(context.TODO(), query)
	if err == mongo.ErrNoDocuments {
		return nil
	}

	return result
}

/*
Insert the interface represented in the 'model' parameter into the MongoDB
instance
*/
func (d Database) Insert(collection string, model interface{}) any {
	coll := d.Database.Collection(collection)

	slog.Debug("Insert Query", "collection", collection)
	result, err := coll.InsertOne(context.TODO(), model)
	if err != nil {
		return nil
	}

	return result
}

/*
Index Return all documents in a collection and unmarshal them into the interface passed
in the 'model' parameter
*/
func (d Database) Index(collection string, limit int64, model interface{}) interface{} {
	opts := options.Find().SetLimit(limit)
	coll := d.Database.Collection(collection)

	cur, err := coll.Find(context.TODO(), bson.M{}, opts)
	if err != nil {
		return nil
	}

	err = cur.All(context.TODO(), model)
	if err == mongo.ErrNoDocuments {
		return nil
	}

	return model
}
