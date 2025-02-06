package context

import (
	"context"
	"github.com/auth0/go-auth0/authentication"
	"github.com/auth0/go-auth0/management"
	"github.com/mitchellh/go-homedir"
	slogmulti "github.com/samber/slog-multi"
	"github.com/spf13/viper"
	"github.com/stevezaluk/mtgjson-sdk/server"
	"log/slog"
	"os"
	"time"
)

const (
	defaultConfigPath = "/.config/mtgjson-api/"
	defaultConfigName = "config.json"
)

/*
ServerContext -
*/
type ServerContext struct {
	logger            *slog.Logger
	logFile           *os.File
	database          *server.Database
	authAPI           *authentication.Authentication
	authManagementAPI *management.Management

	context context.Context
}

/*
NewServerContext - Constructor for the server context. Returns an empty pointer to the server context
*/
func NewServerContext() *ServerContext {
	return &ServerContext{context: context.Background()}
}

/*
WithLog - Set the logger for the server context
*/
func (ctx *ServerContext) WithLog(logPath string) {
	filename := logPath + "/api-" + time.Now().Format(time.RFC3339Nano) + ".json"
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		panic(err)
	}

	multiHandler := slogmulti.Fanout(
		slog.NewJSONHandler(file, nil),
		slog.NewTextHandler(os.Stdout, nil))

	ctx.logger = slog.New(multiHandler)
	slog.SetDefault(ctx.logger)

}

/*
WithDatabase - Set the database instance for the server context. The database will be connected on calling this function
*/
func (ctx *ServerContext) WithDatabase(ipAddress string, port int, username string, password string) {
	database := &server.Database{}

	viper.Set("mongo.uri", server.BuildDatabaseURI(ipAddress, port, username, password))
	database.Connect(viper.GetString("mongo.uri"))
}

/*
WithAuthAPI - Sets the Auth0 Authentication API for the server context.
*/
func (ctx *ServerContext) WithAuthAPI(domain string, clientId string, clientSecret string) {
	authAPI, err := authentication.New(ctx.context, domain, authentication.WithClientID(clientId), authentication.WithClientSecret(clientSecret))
	if err != nil {
		panic(err)
	}

	ctx.authAPI = authAPI
}

/*
WithManagementAPI - Set the Auth0 Management API, for the server context
*/
func (ctx *ServerContext) WithManagementAPI(domain string, clientId string, clientSecret string) {
	managementAPI, err := management.New(domain, management.WithClientCredentials(ctx.context, clientId, clientSecret))
	if err != nil {
		panic(err)
	}

	ctx.authManagementAPI = managementAPI
}

/*
Database - Return a pointer to the MongoDB database
*/
func (ctx *ServerContext) Database() *server.Database {
	return ctx.database
}

/*
Logger - Return a pointer to the slog logger used by the Server Context
*/
func (ctx *ServerContext) Logger() *slog.Logger {
	return ctx.logger
}

/*
AuthAPI - Return a pointer to the Auth0 Authentication API struct
*/
func (ctx *ServerContext) AuthAPI() *authentication.Authentication {
	return ctx.authAPI
}

/*
AuthManagementAPI - Return a pointer to the Auth Management API struct
*/
func (ctx *ServerContext) AuthManagementAPI() *management.Management {
	return ctx.authManagementAPI
}

/*
Context - Return a copy of the context used for the Database
*/
func (ctx *ServerContext) Context() context.Context {
	return ctx.context
}

/*
InitConfig - Initialize viper to parse our config file or use environmental varibales to provide
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
		viper.AddConfigPath(home + defaultConfigPath)
		viper.SetConfigName(defaultConfigName)
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
}
