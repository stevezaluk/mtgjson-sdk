package server

/*
Server - An abstraction of the Server running the MTGJSON-API. Includes any components
necessary for functionality such as the Database, Logging, etc.
*/
type Server struct {
	// database - A pointer to the current database structure
	database *Database

	// log - A pointer to the current log structure
	log *Log

	// authenticationManager - Provides logic for interacting with Auth0
	authenticationManager *AuthenticationManager
}

/*
New - Basic constructor for the Server structure
*/
func New(database *Database, log *Log, authenticationManager *AuthenticationManager) *Server {
	return &Server{
		database:              database,
		log:                   log,
		authenticationManager: authenticationManager,
	}
}

/*
FromConfig - Initializes a new server object using config values from viper
*/
func FromConfig() (*Server, error) {
	authManager, err := NewAuthenticationManagerFromConfig()
	if err != nil {
		return nil, err
	}
	return New(
		NewDatabaseFromConfig(), // this is not connecting to the database here
		NewLoggerFromConfig(),
		authManager,
	), nil
}

/*
Database - Returns a pointer to the currently used Database object for the server
*/
func (server *Server) Database() *Database {
	return server.database
}

/*
Log - Returns a pointer to the currently used Log object for the server
*/
func (server *Server) Log() *Log {
	return server.log
}

/*
AuthenticationManager - Returns a pointer to the AuthenticationManager for the server
*/
func (server *Server) AuthenticationManager() *AuthenticationManager {
	return server.authenticationManager
}

/*
SetAuthManager - Set the auth manager for the Server
*/
func (server *Server) SetAuthManager(auth *AuthenticationManager) {
	server.authenticationManager = auth
}
