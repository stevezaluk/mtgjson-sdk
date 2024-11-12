package context

import (
	"context"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
	"github.com/stevezaluk/mtgjson-sdk/server"
)

const (
	DEFAULT_CONFIG_PATH = "/.config/mtgjson-api/"
	DEFAULT_CONFIG_FILE = "config.json"
)

var ServerContext = context.Background()

func InitConfig(configPath string) {
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			panic(err)
		}

		viper.SetConfigType("json")
		viper.AddConfigPath(home + DEFAULT_CONFIG_PATH)
		viper.SetConfigName(DEFAULT_CONFIG_FILE)
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}

func InitDatabase() {
	var database server.Database

	database.IPAddress = viper.GetString("mongo.ip_address")
	database.Port = viper.GetInt("mongo.port")
	database.Username = viper.GetString("mongo.user")
	database.Password = viper.GetString("mongo.pass")

	database.Connect() // externalize errors to here and check

	ctx := context.WithValue(ServerContext, "database", database)
	ServerContext = ctx
}

func GetDatabase() server.Database {
	database := ServerContext.Value("database")

	return database.(server.Database)
}

func DestroyDatabase() {
	var database = GetDatabase()
	database.Disconnect()
}
