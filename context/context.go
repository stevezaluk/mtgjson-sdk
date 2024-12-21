package context

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/auth0/go-auth0/authentication"
	"github.com/auth0/go-auth0/management"
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

/*
Initialize viper to parse our config file or use environmental varibales to provide
the values we need. Additionally, a config path can be passed to the function to override
the default value
*/
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

/*
Initialize our Slog Multihandler to write logs both to STDOUT and to our log directory. Logger
is then stored within the ServerContext. Logs are written in text output when sent to STDOUT to
make them more readable.
*/
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

/*
Fetch the Logger object that is stored in the ServerContext
*/
func GetLogger() *slog.Logger {
	logger := ServerContext.Value("logger")

	return logger.(*slog.Logger)
}

/*
Initialize our MongoDB instance using values stored within viper, and store
it within the ServerContext
*/
func InitDatabase() {
	database := &server.Database{
		IPAddress: viper.GetString("mongo.ip"),
		Port:      viper.GetInt("mongo.port"),
		Username:  viper.GetString("mongo.user"),
		Password:  viper.GetString("mongo.pass"),
	}

	database.Connect() // externalize errors to here and check

	ctx := context.WithValue(ServerContext, "database", database)
	ServerContext = ctx
}

/*
Fetch the Database object that is stored in the ServerContext
*/
func GetDatabase() *server.Database {
	database := ServerContext.Value("database")

	return database.(*server.Database)
}

/*
Disconnect the database object that is stored in the ServerContext
*/
func DestroyDatabase() {
	var database = GetDatabase()
	database.Disconnect()
}

/*
Initialize the Authentication client used for logging in and registering users.
Then store it within the ServerContext.
*/
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

/*
Initialize the Authentication management client used for resetting user passwords and removing users
from Auth0, then store it within the Server Context
*/
func InitAuthManagementAPI() {
	domain := viper.GetString("auth0.domain")
	clientId := viper.GetString("auth0.client_id")
	clientSecret := viper.GetString("auth0.client_secret")

	managementAPI, err := management.New(
		domain,
		management.WithClientCredentials(context.TODO(), clientId, clientSecret),
	)

	if err != nil {
		panic(err)
	}

	ctx := context.WithValue(ServerContext, "management", managementAPI)
	ServerContext = ctx
}

/*
Fetch the Authentication management client object that is stored in the ServerContext
*/
func GetAuthManagementAPI() *management.Management {
	managementAPI := ServerContext.Value("management")

	return managementAPI.(*management.Management)
}

/*
Fetch the Authentication client object that is stored in the ServerContext
*/
func GetAuthAPI() *authentication.Authentication {
	authAPI := ServerContext.Value("auth")

	return authAPI.(*authentication.Authentication)
}
