package shared

import (
	"fmt"
	ll "github.com/oresoftware/json-logging/jlog/level"
	"github.com/oresoftware/json-logging/jlog/pool"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"reflect"
	"strings"
	"sync"
)

var M1 = sync.Mutex{}

var IsTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))
var PID = os.Getpid()

var Level = map[string]ll.LogLevel{
	"TRACE": ll.TRACE,
	"DEBUG": ll.DEBUG,
	"WARN":  ll.WARN,
	"ERROR": ll.ERROR,
	"INFO":  ll.INFO,
	"":      ll.TRACE,
}

func ToLogLevel(s string) ll.LogLevel {
	var cleanVal = strings.ToUpper(strings.TrimSpace(s))
	if v, ok := Level[cleanVal]; ok {
		return v
	}
	fmt.Println(fmt.Sprintf("warning no log level could be retrieved via value: '%s'", s))
	return ll.TRACE
}

var LevelToString = map[ll.LogLevel]string{
	ll.TRACE: "TRACE",
	ll.DEBUG: "DEBUG",
	ll.WARN:  "WARN",
	ll.ERROR: "ERROR",
	ll.INFO:  "INFO",
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
