package mult

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	uuid "github.com/google/uuid"
	au "github.com/oresoftware/json-logging/jlog/au"
	hlpr "github.com/oresoftware/json-logging/jlog/helper"
	ll "github.com/oresoftware/json-logging/jlog/level"
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
	Level      ll.LogLevel
	File       *os.File
	Tags       *map[string]interface{}
	lock       *sync.RWMutex
	IsJSON     bool
	isTrace    bool
	isDebug    bool
	isInfo     bool
	isWarn     bool
	isError    bool
	isCritical bool
}

type MultiLogger struct {
	Mtx        sync.RWMutex
	AppName    string
	HostName   string
	MetaFields *MetaFields
	TimeZone   time.Location
	LockUuid   string
	EnvPrefix  string
	Files      []*FileLevel
	isTrace    bool
	isDebug    bool
	isInfo     bool
	isWarn     bool
	isError    bool
	isCritical bool
}

type MultLoggerParams struct {
	AppName    string
	HostName   string
	MetaFields *MetaFields
	TimeZone   time.Location
	LockUuid   string
	EnvPrefix  string
	Files      []*FileLevel
}

// TODO: create a goroutine for each Output path
// write to that existing goroutine

func getFileInfo(f *os.File) (string, error) {

	var fd1 = f.Fd()
	var stat1 syscall.Stat_t

	if err := syscall.Fstat(int(fd1), &stat1); err != nil {
		return "", err
	}

	return strconv.Itoa(int(stat1.Dev)) + ":" + strconv.Itoa(int(stat1.Ino)), nil
}

func mapFileLevels(x []*FileLevel) []*FileLevel {

	var results = []*FileLevel{}
	var m1 = map[uintptr]*sync.RWMutex{}
	var m2 = map[string]*sync.RWMutex{}

	for _, z := range x {
		if z == nil {
			continue
		}

		if z.File == nil {
			z.File = os.Stdout
		}

		if z.File.Fd() == os.Stdout.Fd() && !shared.IsTerminal {
			z.IsJSON = true
		}

		var fd = z.File.Fd()
		var fileInfo, err = getFileInfo(z.File)

		if err == nil {
			if _, ok := m2[fileInfo]; !ok {
				m2[fileInfo] = &sync.RWMutex{}
			}
			z.lock = m2[fileInfo]
			results = append(results, z)
			continue
		}

		if _, ok := m1[fd]; !ok {
			m1[fd] = &sync.RWMutex{}
		}

		z.lock = m1[fd]
		results = append(results, z)

	}

	return results
}

func NewLogger(AppName string, envTokenPrefix string, files ...*FileLevel) *MultiLogger {
	return NewMultiLogger(MultLoggerParams{
		AppName:   AppName,
		EnvPrefix: envTokenPrefix,
		Files:     files,
	})
}

func New(AppName string, envTokenPrefix string, files []*FileLevel) *MultiLogger {
	return NewMultiLogger(MultLoggerParams{
		AppName:   AppName,
		EnvPrefix: envTokenPrefix,
		Files:     files,
	})
}

