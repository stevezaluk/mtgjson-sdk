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
}

/*
New - Basic constructor for the Server structure
*/
func New(database *Database, log *Log) *Server {
	return &Server{
		database: database,
		log:      log,
	}
}

/*
FromConfig - Initializes a new server object using config values from viper
*/
func FromConfig() *Server {
	return New(
		NewDatabaseFromConfig(), // this is not connecting to the database here
		NewLoggerFromConfig(),
	)
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
