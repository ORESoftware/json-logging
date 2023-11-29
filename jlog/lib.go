package json_logging

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/logrusorgru/aurora"
	"github.com/oresoftware/json-logging/jlog/writer"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var isTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))
var pid = os.Getpid()

var safeStdout = writer.NewSafeWriter(os.Stdout)
var safeStderr = writer.NewSafeWriter(os.Stderr)

type Logger struct {
	AppName       string
	IsLoggingJSON bool
	HostName      string
	ForceJSON     bool
	ForceNonJSON  bool
	TimeZone      string
	MetaFields    *metaFields
}

type LoggerParams struct {
	AppName       string
	IsLoggingJSON bool
	HostName      string
	ForceJSON     bool
	ForceNonJSON  bool
	MetaFields    metaFields
	TimeZone      string
}

func New(AppName string, forceJSON bool, hostName string) *Logger {

	if hostName == "" {

		hostName = os.Getenv("HOSTNAME")

		if hostName == "" {
			hn, err := os.Hostname()
			if err != nil {
				DefaultLogger.Warn("Could not grab hostname from env.")
				hostName = "unknown_hostname"
			} else {
				hostName = hn
			}
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
		MetaFields: NewMetaFields(
			&map[string]interface{}{},
		),
	}
}

type KV struct {
	Key   string
	Value interface{}
	*marker
}

type M = []KV

var kv = M{{},{}}

func doCopyAndDerefStruct(s interface{}) interface{} {
	val := reflect.ValueOf(s).Elem()
	newStruct := reflect.New(val.Type()).Elem()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		newField := newStruct.Field(i)
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			newField.Set(reflect.Indirect(field))
		} else {
			newField.Set(field)
		}
	}

	return newStruct.Interface()
}

func copyAndDereference(s interface{}) interface{} {

	// Checking the type of myArray

	var kind = reflect.TypeOf(s).Kind()

	if kind == reflect.Slice || kind == reflect.Array {
		val := reflect.ValueOf(s)
		n := val.Len()
		slice := make([]interface{}, n)
		for i := 0; i < n; i++ {
			slice[i] = doCopyAndDerefStruct(val.Index(i).Interface())
		}
		return slice
	}

	// Checking the type of myStruct
	if kind == reflect.Struct {
		return doCopyAndDerefStruct(s)
	}

	return s

}

func NewMetaFields(m *map[string]interface{}) *metaFields {
	return &metaFields{
		marker:       mark,
		UniqueMarker: "UniqueMarker(Brand)",
		m:            m,
	}
}

func (l *Logger) Child(m *map[string]interface{}) *Logger {

	var z = make(map[string]interface{})
	for k, v := range *l.MetaFields.m {
		z[k] = copyAndDereference(v)
	}

	for k, v := range *m {
		z[k] = copyAndDereference(v)
	}

	return &Logger{
		IsLoggingJSON: l.IsLoggingJSON,
		AppName:       l.AppName,
		HostName:      l.HostName,
		MetaFields:    NewMetaFields(&z),
	}
}

func (l *Logger) Create(m *map[string]interface{}) *Logger {

	var z = make(map[string]interface{})
	for k, v := range *m {
		z[k] = copyAndDereference(v)
	}

	return &Logger{
		IsLoggingJSON: l.IsLoggingJSON,
		AppName:       l.AppName,
		HostName:      l.HostName,
		MetaFields:    NewMetaFields(&z),
	}
}

