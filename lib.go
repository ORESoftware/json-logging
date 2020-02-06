package json_logging

import (
	"encoding/json"
	"os"
	"golang.org/x/crypto/ssh/terminal"
	)


var isTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))


type jsons struct {
	Time int32
}

type Logger struct {
   IsLoggingJSON bool
}


func New() Logger{
   return Logger{
     IsLoggingJSON: !isTerminal,
   }
}

func (l Logger) write(args []interface{}) {

	buf := json.Marshal()
	os.Stdout.Write(buf)
}


func (l Logger) Info(args... interface{}) {
   l.write(args)
}

func (l Logger) Warn() {

}

var DefaultLogger = New()

func init()   {

}