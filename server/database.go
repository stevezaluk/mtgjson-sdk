package server

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database struct {
	Config   Config
	Client   *mongo.Client
	Database *mongo.Database
}

func (d *Database) Connect() {
	opts := options.Client()
	uri := d.Config.BuildUri()

	opts.ApplyURI(uri)

	fmt.Println("[info] Connecting to MongoDB")
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		fmt.Printf("[error] Failed to connect to MongoDB Database: %s", uri)
		panic(1) // panic here as this is a fatal error
	}

	d.Database = client.Database("mtgjson")
	d.Client = client
}

func (d Database) Disconnect() {
	d.Health() // this will throw an fatal error when

	fmt.Println("[info] Disconnecting from MongoDB") // these will be replaced with proper logging in a future PR
	err := d.Client.Disconnect(context.Background())
	if err != nil {
		fmt.Println("[error] Failed to disconnect from MongoDB: ", err)
		panic(1)
	}
}

func (d Database) Health() {
	err := d.Client.Ping(context.TODO(), nil)
	if err != nil {
		fmt.Println("[error] Failed to ping MongoDB")
		panic(1)
	}
}

func (d Database) Find(collection string, query bson.M, model interface{}) any {
	coll := d.Database.Collection(collection)

	err := coll.FindOne(context.TODO(), query).Decode(model)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil // log here
		}
	}

	return model
}

func (d Database) Replace(collection string, query bson.M, model interface{}) any {
	coll := d.Database.Collection(collection)

	result, err := coll.ReplaceOne(context.TODO(), query, model)
	if err == mongo.ErrNoDocuments {
		return nil
	}

	return result
}

func (d Database) Delete(collection string, query bson.M) *mongo.DeleteResult {
	coll := d.Database.Collection(collection)

	result, err := coll.DeleteOne(context.TODO(), query)
	if err == mongo.ErrNoDocuments {
		return nil
	}

	return result
}

func (d Database) Insert(collection string, model interface{}) any {
	coll := d.Database.Collection(collection)

	result, err := coll.InsertOne(context.TODO(), model)
	if err != nil {
		return nil
	}

	return result
}

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
