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
}

func New(AppName string) Logger {
	return Logger{
		IsLoggingJSON: !isTerminal,
		AppName:       AppName,
	}
}

func (l Logger) writePretty(args []interface{}) {

	date := time.Now().String()
	buf, err := json.Marshal([2]string{l.AppName, date})

	if err != nil {
		panic(errors.New("could not marshal the string array"))
	}

	os.Stdout.Write(buf)
	os.Stdout.Write([]byte("\n"))
}

func (l Logger) writeJSON(args []interface{}) {

	date := time.Now().String()
	buf, err := json.Marshal([2]string{l.AppName, date})

	if err != nil {
		panic(errors.New("could not marshal the string array"))
	}

	os.Stdout.Write(buf)
	os.Stdout.Write([]byte("\n"))
}

func (l Logger) Info(args ...interface{}) {
	if l.IsLoggingJSON {
		l.writeJSON(args)
	} else {
		l.writePretty(args)
	}
}

func (l Logger) Warn() {

}

var DefaultLogger = New("AppName")

func init() {

}
