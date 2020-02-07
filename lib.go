package json_logging

import (
	"encoding/json"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"reflect"
	"strings"
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

type loggingTypeInternal struct {
	JSON string
	Pretty string
}

var LoggingType = loggingTypeInternal{
	JSON:   "JSON",
	Pretty: "Pretty",
}

var loggingTypeMap = make(map[string]string)

type LoggerParams struct {
	AppName       string
	IsLoggingJSON bool
	HostName      string
	ForceJSON     bool
	ForceNonJSON  bool
	MetaFields  MetaFields
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

func NewLogger(AppName string, forceJSON bool, hostName string) *Logger {

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

func (l Logger) Create(m *MetaFields)  *Logger {
	return &Logger{
		IsLoggingJSON: l.IsLoggingJSON,
		AppName:       l.AppName,
		HostName:      l.HostName,
	}
}

func (l Logger) writePretty(level string, m *MetaFields, args *[]interface{}) {

	date := time.Now().String()
	buf, err := json.Marshal([2]string{l.AppName, date})

	if err != nil {
		panic(errors.New("could not marshal the string array"))
	}

	os.Stdout.Write(buf)
	os.Stdout.Write([]byte("\n"))
}

func (l Logger) writeJSON(level string, m *MetaFields, args *[]interface{}) {

	date := time.Now().String()
	buf, err := json.Marshal([5]interface{}{l.AppName, level, date, m, args})

	if err != nil {
		panic(errors.New("could not marshal the string array"))
	}

	os.Stdout.Write(buf)
	os.Stdout.Write([]byte("\n"))
}

func (l Logger) writeSwitch(level string, m *MetaFields, args *[]interface{}) {
	if l.IsLoggingJSON {
		l.writeJSON(level, m, args)
	} else {
		l.writePretty(level, m, args)
	}
}

func (l Logger) Info(args ...interface{}) {
	l.writeSwitch("INFO", nil, &args)
}

func (l Logger) Warn(args ...interface{}) {
	l.writeSwitch("WARN", nil, &args)
}

func (l Logger) Error(args ...interface{}) {
	l.writeSwitch("ERROR", nil, &args)
}

func (l Logger) Fatal(args ...interface{}) {
	l.writeSwitch("FATAL", nil, &args)
}

func (l Logger) Debug(args ...interface{}) {
	l.writeSwitch("DEBUG", nil, &args)
}

func (l Logger) Trace(args ...interface{}) {
	l.writeSwitch("TRACE", nil, &args)
}

type MetaFields struct {
	Meta map[string]interface{}
}

func Meta(m map[string]interface{}) MetaFields {
	return MetaFields{
		Meta: m,
	}
}

func MetaPairs(
	k1 string, v1 interface{},
	args ...interface{}) MetaFields {

	m := make(map[string]interface{});
	nargs := append([]interface{}{k1, v1}, args...)

	currKey := ""

	for i, a := range nargs {

		if i % 2 == 0 {
			// operate on keys
			v, ok := a.(string)
			if ok {
				currKey = v
			} else{
				panic("even arguments must be strings, odd arguments are interface{}")
			}
			if nargs[i + 1] == nil {
                 panic("a key needs a respective value.")
			}
		  continue
		}

		// operate on values
		m[currKey] = a
	}

	return MetaFields{
		Meta: m,
	}
}

func (l Logger) InfoWithMeta(m MetaFields, args ...interface{}) {
	l.writeSwitch("INFO", &m, &args)
}

func (l Logger) WarnWithMeta(m MetaFields, args ...interface{}) {
	l.writeSwitch("WARN", &m, &args)
}

func (l Logger) ErrorWithMeta(m MetaFields, args ...interface{}) {
	l.writeSwitch("ERROR", &m, &args)
}

func (l Logger) FatalWithMeta(m MetaFields, args ...interface{}) {
	l.writeSwitch("FATAL", &m, &args)
}

func (l Logger) DebugWithMeta(m MetaFields, args ...interface{}) {
	l.writeSwitch("DEBUG", &m, &args)
}

func (l Logger) TraceWithMeta(m MetaFields, args ...interface{}) {
	l.writeSwitch("TRACE", &m, &args)
}

func (l Logger) NewLine() {
	os.Stdout.Write([]byte("\n"))
}

func (l Logger) Spaces(num int32) {
	os.Stdout.Write([]byte(strings.Join(make([]string, num), " ")))
}

func (l Logger) Tabs(num int32) {
	os.Stdout.Write([]byte(strings.Join(make([]string, num), "\t")))
}

var DefaultLogger = Logger{
	AppName:       "Default",
	IsLoggingJSON: !isTerminal,
	HostName:      "",
}

func init() {

	v := reflect.ValueOf(LoggingType)
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		loggingTypeMap[t.Field(i).Name] = v.Field(i).Interface().(string)
	}

	fmt.Println(loggingTypeMap)
}
