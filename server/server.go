package server

/*
Server - An abstraction of the Server running the MTGJSON-API. Includes any components
necessary for functionality such as the Database, Logging, etc.
*/
type Server struct {
	database *Database
}

/*
New - Basic constructor for the Server structure
*/
func New(database *Database) *Server {
	return &Server{
		database: database,
	}
}

/*
FromConfig - Initializes a new server object using config values from viper
*/
func FromConfig() *Server {
	return New(
		NewDatabaseFromConfig(), // this is not connecting to the database here
	)
}

/*
Database - Returns a pointer to the currently used Database object for the server
*/
func (server *Server) Database() *Database {
	return server.database
}