func NewMultiLogger(p MultLoggerParams) *MultiLogger {

	var files = []*FileLevel{}

	if p.Files != nil {
		files = mapFileLevels(p.Files)
	}

	if len(files) < 1 {
		files = append(files, &FileLevel{
			Level:  ll.TRACE,
			File:   os.Stdout,
			Tags:   nil,
			lock:   &sync.RWMutex{},
			IsJSON: !shared.IsTerminal,
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

	var l = &MultiLogger{
		Mtx:        sync.RWMutex{},
		AppName:    appName,
		HostName:   hostName,
		TimeZone:   p.TimeZone,
		MetaFields: metaFields,
		LockUuid:   p.LockUuid,
		EnvPrefix:  p.EnvPrefix,
		Files:      files,
		isError:    true,
		isWarn:     false,
		isInfo:     false,
		isDebug:    false,
		isTrace:    false,
		isCritical: true,
	}

	l.determineInitialLogLevels()
	return l
}

func (l *MultiLogger) SetEnvPrefix(s string) *MultiLogger {
	l.Mtx.Lock()
	defer l.Mtx.Unlock()
	l.EnvPrefix = s
	return l
}

func (l *MultiLogger) SetToDisplayLocalTZ(f *os.File) *MultiLogger {
	// TODO: use lock to set this
	return l
}

func (l *MultiLogger) AddOutputFile(level ll.LogLevel, f *os.File) *MultiLogger {
	if f == nil {
		f = os.Stdout
	}

	l.Mtx.Lock()
	defer l.Mtx.Unlock()

	isJSON := f.Fd() == os.Stdout.Fd() && !shared.IsTerminal

	files := mapFileLevels(append(l.Files, &FileLevel{
		Level:      level,
		File:       f,
		Tags:       nil,
		lock:       nil,
		IsJSON:     isJSON,
		isTrace:    false,
		isDebug:    false,
		isInfo:     false,
		isWarn:     false,
		isError:    false,
		isCritical: false,
	}))
	l.Files = files
	l.determineInitialLogLevels()
	return l
}

func (l *MultiLogger) SetMinLogLevel(f ll.LogLevel) *MultiLogger {
	return l
}

func (l *MultiLogger) SetMaxLogLevel(f ll.LogLevel) *MultiLogger {
	return l
}

func (l *MultiLogger) RemoveTag(s string) *MultiLogger {
	l.Mtx.Lock()
	defer l.Mtx.Unlock()
	delete(*l.MetaFields.m, s)
	return l
}

func (l *MultiLogger) AddTag(s string, v interface{}) *MultiLogger {
	l.Mtx.Lock()
	defer l.Mtx.Unlock()
	(*l.MetaFields.m)[s] = v
	return l
}

func (l *MultiLogger) AddMetaField(s string, v interface{}) *MultiLogger {
	l.Mtx.Lock()
	defer l.Mtx.Unlock()
	(*l.MetaFields.m)[s] = v
	return l
}

func (l *MultiLogger) SetToJSONOutput() *MultiLogger {
	l.Mtx.Lock()
	defer l.Mtx.Unlock()
	for _, f := range l.Files {
		if f != nil {
			f.IsJSON = true
		}
	}
	return l
}

func (l *MultiLogger) SetAppName(h string) *MultiLogger {
	l.Mtx.Lock()
	defer l.Mtx.Unlock()
	l.AppName = h
	return l
}

func (l *MultiLogger) SetTimeZone(h time.Location) *MultiLogger {
	l.Mtx.Lock()
	defer l.Mtx.Unlock()
	l.TimeZone = h
	return l
}

func (l *MultiLogger) SetHostName(h string) *MultiLogger {
	l.Mtx.Lock()
	defer l.Mtx.Unlock()
	l.HostName = h
	return l
}

func (l *MultiLogger) determineInitialLogLevels() {
	///
	l.isTrace = false
	l.isDebug = false
	l.isInfo = false
	l.isWarn = false
	l.isError = false
	l.isCritical = false

	if len(l.Files) < 1 {

		l.isTrace = true
		l.isDebug = true
		l.isInfo = true
		l.isWarn = true
		l.isError = true
		l.isCritical = true

		l.Files = append(l.Files, &FileLevel{
			Level:      ll.TRACE,
			File:       os.Stdout,
			lock:       &sync.RWMutex{},
			IsJSON:     !shared.IsTerminal,
			isTrace:    true,
			isDebug:    true,
			isInfo:     true,
			isWarn:     true,
			isError:    true,
			isCritical: true,
		})

		return
	}

	for _, v := range l.Files {
		if v == nil {
			continue
		}

		if v.File == nil {
			v.File = os.Stdout
		}

		if v.lock == nil {
			v.lock = &sync.RWMutex{}
		}

		v.isTrace = false
		v.isDebug = false
		v.isInfo = false
		v.isWarn = false
		v.isError = false
		v.isCritical = false

		v.isCritical = true

		switch v.Level {
		case ll.CRITICAL:
			l.isCritical = true
		case ll.ERROR:
			v.isError = true
			l.isError = true
			l.isCritical = true
		case ll.WARN:
			v.isWarn = true
			l.isWarn = true
			v.isError = true
			l.isError = true
			l.isCritical = true
		case ll.INFO:
			v.isWarn = true
			l.isWarn = true
			v.isInfo = true
			l.isInfo = true
			v.isError = true
			l.isError = true
			l.isCritical = true
		case ll.DEBUG:
			v.isWarn = true
			l.isWarn = true
			v.isInfo = true
			l.isInfo = true
			v.isDebug = true
			l.isDebug = true
			v.isError = true
			l.isError = true
			l.isCritical = true
		case ll.TRACE:
			v.isWarn = true
			l.isWarn = true
			v.isInfo = true
			l.isInfo = true
			v.isDebug = true
			l.isDebug = true
			v.isTrace = true
			l.isTrace = true
			v.isError = true
			l.isError = true
			l.isCritical = true
		default:
			panic("should have a log-level chosen")
		}

		v.isCritical = true
	}
}

//func NewMultiLogger(AppName string, forceJSON bool, hostName string, envTokenPrefix string) *MultiLogger {
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

func (l *MultiLogger) Id(v string) *LogId {
	return Id(v)
}

func levelEnabled(minLevel ll.LogLevel, level ll.LogLevel) bool {
	return level >= minLevel
}

func fileAllowsLevel(f *FileLevel, level ll.LogLevel) bool {
	if f == nil {
		return false
	}

	switch level {
	case ll.TRACE:
		return f.isTrace
	case ll.DEBUG:
		return f.isDebug
	case ll.INFO:
		return f.isInfo
	case ll.WARN:
		return f.isWarn
	case ll.ERROR:
		return f.isError
	case ll.CRITICAL:
		return f.isCritical
	default:
		return false
	}
}

func V(level ll.LogLevel) bool {
	return DefaultLogger.V(level)
}

func IsLevelEnabled(level ll.LogLevel) bool {
	return DefaultLogger.IsLevelEnabled(level)
}

func (l *MultiLogger) V(level ll.LogLevel) bool {
	return l.IsLevelEnabled(level)
}

func (l *MultiLogger) IsLevelEnabled(level ll.LogLevel) bool {
	l.Mtx.RLock()
	defer l.Mtx.RUnlock()

	for _, f := range l.Files {
		if fileAllowsLevel(f, level) {
			return true
		}
	}

	return false
}

func (l *MultiLogger) IsTraceEnabled() bool {
	return l.IsLevelEnabled(ll.TRACE)
}

func (l *MultiLogger) IsDebugEnabled() bool {
	return l.IsLevelEnabled(ll.DEBUG)
}

func (l *MultiLogger) IsInfoEnabled() bool {
	return l.IsLevelEnabled(ll.INFO)
}

func (l *MultiLogger) IsWarnEnabled() bool {
	return l.IsLevelEnabled(ll.WARN)
}

func (l *MultiLogger) IsErrorEnabled() bool {
	return l.IsLevelEnabled(ll.ERROR)
}

func (l *MultiLogger) IsCriticalEnabled() bool {
	return l.IsLevelEnabled(ll.CRITICAL)
}

func (l *MultiLogger) filesForLevel(level ll.LogLevel) []*FileLevel {
	l.Mtx.RLock()
	defer l.Mtx.RUnlock()

	files := make([]*FileLevel, 0, len(l.Files))
	for _, f := range l.Files {
		if fileAllowsLevel(f, level) {
			files = append(files, f)
		}
	}

	return files
}

func (l *MultiLogger) allFiles() []*FileLevel {
	l.Mtx.RLock()
	defer l.Mtx.RUnlock()
	return append([]*FileLevel(nil), l.Files...)
}

func (l *MultiLogger) writeOutput(file *os.File, b []byte) {
	if file == nil {
		file = os.Stdout
	}

	l.Mtx.RLock()
	isLockedLogger := l.LockUuid != ""
	l.Mtx.RUnlock()

	if !isLockedLogger {
		shared.M1.RLock()
		defer shared.M1.RUnlock()
	}

	if _, err := writer.Write(file, b); err != nil {
		l.writeToStderr("json-logging: could not write log output:", err)
	}
}

func (l *MultiLogger) NewLoggerWithLock() (*MultiLogger, func()) {
	shared.M1.Lock()
	var id = uuid.New().String()
	lockStack.Push(&stack.StackItem{
		Id: id,
	})
	l.Mtx.RLock()
	files := append([]*FileLevel(nil), l.Files...)
	var z = MultiLogger{
		Mtx:        sync.RWMutex{},
		AppName:    l.AppName,
		HostName:   l.HostName,
		TimeZone:   l.TimeZone,
		MetaFields: l.MetaFields,
		LockUuid:   id,
		EnvPrefix:  l.EnvPrefix,
		Files:      files,
		isTrace:    l.isTrace,
		isDebug:    l.isDebug,
		isInfo:     l.isInfo,
		isWarn:     l.isWarn,
		isError:    l.isError,
		isCritical: l.isCritical,
	}
	l.Mtx.RUnlock()
	return &z, z.unlock
}

func (l *MultiLogger) unlock() {
	var peek, err = lockStack.Peek()

	if peek == nil {
		panic("error with lib - peek should not be nil")
	}

	if err != nil {
		panic("error should be nil if peek item exists")
	}

	if peek.Id != l.LockUuid {
		panic("lock ids do not match")
	}

	x, err := lockStack.Pop()
	if err != nil {
		panic(err)
	}
	if x != peek {
		panic("must equal peek")
	}
	shared.M1.Unlock()
}

func (l *MultiLogger) Child(m *map[string]interface{}) *MultiLogger {

	l.Mtx.RLock()
	defer l.Mtx.RUnlock()

	var z = make(map[string]interface{})
	for k, v := range *l.MetaFields.m {
		z[k] = hlpr.CopyAndDereference(v)
	}

	for k, v := range *m {
		z[k] = hlpr.CopyAndDereference(v)
	}

	return &MultiLogger{
		Mtx:        sync.RWMutex{},
		AppName:    l.AppName,
		HostName:   l.HostName,
		TimeZone:   l.TimeZone,
		MetaFields: NewMetaFields(&z),
		LockUuid:   l.LockUuid,
		EnvPrefix:  l.EnvPrefix,
		Files:      l.Files,
		isTrace:    l.isTrace,
		isDebug:    l.isDebug,
		isInfo:     l.isInfo,
		isWarn:     l.isWarn,
		isError:    l.isError,
		isCritical: l.isCritical,
	}
}

type SprintFStruct struct {
	SprintF string
}

func (l *MultiLogger) Create(m *map[string]interface{}) *MultiLogger {
	return l.Child(m)
}

func (l *MultiLogger) getPrettyString(level ll.LogLevel, m *MetaFields, args *[]interface{}) string {

	l.Mtx.RLock()
	appName := l.AppName
	l.Mtx.RUnlock()

	date := time.Now().UTC().Format("15:04:05.000000")
	stylizedLevel := "<undefined>"

	switch level {

	case ll.ERROR:
		stylizedLevel = au.Col.Underline(au.Col.Bold(au.Col.Red("ERROR"))).String()
		break

	case ll.WARN:
		stylizedLevel = au.Col.Magenta("WARN").String()
		break

	case ll.DEBUG:
		stylizedLevel = au.Col.Bold("DEBUG").String()
		break

	case ll.INFO:
		stylizedLevel = au.Col.Gray(12, "INFO").String()
		break

	case ll.TRACE:
		stylizedLevel = au.Col.Gray(4, "TRACE").String()
		break
	}

	var b strings.Builder

	b.WriteString(au.Col.Gray(9, date).String())
	b.WriteString(" ")
	b.WriteString(stylizedLevel)
	b.WriteString(" ")
	b.WriteString(au.Col.Gray(12, "app:").String())
	b.WriteString(au.Col.Italic(appName).String())
	b.WriteString(" ")

	size := 0

	for _, v := range *args {

		var primitive = true

		if v == nil {
			b.WriteString(fmt.Sprintf("<nil 1> - (%T)", v))
			continue
		}

		val := reflect.ValueOf(v)
		var kind = reflect.TypeOf(v).Kind()

		if kind == reflect.Ptr || kind == reflect.Interface {
			if val.IsNil() {
				b.WriteString(fmt.Sprintf("<nil (%T)>", v))
				continue
			}
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

		if !primitive && (level == ll.TRACE || level == ll.DEBUG) {

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

func (l *MultiLogger) writeToStderr(args ...interface{}) {
	if _, err := safeStderr.Write([]byte(fmt.Sprintln(args...))); err != nil {
		fmt.Println("adcca45f-8d7b-4d4a-8fd2-7683b7b375b5", "could not write to stderr:", err)
	}
}

func F(s string, args ...interface{}) string {
	return fmt.Sprintf(s, args...)
}

func (l *MultiLogger) F(s string, args ...interface{}) string {
	return fmt.Sprintf(s, args...)
}

func (l *MultiLogger) writeJSON(level ll.LogLevel, mf *MetaFields, args *[]interface{}) {

	date := time.Now().UTC().Format("2006-01-02 15:04:05.000000")
	var strLevel = shared.LevelToString[level]
	var pid = shared.PID

	if mf == nil {
		mf = NewMetaFields(&MF{})
	}

	l.Mtx.RLock()
	appName := l.AppName
	hostName := l.HostName
	l.Mtx.RUnlock()

	files := l.filesForLevel(level)
	if len(files) < 1 {
		return
	}

	var jsonBuf []byte
	var jsonReady bool
	var prettyBuf []byte

	for _, v := range files {
		if v == nil {
			continue
		}

		if v.File == nil {
			v.File = os.Stdout
		}

		if v.IsJSON {
			if !jsonReady {
				buf, err := json.Marshal([8]interface{}{"@bunion:v1", appName, strLevel, pid, hostName, date, mf.m, *args})

				if err != nil {
					_, file, line, _ := runtime.Caller(3)

					DefaultLogger.Warn("could not marshal the slice:", err.Error(), "file://"+file+":"+strconv.Itoa(line))

					var cache = map[uintptr]interface{}{}
					var cleaned = make([]interface{}, 0, len(*args))

					for i := 0; i < len(*args); i++ {
						v := &(*args)[i]
						c := hlpr.CleanUp(v, &cache)
						cleaned = append(cleaned, c)
					}

					buf, err = json.Marshal([8]interface{}{"@bunion:v1", appName, strLevel, pid, hostName, date, mf.m, cleaned})

					if err != nil {
						l.writeToStderr(errors.New("Json-Logging: could not marshal the slice: " + err.Error()))
						return
					}
				}

				jsonBuf = append(buf, '\n')
				jsonReady = true
			}

			l.writeOutput(v.File, jsonBuf)
			continue
		}

		if prettyBuf == nil {
			prettyBuf = []byte(l.getPrettyString(level, mf, args))
		}
		l.writeOutput(v.File, prettyBuf)
	}

}

func (l *MultiLogger) PrintEnvPlain() {
	envVars := os.Environ() // Get all environment variables as a slice
	sort.Strings(envVars)
	for _, env := range envVars {
		log.Println(env)
	}
}

func (l *MultiLogger) PrintEnv() {
	envVars := os.Environ() // Get all environment variables as a slice
	sort.Strings(envVars)
	for _, env := range envVars {
		l.Info(env)
	}
}

func (l *MultiLogger) writeSwitchForFormattedString(level ll.LogLevel, m *MetaFields, s *[]interface{}) {
	l.writeJSON(level, m, s)
}

func (l *MultiLogger) writeSwitch(level ll.LogLevel, m *MetaFields, args *[]interface{}) {
	l.writeJSON(level, m, args)
}

func (l *MultiLogger) JSON(args ...interface{}) {
	size := len(args)
	var b bytes.Buffer
	for i := 0; i < size; i++ {

		v, err := json.Marshal(args[i])

		if err != nil {
			panic(err)
		}

		b.Write(v)
		if i < size-1 {
			b.WriteByte(' ')
		}
	}
	b.WriteByte('\n')
	l.writeOutput(os.Stdout, b.Bytes())
}

func (l *MultiLogger) RawJSON(args ...interface{}) {
	// raw = no newlines, no spaces
	var b bytes.Buffer
	for i := 0; i < len(args); i++ {

		v, err := json.Marshal(args[i])

		if err != nil {
			panic(err)
		}

		b.Write(v)
	}
	l.writeOutput(os.Stdout, b.Bytes())
}

func (l *MultiLogger) getMetaFields(args *[]interface{}) (*MetaFields, []interface{}) {
	var newArgs = []interface{}{}
	var m = MF{}
	var mf = NewMetaFields(&m)

	l.Mtx.RLock()
	for k, v := range *l.MetaFields.m {
		m[k] = v
	}
	l.Mtx.RUnlock()

	hasLogId := false

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
			hasLogId = true
		} else if z, ok := x.(LogId); ok {
			m["log_id"] = z.val
			hasLogId = true
		} else {
			newArgs = append(newArgs, x)
		}
	}

	if false && !hasLogId {
		fmt.Println("missing log id:", string(debug.Stack()))
	}

	return mf, newArgs
}

func (l *MultiLogger) Info(args ...interface{}) {
	if !l.IsLevelEnabled(ll.INFO) {
		return
	}
	var meta, newArgs = l.getMetaFields(&args)
	l.writeSwitch(ll.INFO, meta, &newArgs)
}

func (l *MultiLogger) Warn(args ...interface{}) {
	if !l.IsLevelEnabled(ll.WARN) {
		return
	}
	var meta, newArgs = l.getMetaFields(&args)
	l.writeSwitch(ll.WARN, meta, &newArgs)
}

func (l *MultiLogger) Error(args ...interface{}) {
	if !l.IsLevelEnabled(ll.ERROR) {
		return
	}
	var meta, newArgs = l.getMetaFields(&args)
	filteredStackTrace := hlpr.GetFilteredStacktrace()
	newArgs = append(newArgs, StackTrace{filteredStackTrace})
	l.writeSwitch(ll.ERROR, meta, &newArgs)
}

func (l *MultiLogger) Debug(args ...interface{}) {
	if !l.IsLevelEnabled(ll.DEBUG) {
		return
	}
	var meta, newArgs = l.getMetaFields(&args)
	l.writeSwitch(ll.DEBUG, meta, &newArgs)
}

func (l *MultiLogger) Trace(args ...interface{}) {
	if !l.IsLevelEnabled(ll.TRACE) {
		return
	}
	var meta, newArgs = l.getMetaFields(&args)
	l.writeSwitch(ll.TRACE, meta, &newArgs)
}

func (l *MultiLogger) Critical(args ...interface{}) {
	if !l.IsLevelEnabled(ll.CRITICAL) {
		return
	}
	var meta, newArgs = l.getMetaFields(&args)
	filteredStackTrace := hlpr.GetFilteredStacktrace()
	newArgs = append(newArgs, StackTrace{filteredStackTrace})
	l.writeSwitch(ll.CRITICAL, meta, &newArgs)
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

type StackTrace struct {
	ErrorTrace *[]string
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

func (l *MultiLogger) TagPair(k string, v interface{}) *MultiLogger {
	var z = map[string]interface{}{k: v}
	return l.Child(&z)
}

func (l *MultiLogger) Tags(z *map[string]interface{}) *MultiLogger {
	return l.Create(z)
}

func (l *MultiLogger) ErrorF(s string, args ...interface{}) {
	if !l.IsLevelEnabled(ll.ERROR) {
		return
	}
	filteredStackTrace := hlpr.GetFilteredStacktrace()
	formattedString := fmt.Sprintf(s, args...)
	l.writeSwitchForFormattedString(ll.ERROR, nil, &[]interface{}{formattedString, StackTrace{filteredStackTrace}})
}

func (l *MultiLogger) WarnF(s string, args ...interface{}) {
	if !l.IsLevelEnabled(ll.WARN) {
		return
	}
	l.writeSwitchForFormattedString(ll.WARN, nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *MultiLogger) InfoF(s string, args ...interface{}) {
	if !l.IsLevelEnabled(ll.INFO) {
		return
	}
	l.writeSwitchForFormattedString(ll.INFO, nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *MultiLogger) DebugF(s string, args ...interface{}) {
	if !l.IsLevelEnabled(ll.DEBUG) {
		return
	}
	l.writeSwitchForFormattedString(ll.DEBUG, nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *MultiLogger) TraceF(s string, args ...interface{}) {
	if !l.IsLevelEnabled(ll.TRACE) {
		return
	}
	l.writeSwitchForFormattedString(ll.TRACE, nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *MultiLogger) CriticalF(s string, args ...interface{}) {
	if !l.IsLevelEnabled(ll.CRITICAL) {
		return
	}
	filteredStackTrace := hlpr.GetFilteredStacktrace()
	formattedString := fmt.Sprintf(s, args...)
	l.writeSwitchForFormattedString(ll.CRITICAL, nil, &[]interface{}{formattedString, StackTrace{filteredStackTrace}})
}

func (l *MultiLogger) NewLine() {
	for _, n := range l.allFiles() {
		if n != nil {
			l.writeOutput(n.File, []byte("\n"))
		}
	}
}

func (l *MultiLogger) Spaces(num int32) {
	if num < 1 {
		return
	}
	buf := []byte(strings.Repeat(" ", int(num)))
	for _, n := range l.allFiles() {
		if n != nil {
			l.writeOutput(n.File, buf)
		}
	}
}

func (l *MultiLogger) Tabs(num int32) {
	if num < 1 {
		return
	}
	buf := []byte(strings.Repeat("\t", int(num)))
	for _, n := range l.allFiles() {
		if n != nil {
			l.writeOutput(n.File, buf)
		}
	}
}

func (l *MultiLogger) JustStdout(args ...interface{}) {
	var b bytes.Buffer
	for _, a := range args {
		v := fmt.Sprintf("((%T) %#v) ", a, a)
		b.WriteString(v)
	}
	b.WriteByte('\n')
	l.writeOutput(os.Stdout, b.Bytes())
}

func (l *MultiLogger) PlainStdout(args ...interface{}) {
	var b bytes.Buffer
	for _, a := range args {
		v := fmt.Sprintf("((%T) %#v) ", a, a)
		b.WriteString(v)
	}
	b.WriteByte('\n')

	for _, n := range l.allFiles() {
		if n != nil {
			l.writeOutput(n.File, b.Bytes())
		}
	}
}

func (l *MultiLogger) PlainStderr(args ...interface{}) {
	var b bytes.Buffer
	for _, a := range args {
		v := fmt.Sprintf("((%T) %#v) ", a, a)
		b.WriteString(v)
	}
	b.WriteByte('\n')
	l.writeOutput(os.Stderr, b.Bytes())
}

var DefaultLogger = New(
	"Default",
	"",
	[]*FileLevel{&FileLevel{
		Level:  ll.TRACE,
		File:   os.Stdout,
		IsJSON: !shared.IsTerminal,
	}},
)

func init() {
	//log.SetFlags(log.LstdFlags | log.Llongfile)
}
