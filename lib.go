package json_logging

import (
	"encoding/json"
	"errors"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"time"
)

var isTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))

type jsons struct {
	Time int32
}

type Logger struct {
	AppName       string
	IsLoggingJSON bool
	HostName      string
	ForceJSON     bool
	ForceNonJSON  bool
}

func New(AppName string, forceJSON bool, hostName string) *Logger {

	if hostName == "" {

		hn, err := os.Hostname()

		if err != nil {
			DefaultLogger.Warn("Could not grab hostname from env.")
			hostName = "unknown_hostname"
		} else {
			hostName = hn
		}

	}

	return &Logger{
		IsLoggingJSON: !isTerminal && !forceJSON,
		AppName:       AppName,
		HostName:      hostName,
	}
}

func (l Logger) writePretty(level string, args []interface{}) {

	date := time.Now().String()
	buf, err := json.Marshal([2]string{l.AppName, date})

	if err != nil {
		panic(errors.New("could not marshal the string array"))
	}

	os.Stdout.Write(buf)
	os.Stdout.Write([]byte("\n"))
}

func (l Logger) writeJSON(level string, args []interface{}) {

	date := time.Now().String()
	buf, err := json.Marshal([4]interface{}{l.AppName, level, date, args})

	if err != nil {
		panic(errors.New("could not marshal the string array"))
	}

	os.Stdout.Write(buf)
	os.Stdout.Write([]byte("\n"))
}

func (l Logger) Info(args ...interface{}) {
	if l.IsLoggingJSON {
		l.writeJSON("INFO", args)
	} else {
		l.writePretty("INFO", args)
	}
}

func (l Logger) Warn(args ...interface{}) {

}

func (l Logger) NewLine() {
	os.Stdout.Write([]byte("\n"))
}

func (l Logger) Spaces(num int32) {

}

func (l Logger) Tabs(num int32) {

}

var DefaultLogger = Logger{
	AppName:       "Default",
	IsLoggingJSON: !isTerminal,
	HostName:      "",
}

func init() {

}
