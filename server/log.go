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
func NewLogger(path string) *Log {
	log := &Log{
		logPath: path,
	}

	log.init()
	return log
}

/*
NewLoggerFromConfig - Instantiate a new logger using flags from viper
*/
func NewLoggerFromConfig() *Log {
	return NewLogger(viper.GetString("log.path"))
}

/*
Path - Return the path that the Log is currently saving files to
*/
func (log *Log) Path() string {
	return log.logPath
}

/*
openFile - Creates a new log file in the Log.logPath directory
*/
func (log *Log) openFile() error {
	timestamp := time.Now().Format(time.RFC3339Nano)

	filename := log.logPath + "/" + "api-" + timestamp + ".json"
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	log.logFile = file

	return nil
}

/*
init - Initializes the internally stored logger field and sets it as the default logger
*/
func (log *Log) init() {
	var stdoutOnly bool

	err := log.openFile()
	if err != nil {
		stdoutOnly = true
	}

	handler := slogmulti.Fanout(
		slog.NewTextHandler(os.Stdout, nil))

	if !stdoutOnly {
		handler = slogmulti.Fanout(
			slog.NewJSONHandler(os.Stdout, nil),
			slog.NewTextHandler(os.Stdout, nil))
	}

	log.logger = slog.New(handler)
	slog.SetDefault(log.logger)

	if stdoutOnly {
		slog.Warn("An error occurred while opening the log file. Only STDOUT logging will be used", "err", err.Error())
	}
}

/*
CloseFile - Close the open log file, if it has been set
*/
func (log *Log) CloseFile() error {
	if log.logFile == nil {
		return nil
	}

	err := log.logFile.Close()
	if err != nil {
		return err
	}

	return nil
}