func (l *Logger) writePretty(level string, m *metaFields, args *[]interface{}) {

	date := time.Now().UTC().String()[11:25] // only first 25 chars

	stylizedLevel := level

	switch level {
	case "ERROR":
		stylizedLevel = aurora.Red(level).String()
		break

	case "WARN":
		stylizedLevel = aurora.Magenta(level).String()
		break

	case "DEBUG":
		stylizedLevel = aurora.Bold(level).String()
		break

	case "INFO":
		stylizedLevel = aurora.Gray(12, level).String()
		break

	case "TRACE":
		stylizedLevel = aurora.Gray(4, level).String()
		break
	}

	buf := []string{
		aurora.Gray(9, date).String(), " ",
		stylizedLevel, " ",
		aurora.Gray(12, "app:").String() + aurora.Italic(l.AppName).String(), " ",
	}

	defer safeStdout.Unlock()
	safeStdout.Lock()

	for _, v := range buf {
		if _, err := safeStdout.Write([]byte(v)); err != nil {
			fmt.Println(err)
		}
	}

	size := 0
	for _, v := range *args {

		s := getPrettyString(v, size) + " "
		i := strings.LastIndex(s, "\n")
		if i >= 0 {
			size = len(s) - i
		} else {
			size = size + len(s)
		}

		if _, err := safeStdout.Write([]byte(s)); err != nil {
			fmt.Println(err)
		}
	}

	if _, err := safeStdout.Write([]byte("\n")); err != nil {
		fmt.Println(err)
	}
}

func (l *Logger) writeJSONFromFormattedStr(level string, m *metaFields, s *[]interface{}) {

	date := time.Now().UTC().String()
	date = date[:26]

	buf, err := json.Marshal([8]interface{}{"@bunion", l.AppName, level, pid, l.HostName, date, m, s})

	if err != nil {
		DefaultLogger.Warn(err)
	} else {
		os.Stdout.Write(buf)
		os.Stdout.Write([]byte("\n"))
	}

}

func (l *Logger) writeJSON(level string, m *metaFields, args *[]interface{}) {

	date := time.Now().UTC().String()
	date = date[:26]

	buf, err := json.Marshal([8]interface{}{"@bunion", l.AppName, level, pid, l.HostName, date, m, args})

	if err != nil {

		_, file, line, _ := runtime.Caller(3)

		DefaultLogger.Warn("could not marshal the slice:", err.Error(),
			"file://"+file+":"+strconv.Itoa(line))

		//cleaned := make([]interface{},0)

		var cleaned = make([]interface{}, 0)

		for i := 0; i < len(*args); i++ {
			cleaned = append(cleaned, cleanUp((*args)[i]))
		}

		buf, err = json.Marshal([8]interface{}{"@bunion", l.AppName, level, pid, l.HostName, date, m, cleaned})

		if err != nil {
			panic(errors.New("could not marshal the slice: " + err.Error()))
		}
	}

	os.Stdout.Write(buf)
	os.Stdout.Write([]byte("\n"))
}

func (l *Logger) writeSwitchForFormattedString(level string, m *metaFields, s *[]interface{}) {
	if l.IsLoggingJSON {
		l.writeJSONFromFormattedStr(level, m, s)
	} else {
		l.writePretty(level, m, &[]interface{}{s})
	}
}

func (l *Logger) writeSwitch(level string, m *metaFields, args *[]interface{}) {
	if l.IsLoggingJSON {
		l.writeJSON(level, m, args)
	} else {
		l.writePretty(level, m, args)
	}
}

func (l *Logger) JSON(args ...interface{}) {
	size := len(args)
	for i := 0; i < size; i++ {

		v, err := json.Marshal(args[i])

		if err != nil {
			panic(err)
		}

		os.Stdout.Write(v)
		if i < size-1 {
			os.Stdout.Write([]byte(" "))
		}
	}
	os.Stdout.Write([]byte("\n"))
}

func (l *Logger) RawJSON(args ...interface{}) {
	// raw = no newlines, no spaces
	for i := 0; i < len(args); i++ {

		v, err := json.Marshal(args[i])

		if err != nil {
			panic(err)
		}

		os.Stdout.Write(v)
	}
}

func (l *Logger) Info(args ...interface{}) {
	l.writeSwitch("INFO", nil, &args)
}

func (l *Logger) Warn(args ...interface{}) {
	l.writeSwitch("WARN", nil, &args)
}

func (l *Logger) Warning(args ...interface{}) {
	l.writeSwitch("WARN", nil, &args)
}

func (l *Logger) Error(args ...interface{}) {
	filteredStackTrace := getFilteredStacktrace()
	args = append(args, StackTrace{filteredStackTrace})
	l.writeSwitch("ERROR", nil, &args)
}

