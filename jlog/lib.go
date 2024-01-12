package json_logging

import (
	"encoding/json"
	"errors"
	"fmt"
	uuid "github.com/google/uuid"
	"github.com/logrusorgru/aurora"
	"github.com/oresoftware/json-logging/jlog/stack"
	"github.com/oresoftware/json-logging/jlog/writer"
	"golang.org/x/crypto/ssh/terminal"
	"log"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type LogLevel int

const (
	TRACE LogLevel = iota
	DEBUG LogLevel = iota
	INFO  LogLevel = iota
	WARN  LogLevel = iota
	ERROR LogLevel = iota
)

var isTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))
var pid = os.Getpid()

func WriteToStderr(args ...interface{}) {
	if _, err := fmt.Fprintln(os.Stderr, args...); err != nil {
		fmt.Println("adcca45f-8d7b-4d4a-8fd2-7683b7b375b5", "could not write to stderr:", err)
	}
}

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
	EnvPrefix     string
	LogLevel      LogLevel
	Files         []*os.File
}

type LoggerParams struct {
	AppName       string
	IsLoggingJSON bool
	HostName      string
	ForceJSON     bool
	ForceNonJSON  bool
	MetaFields    *MetaFields
	TimeZone      string
	LockUuid      string
	EnvPrefix     string
	LogLevel      LogLevel
	Files         []*os.File
}

func NewLogger(p LoggerParams) *Logger {

	var files = []*os.File{}

	if p.Files != nil {
		files = p.Files[:]
	}

	if len(files) < 1 {
		files = append(files, os.Stdout)
	}

	hostName := p.HostName

	if hostName == "" {

		hostName = os.Getenv("HOSTNAME")
		if hostName == "" {
			hn, err := os.Hostname()
			if err != nil {
				hostName = "<unknown_hostname>"
			} else {
				hostName = hn
			}
		}
	}

	var isLoggingJson = !isTerminal

	if p.ForceJSON {
		isLoggingJson = true
	}

	if os.Getenv("jlog_log_json") == "no" {
		isLoggingJson = false
	}

	if os.Getenv("jlog_log_json") == "yes" {
		if p.ForceJSON {
			WriteToStderr("forceJSON:true was used, but the 'jlog_log_json' env var was set to 'yes'.")
		}
		isLoggingJson = true
	}

	var metaFields = MP(
		"git_commit", os.Getenv("jlog_git_commit"),
		"git_repo", os.Getenv("jlog_git_repo"),
	)

	if len(p.EnvPrefix) > 0 {
		for _, env := range os.Environ() {
			parts := strings.SplitN(env, "=", 2)
			key := parts[0]
			value := parts[1]
			if strings.HasPrefix(key, p.EnvPrefix) {
				result := strings.TrimPrefix(key, p.EnvPrefix)
				(*metaFields.m)[result] = value
			}
		}
	}

	if p.MetaFields != nil && p.MetaFields.m != nil {
		for k, v := range *p.MetaFields.m {
			(*metaFields.m)[k] = v
		}
	}

	var appName = "<app>"
	if p.AppName != "" {
		appName = p.AppName
	}

	return &Logger{
		AppName:       appName,
		IsLoggingJSON: isLoggingJson,
		HostName:      hostName,
		ForceJSON:     p.ForceJSON,
		ForceNonJSON:  p.ForceNonJSON,
		TimeZone:      p.TimeZone,
		MetaFields:    metaFields,
		LockUuid:      p.LockUuid,
		EnvPrefix:     p.EnvPrefix,
		LogLevel:      p.LogLevel,
		Files:         files,
	}
}

func New(AppName string, forceJSON bool, hostName string, envTokenPrefix string, level LogLevel, files []*os.File) *Logger {
	return NewLogger(LoggerParams{
		AppName:   AppName,
		ForceJSON: forceJSON,
		HostName:  hostName,
		EnvPrefix: envTokenPrefix,
		Files:     files,
	})
}

//func NewLogger(AppName string, forceJSON bool, hostName string, envTokenPrefix string) *Logger {
//	return New(AppName, forceJSON, hostName, envTokenPrefix)
//}

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

func NewMetaFields(m *MF) *MetaFields {
	return &MetaFields{
		metaFieldsMarker: mfMarker,
		UniqueMarker:     "UniqueMarker(Brand)",
		m:                m,
	}
}

type logIdMarker struct{}

var myLogIdMarker = &logIdMarker{}

type LogId struct {
	*logIdMarker
	val string
}

func (x *LogId) IsLogId() bool {
	return true
}

func Id(v string) *LogId {
	return &LogId{myLogIdMarker, v}
}

func (l *Logger) Id(v string) *LogId {
	return Id(v)
}

