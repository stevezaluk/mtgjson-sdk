package context

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/auth0/go-auth0/authentication"
	"github.com/mitchellh/go-homedir"
	"github.com/samber/slog-multi"
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

func InitLog() {
	timestamp := time.Now().Format(time.RFC3339Nano)

	filename := viper.GetString("log.path") + "/api-" + timestamp + ".json"
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}

	multiHandler := slogmulti.Fanout(
		slog.NewJSONHandler(file, nil),
		slog.NewTextHandler(os.Stdout, nil),
	)

	logger := slog.New(multiHandler)

	slog.SetDefault(logger)

	ctx := context.WithValue(ServerContext, "logger", logger)
	ServerContext = ctx
}

func GetLogger() *slog.Logger {
	logger := ServerContext.Value("logger")

	return logger.(*slog.Logger)
}

func InitDatabase() {
	var database server.Database

	database.IPAddress = viper.GetString("mongo.ip")
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

func InitAuthAPI() {
	domain := viper.GetString("auth0.domain")
	clientId := viper.GetString("auth0.client_id")
	clientSecret := viper.GetString("auth0.client_secret")

	authAPI, err := authentication.New(
		context.Background(),
		domain,
		authentication.WithClientID(clientId),
		authentication.WithClientSecret(clientSecret),
	)

	if err != nil {
		panic(err)
	}

	ctx := context.WithValue(ServerContext, "auth", authAPI)
	ServerContext = ctx

}

func GetAuthAPI() *authentication.Authentication {
	authAPI := ServerContext.Value("auth")

	return authAPI.(*authentication.Authentication)
}
