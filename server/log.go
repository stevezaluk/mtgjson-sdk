package server

import (
	"log/slog"
	"os"
)

/*
Log - An abstraction of the slog logger used for the SDK
*/
type Log struct {
	// logPath - The path that log files should be saved too
	logPath string

	// logFile - The file descriptor for the currently open log file
	logFile *os.File

	// logger - A pointer to the slog.Logger object being used by the logger
	logger *slog.Logger
}