func (l *Logger) NewLoggerWithLock() (*Logger, func()) {
	m1.Lock()
	defer m1.Unlock()
	var newLck = &sync.Mutex{}
	newLck.Lock()
	var id = uuid.New().String()
	lockStack.Push(&stack.StackItem{
		Id:  id,
		Lck: newLck,
	})
	var z = Logger{
		AppName:       l.AppName,
		IsLoggingJSON: l.IsLoggingJSON,
		HostName:      l.HostName,
		ForceJSON:     l.ForceJSON,
		ForceNonJSON:  l.ForceNonJSON,
		TimeZone:      l.TimeZone,
		MetaFields:    l.MetaFields,
		LockUuid:      id,
	}
	return &z, z.unlock
}

func (l *Logger) unlock() {

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

	lockStack.Print("123")
	x, err := lockStack.Pop()
	if x != peek {
		panic("must equal peek")
	}
	fmt.Println("unlocking:", peek.Id)
	lockStack.Print("456")
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

type SprintFStruct struct {
	SprintF string
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
		stylizedLevel = aurora.Underline(aurora.Bold(aurora.Red(level))).String()
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

	m1.Lock()
	peekItem, err := lockStack.Peek()

	if err != nil && peekItem != nil {
		panic("library error.")
	}

	var doUnlock = false
	if peekItem != nil {

		if peekItem.Id != l.LockUuid {
			//fmt.Println(fmt.Sprintf("%+v", lockStack))
			doUnlock = true
			m1.Unlock()
			lockStack.Print("789")
			fmt.Println("here 1", peekItem.Id)
			peekItem.Lck.Lock()
			fmt.Println("here 2")
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

	if !doUnlock {
		m1.Unlock()
	}

	defer m1.Unlock()
	m1.Lock()

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

		if _, err := safeStdout.WriteString(s); err != nil {
			WriteToStderr("771c710b-aba2-46ef-9126-c26d3dfe7925", err)
		}

		if !primitive && (level == "TRACE" || level == "DEBUG") {

			if _, err := safeStdout.WriteString("\n"); err != nil {
				WriteToStderr("18614292-658f-42a5-81e7-593e941ea857", err)
			}

			if _, err := safeStdout.WriteString(fmt.Sprintf("sprintf: %+v", v)); err != nil {
				WriteToStderr("2a795ef2-65bb-4a03-9808-b072e4497d73", err)
			}

			safeStdout.Write([]byte("json:"))
			if x, err := json.Marshal(v); err == nil {
				if _, err := safeStdout.Write(x); err != nil {
					WriteToStderr("err:56831878-8d63-45f4-905b-d1b3bbac2152:", err)
				}
			} else {
				WriteToStderr("err:70bf10e0-6e69-4a3b-bf64-08f6d20c4580:", err)
			}

		}

	}

	if _, err := safeStdout.WriteString("\n"); err != nil {
		WriteToStderr("f834d14a-9735-4fd6-9389-f79144044746", err)
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

	buf, err := json.Marshal([8]interface{}{"@bunion:v1", l.AppName, level, pid, l.HostName, date, m, s})

	if err != nil {
		DefaultLogger.Warn(err)
	} else {
		safeStdout.Write(buf)
		safeStdout.Write([]byte("\n"))
	}

}

func (l *Logger) writeJSON(level string, mf *MetaFields, args *[]interface{}) {

	date := time.Now().UTC().String()
	date = date[:26]

	buf, err := json.Marshal([8]interface{}{"@bunion:v1", l.AppName, level, pid, l.HostName, date, mf.m, args})

	if err != nil {

		_, file, line, _ := runtime.Caller(3)

		DefaultLogger.Warn("could not marshal the slice:", err.Error(), "file://"+file+":"+strconv.Itoa(line))

		//cleaned := make([]interface{},0)

		var cache = map[*interface{}]*interface{}{}
		var cleaned = make([]interface{}, 0)

		for i := 0; i < len(*args); i++ {
			// TODO: for now instead of cleanUp, we can ust fmt.Sprintf()
			v := &(*args)[i]
			c := cleanUp(v, &cache)
			debug.PrintStack()
			cleaned = append(cleaned, c)
		}

		buf, err = json.Marshal([8]interface{}{"@bunion:v1", l.AppName, level, pid, l.HostName, date, mf.m, cleaned})

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

func (l *Logger) getMetaFields(args *[]interface{}) (*MetaFields, []interface{}) {
	var newArgs = []interface{}{}
	var m = MF{}
	var mf = NewMetaFields(&m)

	for k, v := range *l.MetaFields.m {
		m[k] = v
	}

	for _, x := range *args {
		if z, ok := x.(MetaFields); ok {
			for k, v := range *z.m {
				m[k] = v
			}
		} else if z, ok := x.(*MetaFields); ok {
			for k, v := range *z.m {
				m[k] = v
			}
		} else if z, ok := x.(*LogId); ok {
			m["LogId"] = z.val
			newArgs = append(newArgs, z.val)
		} else if z, ok := x.(LogId); ok {
			m["LogId"] = z.val
			newArgs = append(newArgs, z.val)
		} else {
			newArgs = append(newArgs, x)
		}
	}

	return mf, newArgs
}

func (l *Logger) Info(args ...interface{}) {
	switch l.LogLevel {
	case WARN, ERROR:
		return
	}
	var meta, newArgs = l.getMetaFields(&args)
	l.writeSwitch("INFO", meta, &newArgs)
}

func (l *Logger) Warn(args ...interface{}) {
	switch l.LogLevel {
	case ERROR:
		return
	}
	var meta, newArgs = l.getMetaFields(&args)
	l.writeSwitch("WARN", meta, &newArgs)
}

func (l *Logger) Warning(args ...interface{}) {
	switch l.LogLevel {
	case ERROR:
		return
	}
	var meta, newArgs = l.getMetaFields(&args)
	l.writeSwitch("WARN", meta, &newArgs)
}

func (l *Logger) Error(args ...interface{}) {
	var meta, newArgs = l.getMetaFields(&args)
	filteredStackTrace := getFilteredStacktrace()
	newArgs = append(newArgs, StackTrace{filteredStackTrace})
	l.writeSwitch("ERROR", meta, &newArgs)
}

func (l *Logger) Debug(args ...interface{}) {
	switch l.LogLevel {
	case INFO, WARN, ERROR:
		return
	}
	var meta, newArgs = l.getMetaFields(&args)
	l.writeSwitch("DEBUG", meta, &newArgs)
}

func (l *Logger) Trace(args ...interface{}) {
	switch l.LogLevel {
	case DEBUG, INFO, WARN, ERROR:
		return
	}
	var meta, newArgs = l.getMetaFields(&args)
	l.writeSwitch("TRACE", meta, &newArgs)
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

type MF = map[string]interface{}

// brand the below struct with unique ref
type metaFieldsMarker struct{}

var mfMarker = &metaFieldsMarker{}

type MetaFields struct {
	*metaFieldsMarker
	UniqueMarker string
	m            *MF
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

}

func getFilteredStacktrace() *[]string {
	// Capture the stack trace
	buf := make([]byte, 1024)
	n := runtime.Stack(buf, false)
	stackTrace := string(buf[:n])

	// Filter the stack trace
	lines := strings.Split(stackTrace, "\n")
	var filteredLines = []string{}
	for _, line := range lines {
		if !strings.Contains(line, "oresoftware/json-logging") {
			filteredLines = append(filteredLines, fmt.Sprintf("%s", strings.TrimSpace(line)))
		}
	}

	return &filteredLines
}

func (l *Logger) TagPair(k string, v interface{}) *Logger {
	var z = map[string]interface{}{k: v}
	return l.Child(&z)
}

func (l *Logger) Tags(z *map[string]interface{}) *Logger {
	return l.Create(z)
}

func (l *Logger) InfoF(s string, args ...interface{}) {
	switch l.LogLevel {
	case WARN, ERROR:
		return
	}
	l.writeSwitchForFormattedString("INFO", nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) WarnF(s string, args ...interface{}) {
	switch l.LogLevel {
	case ERROR:
		return
	}
	l.writeSwitchForFormattedString("WARN", nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) WarningF(s string, args ...interface{}) {
	switch l.LogLevel {
	case ERROR:
		return
	}
	l.writeSwitchForFormattedString("WARN", nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

type StackTrace struct {
	ErrorTrace *[]string
}

func (l *Logger) ErrorF(s string, args ...interface{}) {
	filteredStackTrace := getFilteredStacktrace()
	formattedString := fmt.Sprintf(s, args...)
	l.writeSwitchForFormattedString("ERROR", nil, &[]interface{}{formattedString, StackTrace{filteredStackTrace}})
}

func (l *Logger) DebugF(s string, args ...interface{}) {
	switch l.LogLevel {
	case INFO, WARN, ERROR:
		return
	}
	l.writeSwitchForFormattedString("DEBUG", nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) TraceF(s string, args ...interface{}) {
	switch l.LogLevel {
	case DEBUG, INFO, WARN, ERROR:
		return
	}
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

var DefaultLogger = New(
	"Default",
	true,
	"<hostname>",
	"",
	TRACE,
	[]*os.File{os.Stdout},
)

func init() {

	//log.SetFlags(log.LstdFlags | log.Llongfile)

}
