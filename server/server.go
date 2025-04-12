package server

/*
Server - An abstraction of the Server running the MTGJSON-API. Includes any components
necessary for functionality such as the Database, Logging, etc.
*/
type Server struct {
	Database *Database
}
