package mult

import (
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
  "sort"
  "strconv"
  "strings"
  "sync"
  "syscall"
  "time"
  "runtime/debug"
)

var safeStdout = writer.NewSafeWriter(os.Stdout)
var safeStderr = writer.NewSafeWriter(os.Stderr)
var lockStack = stack.NewStack()

type FileLevel struct {
  Level   ll.LogLevel
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

type MultiLogger struct {
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
  var m1 = map[uintptr]*sync.Mutex{}
  var m2 = map[string]*sync.Mutex{}

  for _, z := range x {

    var fd = z.File.Fd()
    //var x = z.File.Name()

    if _, ok := m1[fd]; !ok {
      m1[fd] = &sync.Mutex{}
    }

    if v, ok := m1[fd]; ok {
      z.lock = v
      results = append(results, z)
      continue
    }

    var fileInfo, err = getFileInfo(z.File)

    if err != nil {
      z.lock = &sync.Mutex{}
      results = append(results, z)
      continue
    }

    if _, ok := m2[fileInfo]; !ok {
      m2[fileInfo] = &sync.Mutex{}
    }

    var mtx, _ = m2[fileInfo]
    z.lock = mtx
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
      Level: ll.TRACE,
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

  var l = &MultiLogger{
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
  }

  l.determineInitialLogLevels()
  return l
}

func (l *MultiLogger) SetEnvPrefix(s string) *MultiLogger {
  l.EnvPrefix = s
  return l
}

func (l *MultiLogger) SetToDisplayLocalTZ(f *os.File) *MultiLogger {
  // TODO: use lock to set this
  return l
}

func (l *MultiLogger) AddOutputFile(level ll.LogLevel, f *os.File) *MultiLogger {
  // TODO: use lock to set this
  l.Files = append(l.Files, &FileLevel{
    Level:   0,
    File:    f,
    Tags:    nil,
    lock:    nil,
    IsJSON:  false,
    isTrace: false,
    isDebug: false,
    isInfo:  false,
    isWarn:  false,
    isError: false,
  })
  return l
}

func (l *MultiLogger) SetMinLogLevel(f ll.LogLevel) *MultiLogger {
  return l
}

func (l *MultiLogger) SetMaxLogLevel(f ll.LogLevel) *MultiLogger {
  return l
}

func (l *MultiLogger) RemoveTag(s string) *MultiLogger {
  delete(*l.MetaFields.m, s)
  return l
}

func (l *MultiLogger) AddTag(s string, v interface{}) *MultiLogger {
  (*l.MetaFields.m)[s] = v
  return l
}

func (l *MultiLogger) AddMetaField(s string, v interface{}) *MultiLogger {
  (*l.MetaFields.m)[s] = v
  return l
}

func (l *MultiLogger) SetToJSONOutput() *MultiLogger {
  return l
}

func (l *MultiLogger) SetAppName(h string) *MultiLogger {
  l.AppName = h
  return l
}

func (l *MultiLogger) SetTimeZone(h time.Location) *MultiLogger {
  l.TimeZone = h
  return l
}

func (l *MultiLogger) SetHostName(h string) *MultiLogger {
  l.HostName = h
  return l
}

func (l *MultiLogger) determineInitialLogLevels() {
  ///
  l.isTrace = false
  l.isDebug = false
  l.isInfo = false
  l.isWarn = false
  l.isError = true // the special one:

  if len(l.Files) < 1 {

    l.isTrace = true
    l.isDebug = true
    l.isInfo = true
    l.isWarn = true
    l.isError = true

    l.Files = append(l.Files, &FileLevel{
      Level:   ll.TRACE,
      File:    os.Stdout,
      lock:    &sync.Mutex{},
      isTrace: true,
      isDebug: true,
      isInfo:  true,
      isWarn:  true,
      isError: true,
    })

    return
  }

  for _, v := range l.Files {

    // we always log errors
    v.isError = true

    switch v.Level {
    case ll.WARN:
      v.isWarn = true
      l.isWarn = true
    case ll.INFO:
      v.isWarn = true
      l.isWarn = true
      v.isInfo = true
      l.isInfo = true
    case ll.DEBUG:
      v.isWarn = true
      l.isWarn = true
      v.isInfo = true
      l.isInfo = true
      v.isDebug = true
      l.isDebug = true
    case ll.TRACE:
      v.isWarn = true
      l.isWarn = true
      v.isInfo = true
      l.isInfo = true
      v.isDebug = true
      l.isDebug = true
      v.isTrace = true
      l.isTrace = true
    default:
      panic("should have a log-level chosen")
    }

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

func (l *MultiLogger) NewLoggerWithLock() (*MultiLogger, func()) {
  shared.M1.Lock()
  defer shared.M1.Unlock()
  var newLck = &sync.Mutex{}
  newLck.Lock()
  var id = uuid.New().String()
  lockStack.Push(&stack.StackItem{
    Id:  id,
    Lck: newLck,
  })
  var z = MultiLogger{
    AppName:    l.AppName,
    HostName:   l.HostName,
    TimeZone:   l.TimeZone,
    MetaFields: l.MetaFields,
    LockUuid:   id,
  }
  return &z, z.unlock
}

func (l *MultiLogger) unlock() {

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

func (l *MultiLogger) Child(m *map[string]interface{}) *MultiLogger {

  var z = make(map[string]interface{})
  for k, v := range *l.MetaFields.m {
    z[k] = hlpr.CopyAndDereference(v)
  }

  for k, v := range *m {
    z[k] = hlpr.CopyAndDereference(v)
  }

  return &MultiLogger{
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

func (l *MultiLogger) Create(m *map[string]interface{}) *MultiLogger {
  return l.Child(m)
}

func (l *MultiLogger) getPrettyString(level ll.LogLevel, m *MetaFields, args *[]interface{}) string {

  date := time.Now().UTC().String()[11:25] // only first 25 chars
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
  b.WriteString(au.Col.Italic(l.AppName).String())
  b.WriteString(" ")

  size := 0

  for _, v := range *args {

    var primitive = true

    val := reflect.ValueOf(v)
    var kind = reflect.TypeOf(v).Kind()

    if kind == reflect.Ptr || kind == reflect.Interface {
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
  if _, err := fmt.Fprintln(os.Stderr, args...); err != nil {
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

  date := time.Now().UTC().String()
  date = date[:26]
  var strLevel = shared.LevelToString[level]
  var pid = shared.PID

  if mf == nil {
    mf = NewMetaFields(&MF{})
  }

  shared.StdioPool.Run(func(g *sync.WaitGroup) {

    // TODO - see if manually created JSON is faster
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
        //debug.PrintStack()
        cleaned = append(cleaned, c)
      }

      buf, err = json.Marshal([8]interface{}{"@bunion:v1", l.AppName, level, pid, l.HostName, date, mf.m, cleaned})

      if err != nil {
        fmt.Println(errors.New("Json-Logging: could not marshal the slice: " + err.Error()))
        return
      }
    }

    for _, v := range l.Files {

      switch level {

      case ll.TRACE:
        if !v.isTrace {
          continue
        }

      case ll.DEBUG:
        if !v.isDebug {
          continue
        }

      case ll.INFO:
        if !v.isInfo {
          continue
        }

      case ll.WARN:
        if !v.isWarn {
          continue
        }
      }

      v.lock.Lock()

      if _, err := v.File.Write(buf); err != nil {
        l.writeToStderr("1944431c-d90f-4e41-975f-206da000d85d", err)
      }
      if _, err := v.File.Write([]byte("\n")); err != nil {
        l.writeToStderr("ea20aee7-862d-4596-8639-52073c835757", err)
      }

      v.lock.Unlock()

    }
  })

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

func (l *MultiLogger) RawJSON(args ...interface{}) {
  // raw = no newlines, no spaces
  for i := 0; i < len(args); i++ {

    v, err := json.Marshal(args[i])

    if err != nil {
      panic(err)
    }

    os.Stdout.Write(v)
  }
}

func (l *MultiLogger) getMetaFields(args *[]interface{}) (*MetaFields, []interface{}) {
  var newArgs = []interface{}{}
  var m = MF{}
  var mf = NewMetaFields(&m)

  for k, v := range *l.MetaFields.m {
    m[k] = v
  }

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
  if l.isInfo {
    var meta, newArgs = l.getMetaFields(&args)
    l.writeSwitch(ll.INFO, meta, &newArgs)
  }
}

func (l *MultiLogger) Warn(args ...interface{}) {
  if l.isWarn {
    var meta, newArgs = l.getMetaFields(&args)
    l.writeSwitch(ll.WARN, meta, &newArgs)
  }
}

func (l *MultiLogger) Error(args ...interface{}) {
  var meta, newArgs = l.getMetaFields(&args)
  filteredStackTrace := hlpr.GetFilteredStacktrace()
  newArgs = append(newArgs, StackTrace{filteredStackTrace})
  l.writeSwitch(ll.ERROR, meta, &newArgs)
}

func (l *MultiLogger) Debug(args ...interface{}) {
  if l.isDebug {
    var meta, newArgs = l.getMetaFields(&args)
    l.writeSwitch(ll.DEBUG, meta, &newArgs)
  }

}

func (l *MultiLogger) Trace(args ...interface{}) {
  if l.isTrace {
    var meta, newArgs = l.getMetaFields(&args)
    l.writeSwitch(ll.TRACE, meta, &newArgs)
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
  filteredStackTrace := hlpr.GetFilteredStacktrace()
  formattedString := fmt.Sprintf(s, args...)
  l.writeSwitchForFormattedString(ll.ERROR, nil, &[]interface{}{formattedString, StackTrace{filteredStackTrace}})
}

func (l *MultiLogger) WarnF(s string, args ...interface{}) {
  if l.isWarn {
    l.writeSwitchForFormattedString(ll.WARN, nil, &[]interface{}{fmt.Sprintf(s, args...)})
  }
}

func (l *MultiLogger) InfoF(s string, args ...interface{}) {
  if l.isInfo {
    l.writeSwitchForFormattedString(ll.INFO, nil, &[]interface{}{fmt.Sprintf(s, args...)})
  }
}

func (l *MultiLogger) DebugF(s string, args ...interface{}) {
  if l.isDebug {
    l.writeSwitchForFormattedString(ll.DEBUG, nil, &[]interface{}{fmt.Sprintf(s, args...)})
  }
}

func (l *MultiLogger) TraceF(s string, args ...interface{}) {
  if l.isTrace {
    l.writeSwitchForFormattedString(ll.TRACE, nil, &[]interface{}{fmt.Sprintf(s, args...)})
  }
}

func (l *MultiLogger) NewLine() {
  for _, n := range l.Files {
    n.lock.Lock()
    n.File.Write([]byte("\n"))
    n.lock.Unlock()
  }
}

func (l *MultiLogger) Spaces(num int32) {
  for _, n := range l.Files {
    n.lock.Lock()
    n.File.Write([]byte(strings.Join(make([]string, num), " ")))
    n.lock.Unlock()
  }
}

func (l *MultiLogger) Tabs(num int32) {
  for _, n := range l.Files {
    n.lock.Lock()
    n.File.Write([]byte(strings.Join(make([]string, num), "\t")))
    n.lock.Unlock()
  }
}

func (l *MultiLogger) JustStdout(args ...interface{}) {
  safeStdout.Lock()
  for _, a := range args {
    v := fmt.Sprintf("((%T) %#v) ", a, a)
    safeStdout.Write([]byte(v))
  }
  safeStdout.Write([]byte("\n"))
  safeStdout.Unlock()
}

func (l *MultiLogger) PlainStdout(args ...interface{}) {

  go func() {
    for _, n := range l.Files {
      n.lock.Lock()
      for _, a := range args {
        v := fmt.Sprintf("((%T) %#v) ", a, a)
        n.File.Write([]byte(v))
      }
      n.File.Write([]byte("\n"))
      n.lock.Unlock()
    }
  }()

}

func (l *MultiLogger) PlainStderr(args ...interface{}) {
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
    Level: ll.TRACE,
    File:  os.Stdout,
  }},
)

func init() {
  //log.SetFlags(log.LstdFlags | log.Llongfile)
}
