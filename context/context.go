package context

import (
	"context"
	config_model "github.com/stevezaluk/mtgjson-sdk/config"
	"github.com/stevezaluk/mtgjson-sdk/server"
)

var ServerContext = context.Background()

func InitConfig(config config_model.Config) {
	ctx := context.WithValue(ServerContext, "config", config)
	ServerContext = ctx
}

func InitDatabase() {
	var database server.Database
	database.Config = ServerContext.Value("config").(config_model.Config)

	database.Connect() // externalize errors to here instead of printing

	ctx := context.WithValue(ServerContext, "database", database)
	ServerContext = ctx
}

func GetDatabase() server.Database {
	database := ServerContext.Value("database")
	if database == nil {
		InitDatabase()
	}

	return database.(server.Database)
}

func DestroyDatabase() {
	var database = GetDatabase()
	database.Disconnect()
}
