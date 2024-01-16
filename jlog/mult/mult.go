package mult

import (
	"encoding/json"
	"errors"
	"fmt"
	uuid "github.com/google/uuid"
	"github.com/logrusorgru/aurora"
	hlpr "github.com/oresoftware/json-logging/jlog/helper"
	"github.com/oresoftware/json-logging/jlog/shared"
	"github.com/oresoftware/json-logging/jlog/stack"
	"github.com/oresoftware/json-logging/jlog/writer"
	"log"
	"os"
	"reflect"
	"runtime"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var safeStdout = writer.NewSafeWriter(os.Stdout)
var safeStderr = writer.NewSafeWriter(os.Stderr)

var lockStack = stack.NewStack()

type FileLevel struct {
	Level   shared.LogLevel
	File    *os.File
	Tags    *map[string]interface{}
	lock    *sync.Mutex
	IsJSON  bool
	isTrace bool
	isDebug bool
	isInfo  bool
	isWarn  bool
	isError bool
}

type MultLogger struct {
	AppName    string
	HostName   string
	TimeZone   string
	MetaFields *MetaFields
	LockUuid   string
	EnvPrefix  string
	Files      []*FileLevel
	isTrace    bool
	isDebug    bool
	isInfo     bool
	isWarn     bool
	isError    bool
}

type MultLoggerParams struct {
	AppName    string
	HostName   string
	MetaFields *MetaFields
	TimeZone   string
	LockUuid   string
	EnvPrefix  string
	Files      []*FileLevel
}

func mapFileLevels(x []*FileLevel) []*FileLevel {

	var results = []*FileLevel{}
	var m = map[uintptr]*sync.Mutex{}

	for _, z := range x {

		if _, ok := m[z.File.Fd()]; !ok {
			m[z.File.Fd()] = &sync.Mutex{}
		}

		x, _ := m[z.File.Fd()]

		z := &FileLevel{
			Level: z.Level,
			File:  z.File,
			Tags:  z.Tags,
			lock:  x,
		}
		results = append(results, z)
	}

	return results
}

func NewLogger(p MultLoggerParams) *MultLogger {

	var files = []*FileLevel{}

	if p.Files != nil {
		files = mapFileLevels(p.Files)
	}

	if len(files) < 1 {
		files = append(files, &FileLevel{
			Level: shared.TRACE,
			File:  os.Stdout,
			Tags:  nil,
			lock:  nil,
		})
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

	var metaFields = NewMetaFields(&MF{})

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

	return &MultLogger{
		AppName:    appName,
		HostName:   hostName,
		TimeZone:   p.TimeZone,
		MetaFields: metaFields,
		LockUuid:   p.LockUuid,
		EnvPrefix:  p.EnvPrefix,
		Files:      files,
	}
}

func (l *MultLogger) determineInitialLogLevels() {
	///
	l.isTrace = false
	l.isDebug = false
	l.isInfo = false
	l.isWarn = false
	l.isError = true

	if len(l.Files) < 1 {
		l.Files = append(l.Files, &FileLevel{
			Level:   shared.TRACE,
			File:    os.Stdout,
			isTrace: true,
			isDebug: true,
			isInfo:  true,
			isWarn:  true,
			isError: true,
		})

		return
	}

	for _, v := range l.Files {

		switch v.Level {
		case shared.WARN:
			v.isWarn = true
			l.isWarn = true
		case shared.INFO:
			v.isInfo = true
			l.isInfo = true
		case shared.DEBUG:
			v.isDebug = true
			l.isDebug = true
		case shared.TRACE:
			v.isTrace = true
			l.isTrace = true
		default:
			v.isTrace = true
			l.isTrace = true
		}

	}
}

func isSameFile(fd1 uintptr, fd2 uintptr) (bool, error) {
	var stat1, stat2 syscall.Stat_t
	if err := syscall.Fstat(int(fd1), &stat1); err != nil {
		return false, err
	}
	if err := syscall.Fstat(int(fd2), &stat2); err != nil {
		return false, err
	}
	return stat1.Dev == stat2.Dev && stat1.Ino == stat2.Ino, nil
}

func checkIfSameFile() {
	same, err := isSameFile(os.Stdout.Fd(), os.Stderr.Fd())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error checking file descriptors: %v\n", err)
		os.Exit(1)
	}
	if same {
		fmt.Println("os.Stdout and os.Stderr are directed to the same file/terminal")
	} else {
		fmt.Println("os.Stdout and os.Stderr are directed to different files/terminals")
	}
}

func NewBasicLogger(AppName string, envTokenPrefix string, files ...*FileLevel) *MultLogger {
	return NewLogger(MultLoggerParams{
		AppName:   AppName,
		EnvPrefix: envTokenPrefix,
		Files:     files,
	})
}

func New(AppName string, envTokenPrefix string, files []*FileLevel) *MultLogger {
	return NewLogger(MultLoggerParams{
		AppName:   AppName,
		EnvPrefix: envTokenPrefix,
		Files:     files,
	})
}

//func NewLogger(AppName string, forceJSON bool, hostName string, envTokenPrefix string) *MultLogger {
//	return New(AppName, forceJSON, hostName, envTokenPrefix)
//}

type KV struct {
	Key   string
	Value interface{}
	*metaFieldsMarker
}

type M = map[string]interface{}
type L = []KV

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

func (l *MultLogger) Id(v string) *LogId {
	return Id(v)
}

func (l *MultLogger) NewLoggerWithLock() (*MultLogger, func()) {
	shared.M1.Lock()
	defer shared.M1.Unlock()
	var newLck = &sync.Mutex{}
	newLck.Lock()
	var id = uuid.New().String()
	lockStack.Push(&stack.StackItem{
		Id:  id,
		Lck: newLck,
	})
	var z = MultLogger{
		AppName:    l.AppName,
		HostName:   l.HostName,
		TimeZone:   l.TimeZone,
		MetaFields: l.MetaFields,
		LockUuid:   id,
	}
	return &z, z.unlock
}

func (l *MultLogger) unlock() {

	shared.M1.Lock()
	defer shared.M1.Unlock()

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

	x, err := lockStack.Pop()
	if x != peek {
		panic("must equal peek")
	}
	fmt.Println("unlocking:", peek.Id)
	peek.Lck.Unlock()

}

func (l *MultLogger) Child(m *map[string]interface{}) *MultLogger {

	var z = make(map[string]interface{})
	for k, v := range *l.MetaFields.m {
		z[k] = hlpr.CopyAndDereference(v)
	}

	for k, v := range *m {
		z[k] = hlpr.CopyAndDereference(v)
	}

	return &MultLogger{
		AppName:    l.AppName,
		HostName:   l.HostName,
		TimeZone:   l.TimeZone,
		MetaFields: NewMetaFields(&z),
		LockUuid:   l.LockUuid,
		EnvPrefix:  l.EnvPrefix,
		Files:      l.Files,
	}
}

type SprintFStruct struct {
	SprintF string
}

func (l *MultLogger) Create(m *map[string]interface{}) *MultLogger {
	return l.Child(m)
}

func (l *MultLogger) getPrettyString(level shared.LogLevel, m *MetaFields, args *[]interface{}) string {

	date := time.Now().UTC().String()[11:25] // only first 25 chars
	stylizedLevel := "<undefined>"

	switch level {

	case shared.ERROR:
		stylizedLevel = aurora.Underline(aurora.Bold(aurora.Red("ERROR"))).String()
		break

	case shared.WARN:
		stylizedLevel = aurora.Magenta("WARN").String()
		break

	case shared.DEBUG:
		stylizedLevel = aurora.Bold("DEBUG").String()
		break

	case shared.INFO:
		stylizedLevel = aurora.Gray(12, "INFO").String()
		break

	case shared.TRACE:
		stylizedLevel = aurora.Gray(4, "TRACE").String()
		break
	}

	var b strings.Builder

	b.WriteString(aurora.Gray(9, date).String())
	b.WriteString(" ")
	b.WriteString(stylizedLevel)
	b.WriteString(" ")
	b.WriteString(aurora.Gray(12, "app:").String())
	b.WriteString(aurora.Italic(l.AppName).String())
	b.WriteString(" ")

	size := 0

	for _, v := range *args {

		var primitive = true

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

		if shared.IsNonPrimitive(kind) {
			primitive = false
		}

		s := hlpr.GetPrettyString(v, size) + " "
		i := strings.LastIndex(s, "\n")
		if i >= 0 {
			size = len(s) - i
		} else {
			size = size + len(s)
		}

		if _, err := b.WriteString(s); err != nil {
			l.writeToStderr("771c710b-aba2-46ef-9126-c26d3dfe7925", err)
		}

		if !primitive && (level == shared.TRACE || level == shared.DEBUG) {

			if _, err := b.WriteString("\n"); err != nil {
				l.writeToStderr("18614292-658f-42a5-81e7-593e941ea857", err)
			}

			if _, err := b.WriteString(fmt.Sprintf("sprintf: %+v", v)); err != nil {
				l.writeToStderr("2a795ef2-65bb-4a03-9808-b072e4497d73", err)
			}

			b.Write([]byte("json:"))
			if x, err := json.Marshal(v); err == nil {
				if _, err := b.Write(x); err != nil {
					l.writeToStderr("err:56831878-8d63-45f4-905b-d1b3bbac2152:", err)
				}
			} else {
				l.writeToStderr("err:70bf10e0-6e69-4a3b-bf64-08f6d20c4580:", err)
			}

		}

	}

	if _, err := b.WriteString("\n"); err != nil {
		l.writeToStderr("f834d14a-9735-4fd6-9389-f79144044746", err)
	}

	return b.String()
}

func (l *MultLogger) writeToStderr(args ...interface{}) {
	if _, err := fmt.Fprintln(os.Stderr, args...); err != nil {
		fmt.Println("adcca45f-8d7b-4d4a-8fd2-7683b7b375b5", "could not write to stderr:", err)
	}
}

func (l *MultLogger) writeJSONFromFormattedStr(level shared.LogLevel, m *MetaFields, s *[]interface{}) {

	date := time.Now().UTC().String()
	date = date[:26]
	var strLevel = shared.LevelToString[level]
	var pid = shared.PID

	buf, err := json.Marshal([8]interface{}{"@bunion:v1", l.AppName, strLevel, pid, l.HostName, date, m, s})

	if err != nil {
		DefaultLogger.Warn("1f1512fa-d1ff-42af-a8d0-6a52801f917d", err)
		return
	}

	for _, v := range l.Files {
		go func(v *FileLevel) {
			v.lock.Lock()
			defer v.lock.Unlock()
			v.File.Write(buf)
			v.File.Write([]byte("\n"))
		}(v)
	}

}

func (l *MultLogger) writeJSON(level shared.LogLevel, mf *MetaFields, args *[]interface{}) {

	date := time.Now().UTC().String()
	date = date[:26]
	var strLevel = shared.LevelToString[level]
	var pid = shared.PID

	buf, err := json.Marshal([8]interface{}{"@bunion:v1", l.AppName, strLevel, pid, l.HostName, date, mf.m, args})

	if err != nil {

		_, file, line, _ := runtime.Caller(3)

		DefaultLogger.Warn("could not marshal the slice:", err.Error(), "file://"+file+":"+strconv.Itoa(line))

		var cache = map[*interface{}]*interface{}{}
		var cleaned = make([]interface{}, 0)

		for i := 0; i < len(*args); i++ {
			// TODO: for now instead of cleanUp, we can ust fmt.Sprintf()
			v := &(*args)[i]
			c := hlpr.CleanUp(v, &cache)
			debug.PrintStack()
			cleaned = append(cleaned, c)
		}

		buf, err = json.Marshal([8]interface{}{"@bunion:v1", l.AppName, level, pid, l.HostName, date, mf.m, cleaned})

		if err != nil {
			fmt.Println(errors.New("Json-Logging: could not marshal the slice: " + err.Error()))
			return
		}
	}

	for _, v := range l.Files {
		go func(v *FileLevel) {
			v.lock.Lock()
			defer v.lock.Unlock()
			if _, err := v.File.Write(buf); err != nil {
				l.writeToStderr("1944431c-d90f-4e41-975f-206da000d85d", err)
			}
			if _, err := v.File.Write([]byte("\n")); err != nil {
				l.writeToStderr("ea20aee7-862d-4596-8639-52073c835757", err)
			}
		}(v)
	}
}

func (l *MultLogger) PrintEnvPlain() {
	envVars := os.Environ() // Get all environment variables as a slice
	sort.Strings(envVars)
	for _, env := range envVars {
		log.Println(env)
	}
}

func (l *MultLogger) PrintEnv() {
	envVars := os.Environ() // Get all environment variables as a slice
	sort.Strings(envVars)
	for _, env := range envVars {
		l.Info(env)
	}
}

func (l *MultLogger) writeSwitchForFormattedString(level shared.LogLevel, m *MetaFields, s *[]interface{}) {
	if l.IsLoggingJSON {
		l.writeJSONFromFormattedStr(level, m, s)
	}
}

func (l *MultLogger) writeSwitch(level shared.LogLevel, m *MetaFields, args *[]interface{}) {
	if l.IsLoggingJSON {
		l.writeJSON(level, m, args)
	} else {
		l.writePretty(level, m, args)
	}
}

func (l *MultLogger) JSON(args ...interface{}) {
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

func (l *MultLogger) RawJSON(args ...interface{}) {
	// raw = no newlines, no spaces
	for i := 0; i < len(args); i++ {

		v, err := json.Marshal(args[i])

		if err != nil {
			panic(err)
		}

		os.Stdout.Write(v)
	}
}

func (l *MultLogger) getMetaFields(args *[]interface{}) (*MetaFields, []interface{}) {
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
			m["log_id"] = z.val
			newArgs = append(newArgs, z.val)
		} else if z, ok := x.(LogId); ok {
			m["log_id"] = z.val
			newArgs = append(newArgs, z.val)
		} else {
			newArgs = append(newArgs, x)
		}
	}

	return mf, newArgs
}

