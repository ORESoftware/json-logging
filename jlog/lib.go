package json_logging

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/logrusorgru/aurora"
	"github.com/oresoftware/json-logging/jlog/stack"
	"github.com/oresoftware/json-logging/jlog/writer"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var isTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))
var pid = os.Getpid()

var m1 = sync.Mutex{}

var safeStdout = writer.NewSafeWriter(os.Stdout)
var safeStderr = writer.NewSafeWriter(os.Stderr)

var lockStack = stack.NewStack()

type Logger struct {
	AppName       string
	IsLoggingJSON bool
	HostName      string
	ForceJSON     bool
	ForceNonJSON  bool
	TimeZone      string
	MetaFields    *MetaFields
	LockUuid      string
}

type LoggerParams struct {
	AppName       string
	IsLoggingJSON bool
	HostName      string
	ForceJSON     bool
	ForceNonJSON  bool
	MetaFields    MetaFields
	TimeZone      string
	LockUuid      string
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

	var isLoggingJson = !isTerminal

	if os.Getenv("jlog_log_json") == "no" {
		isLoggingJson = false
	}

	return &Logger{
		//IsLoggingJSON: !isTerminal && !forceJSON,
		IsLoggingJSON: isLoggingJson,
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
	*metaFieldsMarker
}

type M = map[string]interface{}
type L = []KV

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

func NewMetaFields(m *map[string]interface{}) *MetaFields {
	return &MetaFields{
		metaFieldsMarker: mfMarker,
		UniqueMarker:     "UniqueMarker(Brand)",
		m:                m,
	}
}

func (l *Logger) Lock() *Logger {
	var newLck = &sync.Mutex{}
	newLck.Lock()
	var id = uuid.New()
	lockStack.Push(&stack.StackItem{
		Id:  id,
		Lck: newLck,
	})
	return &Logger{
		AppName:       l.AppName,
		IsLoggingJSON: l.IsLoggingJSON,
		HostName:      l.HostName,
		ForceJSON:     l.ForceJSON,
		ForceNonJSON:  l.ForceNonJSON,
		TimeZone:      l.TimeZone,
		MetaFields:    l.MetaFields,
		LockUuid:      id,
	}
}

func (l *Logger) Unlock() {

	m1.Lock()
	defer m1.Unlock()

	var peek, err = lockStack.Peek()

	if peek == nil {
		panic("error with lib - peek should not be nil")
		return
	}

	if err != nil {
		panic("error should be nil if peek item exists")
	}

	if peek.Id != l.LockUuid {
		panic("lock ids do not match")
	}

	lockStack.Pop()
	peek.Lck.Unlock()

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

func (l *Logger) writePretty(level string, m *MetaFields, args *[]interface{}) {

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

	defer m1.Unlock()
	m1.Lock()

	peekItem, err := lockStack.Peek()

	if err != nil && peekItem != nil {
		panic("library error.")
	}

	if peekItem != nil {

		if peekItem.Id != l.LockUuid {
			peekItem.Lck.Lock()
			defer func() {
				peekItem.Lck.Unlock()
			}()
		}
		//if peekItem.Id == l.LockUuid {
		//	defer func() {
		//		peekItem.Lck.Unlock()
		//		lockStack.Pop()
		//	}()
		//} else {
		//	peekItem.Lck.Lock()
		//	defer func() {
		//		peekItem.Lck.Unlock()
		//	}()
		//}
	}

	for _, v := range buf {
		if _, err := safeStdout.WriteString(v); err != nil {
			fmt.Println(err)
		}
	}

	size := 0

	var primitive = true

	for _, v := range *args {

		val := reflect.ValueOf(v)
		var kind = reflect.TypeOf(v).Kind()

		if kind == reflect.Ptr {
			//v = val.Elem().Interface()
			//val = reflect.ValueOf(v)
			val = val.Elem()
			if val.IsValid() { // Check if the dereferenced value is valid
				v = val.Interface()
				val = reflect.ValueOf(v)
				kind = val.Kind()
			}
		}

		if isNonPrimitive(kind) {
			primitive = false
		}

		s := getPrettyString(v, size) + " "
		i := strings.LastIndex(s, "\n")
		if i >= 0 {
			size = len(s) - i
		} else {
			size = size + len(s)
		}

		if _, err := safeStdout.Write([]byte(s)); err != nil {
			fmt.Fprintln(os.Stderr, "771c710b-aba2-46ef-9126-c26d3dfe7925", err)
		}

		if !primitive {

			if _, err := safeStdout.Write([]byte("\n")); err != nil {
				fmt.Fprintln(os.Stderr, "18614292-658f-42a5-81e7-593e941ea857", err)
			}

			safeStdout.WriteString("\n")
			zz := fmt.Sprintf("sprintf: %+v", v)
			if _, err := safeStdout.WriteString(zz); err != nil {
				fmt.Fprintln(os.Stderr, "2a795ef2-65bb-4a03-9808-b072e4497d73", err)
			}

			safeStdout.WriteString("\n")

			//safeStdout.Write([]byte("json:"))
			//if x, err := json.Marshal(v); err == nil {
			//	if _, err := safeStdout.Write([]byte(x)); err != nil {
			//		fmt.Println("err1:", err)
			//	}
			//} else {
			//	fmt.Println("err2:", err)
			//}

		}

	}

	if _, err := safeStdout.WriteString("\n"); err != nil {
		fmt.Fprintln(os.Stderr, "f834d14a-9735-4fd6-9389-f79144044746", err)
	}
}

func isNonPrimitive(kind reflect.Kind) bool {
	return kind == reflect.Slice ||
		kind == reflect.Array ||
		kind == reflect.Struct ||
		kind == reflect.Func ||
		kind == reflect.Map ||
		kind == reflect.Chan ||
		kind == reflect.Interface
}

func (l *Logger) writeJSONFromFormattedStr(level string, m *MetaFields, s *[]interface{}) {

	date := time.Now().UTC().String()
	date = date[:26]

	buf, err := json.Marshal([8]interface{}{"@bunion", l.AppName, level, pid, l.HostName, date, m, s})

	if err != nil {
		DefaultLogger.Warn(err)
	} else {
		safeStdout.Write(buf)
		safeStdout.Write([]byte("\n"))
	}

}

func (l *Logger) writeJSON(level string, m *MetaFields, args *[]interface{}) {

	date := time.Now().UTC().String()
	date = date[:26]

	buf, err := json.Marshal([8]interface{}{"@bunion", l.AppName, level, pid, l.HostName, date, m, args})

	if err != nil {

		_, file, line, _ := runtime.Caller(3)

		DefaultLogger.Warn("could not marshal the slice:", err.Error(), "file://"+file+":"+strconv.Itoa(line))

		//cleaned := make([]interface{},0)

		var cleaned = make([]interface{}, 0)

		for i := 0; i < len(*args); i++ {
			// TODO: for now instead of cleanUp, we can ust fmt.Sprintf()
			cleaned = append(cleaned, cleanUp((*args)[i]))
		}

		buf, err = json.Marshal([8]interface{}{"@bunion", l.AppName, level, pid, l.HostName, date, m, cleaned})

		if err != nil {
			fmt.Println(errors.New("Json-Logging: could not marshal the slice: " + err.Error()))
			return
		}
	}

	m1.Lock()
	safeStdout.Write(buf)
	safeStdout.Write([]byte("\n"))
	m1.Unlock()
}

func (l *Logger) writeSwitchForFormattedString(level string, m *MetaFields, s *[]interface{}) {
	if l.IsLoggingJSON {
		l.writeJSONFromFormattedStr(level, m, s)
	} else {
		l.writePretty(level, m, s)
	}
}

func (l *Logger) writeSwitch(level string, m *MetaFields, args *[]interface{}) {
	if l.IsLoggingJSON {
		l.writeJSON(level, m, args)
	} else {
		l.writePretty(level, m, args)
	}
}

func (l *Logger) PrintEnvPlain() {
	envVars := os.Environ() // Get all environment variables as a slice
	sort.Strings(envVars)
	for _, env := range envVars {
		log.Println(env)
	}
}

func (l *Logger) PrintEnv() {
	envVars := os.Environ() // Get all environment variables as a slice
	sort.Strings(envVars)
	for _, env := range envVars {
		l.Info(env)
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

type errorIdMarker struct{}

var eidMarker = &errorIdMarker{}

type ErrorId struct {
	Id            string
	errorIdMarker *errorIdMarker
}

type Opts struct {
	IsPrintStackTrace bool
	errorIdMarker     *errorIdMarker
}

func ErrId(id string) *ErrorId {
	return &ErrorId{
		id, eidMarker,
	}
}

func ErrOpts(id string) *ErrorId {
	return &ErrorId{
		id, eidMarker,
	}
}

// brand the below struct with unique ref
type metaFieldsMarker struct{}

var mfMarker = &metaFieldsMarker{}

type MetaFields struct {
	*metaFieldsMarker
	UniqueMarker string
	m            *map[string]interface{}
}

func MetaPairs(
	k1 string, v1 interface{},
	args ...interface{}) *MetaFields {
	return MP(k1, v1, args...)
}

func MP(
	k1 string, v1 interface{},
	args ...interface{}) *MetaFields {

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
	var filteredLines = []string{""}
	for _, line := range lines {
		if !strings.Contains(line, "oresoftware/json-logging") {
			filteredLines = append(filteredLines, fmt.Sprintf("\t%s", strings.TrimSpace(line)))
		}
	}

	return strings.Join(filteredLines, "\n")
}

func (l *Logger) TagPair(k string, v interface{}) *Logger {
	var z = map[string]interface{}{k: v}
	return l.Child(&z)
}

func (l *Logger) Tags(z *map[string]interface{}) *Logger {
	return l.Create(z)
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
	Trace string
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
	safeStdout.Write([]byte("\n"))
}

func (l *Logger) Spaces(num int32) {
	safeStdout.Write([]byte(strings.Join(make([]string, num), " ")))
}

func (l *Logger) Tabs(num int32) {
	safeStdout.Write([]byte(strings.Join(make([]string, num), "\t")))
}

func (l *Logger) PlainStdout(args ...interface{}) {
	safeStdout.Lock()
	for _, a := range args {
		v := fmt.Sprintf("((%T) %#v) ", a, a)
		os.Stdout.Write([]byte(v))
	}
	os.Stdout.Write([]byte("\n"))
	safeStdout.Unlock()
}

func (l *Logger) PlainStderr(args ...interface{}) {
	safeStderr.Lock()
	for _, a := range args {
		v := fmt.Sprintf("((%T) %#v) ", a, a)
		os.Stderr.Write([]byte(v))
	}
	os.Stderr.Write([]byte("\n"))
	safeStderr.Unlock()
}

var DefaultLogger = Logger{
	AppName:       "Default",
	IsLoggingJSON: !isTerminal,
	HostName:      os.Getenv("HOSTNAME"),
}

func init() {

	//log.SetFlags(log.LstdFlags | log.Llongfile)

}
