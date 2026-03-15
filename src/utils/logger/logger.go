package logger

import (
	"fmt"
	syslog "log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

type CustomLogger struct {
	*syslog.Logger
	uniqueid string
	urlpath  string
}

func New(urlpath string, uniqueid string) *CustomLogger {
	logger := syslog.New(os.Stdout, "", 0)

	return &CustomLogger{
		Logger:   logger,
		uniqueid: uniqueid,
		urlpath:  urlpath,
	}
}

func (c *CustomLogger) Info(v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	c.Logger.Printf("INFO: %s [%s] [%s] [%s:%d] %s\n", getTimestamp(), c.uniqueid, c.urlpath, getPathFile(file), line, v)
}

func (c *CustomLogger) Error(err error, v ...interface{}) {
	_, file, line, _ := runtime.Caller(1)
	c.Logger.Printf("ERROR: %s [%s] [%s] [%s:%d] %s, Error: %v\n", getTimestamp(), c.uniqueid, c.urlpath, getPathFile(file), line, v, err)
}

func (c *CustomLogger) RecoverInfo(frame uintptr, file string, line int, v ...interface{}) {
	c.Logger.Printf("RECOVERED-INFO: %s [%s] [%s] [%s:%d] %s\n", getTimestamp(), c.uniqueid, c.urlpath, getPathFile(file), line, v)
}

func (c *CustomLogger) RecoverError(frame uintptr, file string, line int, err error, v ...interface{}) {
	c.Logger.Printf("RECOVERED-ERROR: %s [%s] [%s] [%s:%d] %s, Stack trace: %v %s\n", getTimestamp(), c.uniqueid, c.urlpath, getPathFile(file), line, v, err, strings.ReplaceAll(fmt.Sprintf("stackTraces: %s", debug.Stack()), "\n", ";"))
}

func getPathFile(file string) string {

	targetpath := file

	cwd, err := os.Getwd()
	if err == nil {
		if rel, err := filepath.Rel(cwd, file); err == nil {
			targetpath = rel
		}
	}

	if _, after, found := strings.Cut(targetpath, "src"); found {
		return "src" + after
	}

	return targetpath
}

func getFungsi(frame uintptr) string {
	return runtime.FuncForPC(frame).Name()
}

func getTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05.000")
}