func (l *MultLogger) Info(args ...interface{}) {
	var meta, newArgs = l.getMetaFields(&args)
	if l.isInfo {
		l.writeSwitch(shared.INFO, meta, &newArgs)
	}
}

func (l *MultLogger) Warn(args ...interface{}) {
	var meta, newArgs = l.getMetaFields(&args)
	if l.isWarn {
		l.writeSwitch(shared.WARN, meta, &newArgs)
	}

}

func (l *MultLogger) Error(args ...interface{}) {
	var meta, newArgs = l.getMetaFields(&args)
	filteredStackTrace := hlpr.GetFilteredStacktrace()
	newArgs = append(newArgs, StackTrace{filteredStackTrace})
	l.writeSwitch(shared.ERROR, meta, &newArgs)
}

func (l *MultLogger) Debug(args ...interface{}) {
	if l.isDebug {
		var meta, newArgs = l.getMetaFields(&args)
		l.writeSwitch(shared.DEBUG, meta, &newArgs)
	}

}

func (l *MultLogger) Trace(args ...interface{}) {
	if l.isTrace {
		var meta, newArgs = l.getMetaFields(&args)
		l.writeSwitch(shared.TRACE, meta, &newArgs)
	}
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

func (l *MultLogger) TagPair(k string, v interface{}) *MultLogger {
	var z = map[string]interface{}{k: v}
	return l.Child(&z)
}

func (l *MultLogger) Tags(z *map[string]interface{}) *MultLogger {
	return l.Create(z)
}

func (l *MultLogger) InfoF(s string, args ...interface{}) {
	if l.isInfo {
		l.writeSwitchForFormattedString(shared.INFO, nil, &[]interface{}{fmt.Sprintf(s, args...)})
	}
}

func (l *MultLogger) WarnF(s string, args ...interface{}) {
	if l.isWarn {
		l.writeSwitchForFormattedString(shared.WARN, nil, &[]interface{}{fmt.Sprintf(s, args...)})
	}

}

type StackTrace struct {
	ErrorTrace *[]string
}

func (l *MultLogger) ErrorF(s string, args ...interface{}) {
	filteredStackTrace := hlpr.GetFilteredStacktrace()
	formattedString := fmt.Sprintf(s, args...)
	l.writeSwitchForFormattedString(shared.ERROR, nil, &[]interface{}{formattedString, StackTrace{filteredStackTrace}})
}

func (l *MultLogger) DebugF(s string, args ...interface{}) {
	if l.isDebug {
		l.writeSwitchForFormattedString(shared.DEBUG, nil, &[]interface{}{fmt.Sprintf(s, args...)})
	}
}

func (l *MultLogger) TraceF(s string, args ...interface{}) {
	if l.isTrace {
		l.writeSwitchForFormattedString(shared.TRACE, nil, &[]interface{}{fmt.Sprintf(s, args...)})
	}

}

func (l *MultLogger) NewLine() {
	safeStdout.Write([]byte("\n"))
}

func (l *MultLogger) Spaces(num int32) {
	safeStdout.Write([]byte(strings.Join(make([]string, num), " ")))
}

func (l *MultLogger) Tabs(num int32) {
	safeStdout.Write([]byte(strings.Join(make([]string, num), "\t")))
}

func (l *MultLogger) PlainStdout(args ...interface{}) {
	safeStdout.Lock()
	for _, a := range args {
		v := fmt.Sprintf("((%T) %#v) ", a, a)
		os.Stdout.Write([]byte(v))
	}
	os.Stdout.Write([]byte("\n"))
	safeStdout.Unlock()
}

func (l *MultLogger) PlainStderr(args ...interface{}) {
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
	"",
	[]*FileLevel{&FileLevel{
		Level: shared.TRACE,
		File:  os.Stdout,
	}},
)

func init() {

	//log.SetFlags(log.LstdFlags | log.Llongfile)

}