func (l *Logger) Debug(args ...interface{}) {
	l.writeSwitch("DEBUG", nil, &args)
}

func (l *Logger) Trace(args ...interface{}) {
	l.writeSwitch("TRACE", nil, &args)
}

// brand the below struct with unique ref
type marker struct{}

var mark = &marker{}

type metaFields struct {
	*marker
	UniqueMarker string
	m            *map[string]interface{}
}

func MetaPairs(
	k1 string, v1 interface{},
	args ...interface{}) *metaFields {
	return MP(k1, v1, args...)
}

func MP(
	k1 string, v1 interface{},
	args ...interface{}) *metaFields {

	m := make(map[string]interface{})
	nargs := append([]interface{}{k1, v1}, args...) // prepend the first two arguments to new slice

	currKey := ""

	for i, a := range nargs {

		if i%2 == 0 {
			// operate on keys
			v, ok := a.(string)
			if ok {
				currKey = v
			} else {
				panic("even arguments must be strings, odd arguments are interface{}")
			}
			if len(nargs) < i+2 {
				panic("a key needs a respective value.")
			}
			continue
		}

		// operate on values
		m[currKey] = a
	}

	return NewMetaFields(&m)

	//return metaFields{
	//	Meta: m,
	//}
}

func getFilteredStacktrace() string {
	// Capture the stack trace
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	stackTrace := string(buf[:n])

	// Filter the stack trace
	lines := strings.Split(stackTrace, "\n")
	var filteredLines []string
	for _, line := range lines {
		if !strings.Contains(line, "github.com/oresoftware") {
			filteredLines = append(filteredLines, line)
		}
	}

	return strings.Join(filteredLines, "\n")
}

func (l *Logger) TagPair(k string, v interface{}) *Logger {

	return &l.Create(NewMetaFields(
		   *map[string]interface{}{
			   k, v
		   }
		))
}

func (l *Logger) Tags() *Logger {

	return &l.Create()
}

func (l *Logger) InfoF(s string, args ...interface{}) {
	l.writeSwitchForFormattedString("INFO", nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) WarnF(s string, args ...interface{}) {
	l.writeSwitchForFormattedString("WARN", nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) WarningF(s string, args ...interface{}) {
	l.writeSwitchForFormattedString("WARN", nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

type StackTrace struct {
	StackTrace string
}

func (l *Logger) ErrorF(s string, args ...interface{}) {
	filteredStackTrace := getFilteredStacktrace()
	formattedString := fmt.Sprintf(s, args...)
	l.writeSwitchForFormattedString("ERROR", nil, &[]interface{}{formattedString, StackTrace{filteredStackTrace}})
}

func (l *Logger) DebugF(s string, args ...interface{}) {
	l.writeSwitchForFormattedString("DEBUG", nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) TraceF(s string, args ...interface{}) {
	l.writeSwitchForFormattedString("TRACE", nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) NewLine() {
	os.Stdout.Write([]byte("\n"))
}

func (l *Logger) Spaces(num int32) {
	os.Stdout.Write([]byte(strings.Join(make([]string, num), " ")))
}

func (l *Logger) Tabs(num int32) {
	os.Stdout.Write([]byte(strings.Join(make([]string, num), "\t")))
}

func (l *Logger) PlainStdout(args ...interface{}) {
	for _, a := range args {
		v := fmt.Sprintf("((%T) %#v) ", a, a)
		os.Stdout.Write([]byte(v))
	}
	os.Stdout.Write([]byte("\n"))
}

func (l *Logger) PlainStderr(args ...interface{}) {
	for _, a := range args {
		v := fmt.Sprintf("((%T) %#v) ", a, a)
		os.Stderr.Write([]byte(v))
	}
	os.Stderr.Write([]byte("\n"))
}

var DefaultLogger = Logger{
	AppName:       "Default",
	IsLoggingJSON: !isTerminal,
	HostName:      os.Getenv("HOSTNAME"),
}

func init() {

	//log.SetFlags(log.LstdFlags | log.Llongfile)

}
