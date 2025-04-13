package server

import (
	slogmulti "github.com/samber/slog-multi"
	"github.com/spf13/viper"
	"log/slog"
	"os"
	"time"
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

/*
NewLogger - Instantiate a new logger object. The 'path' parameter should
be the UNIX path of where you want logs saved to
*/
func NewLogger(path string) (*Log, error) {
	log := &Log{
		logPath: path,
	}

	err := log.init()
	if err != nil {
		return nil, err
	}

	return log, nil
}

/*
NewLoggerFromConfig - Instantiate a new logger using flags from viper
*/
func NewLoggerFromConfig() (*Log, error) {
	log, err := NewLogger(viper.GetString("log.path"))
	if err != nil {
		return nil, err
	}

	return log, nil
}

/*
init - Initializes the internally stored logger field and sets it as the default logger
*/
func (log *Log) init() error {
	timestamp := time.Now().Format(time.RFC3339Nano)

	filename := log.logPath + "/" + "api-" + timestamp + ".json"
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	log.logFile = file
	handler := slogmulti.Fanout(
		slog.NewJSONHandler(file, nil),
		slog.NewTextHandler(os.Stdout, nil))

	log.logger = slog.New(handler)
	slog.SetDefault(log.logger)

	return nil

}
