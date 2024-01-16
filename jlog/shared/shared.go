package shared

import (
	"fmt"
	"github.com/oresoftware/json-logging/jlog/pool"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"reflect"
	"strings"
	"sync"
)

type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG LogLevel = iota
	INFO  LogLevel = iota
	WARN  LogLevel = iota
	ERROR LogLevel = iota
)

var M1 = sync.Mutex{}

var IsTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))
var PID = os.Getpid()

var Level = map[string]LogLevel{
	"TRACE": TRACE,
	"DEBUG": DEBUG,
	"WARN":  WARN,
	"ERROR": ERROR,
	"INFO":  INFO,
	"":      TRACE,
}

func ToLogLevel(s string) LogLevel {
	var cleanVal = strings.ToUpper(strings.TrimSpace(s))
	if v, ok := Level[cleanVal]; ok {
		return v
	}
	fmt.Println(fmt.Sprintf("warning no log level could be retrieved via value: '%s'", s))
	return TRACE
}

var LevelToString = map[LogLevel]string{
	TRACE: "TRACE",
	DEBUG: "DEBUG",
	WARN:  "WARN",
	ERROR: "ERROR",
	INFO:  "INFO",
}

func IsNonPrimitive(kind reflect.Kind) bool {
	return kind == reflect.Slice ||
		kind == reflect.Array ||
		kind == reflect.Struct ||
		kind == reflect.Func ||
		kind == reflect.Map ||
		kind == reflect.Chan ||
		kind == reflect.Interface
}

var StdioPool = pool.CreatePool(30)
