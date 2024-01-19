package lib

import (
  "encoding/json"
  "errors"
  "fmt"
  uuid "github.com/google/uuid"
  "github.com/logrusorgru/aurora"
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
  "time"
)

func writeToStderr(args ...interface{}) {
  safeStderr.Lock()
  if _, err := fmt.Fprintln(os.Stderr, args...); err != nil {
    fmt.Println("adcca45f-8d7b-4d4a-8fd2-7683b7b375b5", "could not write to stderr:", err)
  }
  safeStderr.Unlock()
}

var safeStdout = writer.NewSafeWriter(os.Stdout)
var safeStderr = writer.NewSafeWriter(os.Stderr)

var lockStack = stack.NewStack()

type Logger struct {
  AppName       string
  IsLoggingJSON bool
  HostName      string
  ForceJSON     bool
  ForceNonJSON  bool
  TimeZone      *time.Location
  MetaFields    *MetaFields
  LockUuid      string
  EnvPrefix     string
  LogLevel      ll.LogLevel
  File          *os.File
  IsShowLocalTZ bool
}

type LoggerParams struct {
  AppName       string
  IsLoggingJSON bool
  HostName      string
  ForceJSON     bool
  ForceNonJSON  bool
  MetaFields    *MetaFields
  TimeZone      *time.Location
  LockUuid      string
  EnvPrefix     string
  LogLevel      ll.LogLevel
  File          *os.File
  IsShowLocalTZ bool
}

func NewLogger(p LoggerParams) *Logger {

  file := p.File

  if file == nil {
    file = os.Stdout
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

  var isLoggingJson = !shared.IsTerminal

  if p.ForceJSON {
    isLoggingJson = true
  }

  if os.Getenv("jlog_log_json") == "no" {
    isLoggingJson = false
  }

  if os.Getenv("jlog_log_json") == "yes" {
    if p.ForceJSON {
      writeToStderr("forceJSON:true was used, but the 'jlog_log_json' env var was set to 'yes'.")
    }
    isLoggingJson = true
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
    File:          file,
  }
}

func NewBasicLogger(AppName string, envTokenPrefix string, level ll.LogLevel) *Logger {
  return NewLogger(LoggerParams{
    AppName:   AppName,
    EnvPrefix: envTokenPrefix,
    LogLevel:  level,
  })
}

func CreateLogger(AppName string) *Logger {
  return NewLogger(LoggerParams{
    AppName: AppName,
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
  Val string
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
  shared.M1.Lock()
  defer shared.M1.Unlock()
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

func (l *Logger) SetEnvPrefix(s string) *Logger {
  l.EnvPrefix = s

  if len(l.EnvPrefix) > 0 {
    for _, env := range os.Environ() {
      parts := strings.SplitN(env, "=", 2)
      key := parts[0]
      value := parts[1]
      if strings.HasPrefix(key, l.EnvPrefix) {
        result := strings.TrimPrefix(key, l.EnvPrefix)
        (*l.MetaFields.m)[result] = value
      }
    }
  }

  return l
}

func (l *Logger) SetToDisplayUTC() *Logger {
  // TODO: use lock to set this
  l.IsShowLocalTZ = false
  return l
}

func (l *Logger) SetToUseTZ() *Logger {
  // TODO: use lock to set this
  l.IsShowLocalTZ = false
  return l
}

func (l *Logger) SetToDisplayLocalTZ() *Logger {
  // TODO: use lock to set this
  l.IsShowLocalTZ = true
  return l
}

func (l *Logger) SetOutputFile(f *os.File) *Logger {
  // TODO: use lock to set this
  l.File = f
  return l
}

func (l *Logger) SetLogLevel(f ll.LogLevel) *Logger {
  l.LogLevel = f
  return l
}

func (l *Logger) AddMetaField(s string, v interface{}) *Logger {
  (*l.MetaFields.m)[s] = v
  return l
}

func (l *Logger) SetToJSONOutput() *Logger {
  l.IsLoggingJSON = true
  return l
}

func (l *Logger) SetAppName(h string) *Logger {
  l.AppName = h
  return l
}

func (l *Logger) SetTimeZone(h time.Location) *Logger {
  l.TimeZone = &h
  return l
}

func (l *Logger) SetHostName(h string) *Logger {
  l.HostName = h
  return l
}

func (l *Logger) unlock() {

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
    z[k] = hlpr.CopyAndDereference(v)
  }

  for k, v := range *m {
    z[k] = hlpr.CopyAndDereference(v)
  }

  return &Logger{
    AppName:       l.AppName,
    IsLoggingJSON: l.IsLoggingJSON,
    HostName:      l.HostName,
    ForceJSON:     l.ForceJSON,
    ForceNonJSON:  l.ForceNonJSON,
    TimeZone:      l.TimeZone,
    MetaFields:    NewMetaFields(&z),
    LockUuid:      l.LockUuid,
    EnvPrefix:     l.EnvPrefix,
    LogLevel:      l.LogLevel,
  }
}

type SprintFStruct struct {
  SprintF string
}

func (l *Logger) Create(m *map[string]interface{}) *Logger {
  return l.Child(m)
}

func (l *Logger) writeToFile(time time.Time, level ll.LogLevel, m *MetaFields, args *[]interface{}) {
  b := l.getPrettyString(time, level, m, args)
  shared.M1.Lock()
  l.File.WriteString(b.String())
  shared.M1.Unlock()
  // _, err := io.Copy(l.File, b.)  // TODO: copy to file, instead of buffering b.String()
}

func (l *Logger) getPrettyString(time time.Time, level ll.LogLevel, m *MetaFields, args *[]interface{}) *strings.Builder {

  var b strings.Builder
  date := time.UTC().String()[11:25] // only first 25 chars

  if l.IsShowLocalTZ {
    if l.TimeZone != nil {
      date = time.UTC().In(l.TimeZone).String()[11:25]
    } else {
      date = time.Local().String()[11:25] // only first 25 chars
    }
  }

  stylizedLevel := "<undefined>"

  switch level {

  case ll.ERROR:
    stylizedLevel = aurora.Underline(aurora.Bold(aurora.Red("ERROR"))).String()
    break

  case ll.WARN:
    stylizedLevel = aurora.Magenta("WARN").String()
    break

  case ll.DEBUG:
    stylizedLevel = aurora.Bold("DEBUG").String()
    break

  case ll.INFO:
    stylizedLevel = aurora.Gray(12, "INFO").String()
    break

  case ll.TRACE:
    stylizedLevel = aurora.Gray(4, "TRACE").String()
    break
  }

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

    if v == nil {
      b.WriteString(fmt.Sprintf("<nil> - (%T)", v))
      continue
    }

    if &v == nil {
      b.WriteString(fmt.Sprintf("<nil> (%T)", v))
      continue
    }

    val := reflect.ValueOf(v)
    var t = reflect.TypeOf(v)
    var kind = t.Kind()

    if kind == reflect.Ptr {
      //v = Val.Elem().Interface()
      //Val = reflect.ValueOf(v)
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
      writeToStderr("771c710b-aba2-46ef-9126-c26d3dfe7925", err)
    }

    if !primitive && (level == ll.TRACE || level == ll.DEBUG) {

      if _, err := b.WriteString("\n"); err != nil {
        writeToStderr("18614292-658f-42a5-81e7-593e941ea857", err)
      }

      if _, err := b.WriteString(fmt.Sprintf("info as sprintf: %+v", v)); err != nil {
        writeToStderr("2a795ef2-65bb-4a03-9808-b072e4497d73", err)
      }

      b.Write([]byte("json:"))
      if x, err := json.Marshal(v); err == nil {
        if _, err := b.Write(x); err != nil {
          writeToStderr("err:56831878-8d63-45f4-905b-d1b3bbac2152:", err)
        }
      } else {
        writeToStderr("err:70bf10e0-6e69-4a3b-bf64-08f6d20c4580:", err)
      }

    }

  }

  if _, err := b.WriteString("\n"); err != nil {
    writeToStderr("f834d14a-9735-4fd6-9389-f79144044746", err)
  }

  return &b
}

func (l *Logger) writeJSON(time time.Time, level ll.LogLevel, mf *MetaFields, args *[]interface{}) {

  date := time.UTC().String()

  if l.IsShowLocalTZ {
    if l.TimeZone != nil {
      date = time.UTC().In(l.TimeZone).String()
    } else {
      date = time.Local().String() // only first 25 chars
    }
  }

  date = date[:26]
  var strLevel = shared.LevelToString[level]
  var pid = shared.PID

  if mf == nil {
    mf = NewMetaFields(&MF{})
  }

  //shared.StdioPool.Run(func(g *sync.WaitGroup) {

  var wg = sync.WaitGroup{}
  wg.Add(1)

  shared.StdioPool.Run(func(g *sync.WaitGroup) {

    if g != nil {
      defer g.Done()
    }

    defer wg.Done()

    // TODO: maybe manually generating JSON is better? prob not worth it
    buf, err := json.Marshal([8]interface{}{"@bunion:v1", l.AppName, strLevel, pid, l.HostName, date, mf.m, args})

    if err != nil {

      _, file, line, _ := runtime.Caller(3)
      DefaultLogger.Warn("could not marshal the slice:", err.Error(), "file://"+file+":"+strconv.Itoa(line))

      //cleaned := make([]interface{},0)

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

    shared.M1.Lock()
    safeStdout.Write(buf)
    safeStdout.Write([]byte("\n"))
    shared.M1.Unlock()
  })

  wg.Wait()

}

func (l *Logger) writeSwitch(time time.Time, level ll.LogLevel, m *MetaFields, args *[]interface{}) {
  if l.IsLoggingJSON {
    l.writeJSON(time, level, m, args)
  } else {
    l.writeToFile(time, level, m, args)
    // l.getPrettyString(time, level, m, args)
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

type Stringer interface {
  String() string
}

type LogItem struct {
  AsString  string
  ErrString string
  Value     interface{}
}

func getInspectableVal(obj interface{}, depth int) interface{} {
  ///
  val := reflect.ValueOf(obj)
  v := val.Interface()

  if val.Kind() == reflect.Ptr {
    val = val.Elem()
    v = val.Interface()
  }

  if val.Kind() != reflect.Struct {
    return obj
  }

  var errStr = ""
  var toString = ""

  if z, ok := v.(error); ok {
    errStr = z.Error()
  }

  if z, ok := v.(Stringer); ok {
    toString = z.String()
  }

  result := make(map[string]interface{})

  if errStr != "" {
    result["@ErrString"] = errStr
  }

  if errStr != "" {
    result["@ToString"] = toString
  }

  typ := val.Type()

  for i := 0; i < val.NumField(); i++ {
    field := val.Field(i)
    fieldName := typ.Field(i).Name

    for true {
      if field.IsValid() && field.CanInterface() {

        inf := field.Interface()
        var errStr = ""
        var toString = ""

        if z, ok := inf.(error); ok {
          errStr = z.Error()
        }

        if z, ok := inf.(Stringer); ok {
          toString = z.String()
        }

        if errStr == "" && toString == "" {
          result[fieldName] = inf
        } else {
          result[fieldName] = LogItem{
            AsString:  toString,
            ErrString: errStr,
            Value:     inf,
          }
        }

        break
      }
      //result[fieldName] = field.Elem().Interface()
      //result[fieldName] = field.Kind()
      if field.Kind() == reflect.Ptr {
        field = field.Elem()
      } else {
        result[fieldName] = fmt.Sprintf("%v (%s)", field, field.String())
        break
      }

    }

  }

  return result
}

func (l *Logger) getMetaFields(args *[]interface{}) (*MetaFields, []interface{}) {
  ////
  var newArgs = []interface{}{}
  var m = MF{}
  var mf = NewMetaFields(&m)

  for k, v := range *l.MetaFields.m {
    m[k] = v
  }

  for _, x := range *args {
    if z, ok := x.(MetaFields); ok {
      for k, v := range *z.m {
        m[k] = getInspectableVal(v, 0)
      }
    } else if z, ok := x.(*MetaFields); ok {
      for k, v := range *z.m {
        m[k] = getInspectableVal(v, 0)
      }
    } else if z, ok := x.(*LogId); ok {
      m["log_id"] = z.Val
    } else if z, ok := x.(LogId); ok {
      m["log_id"] = z.Val
    } else {
      newArgs = append(newArgs, getInspectableVal(x, 0))
    }
  }

  return mf, newArgs
}

func (l *Logger) Info(args ...interface{}) {
  switch l.LogLevel {
  case ll.WARN, ll.ERROR:
    return
  }
  t := time.Now()
  var meta, newArgs = l.getMetaFields(&args)
  l.writeSwitch(t, ll.INFO, meta, &newArgs)
}

func (l *Logger) Warn(args ...interface{}) {
  switch l.LogLevel {
  case ll.ERROR:
    return
  }
  t := time.Now()
  var meta, newArgs = l.getMetaFields(&args)
  l.writeSwitch(t, ll.WARN, meta, &newArgs)
}

func (l *Logger) Error(args ...interface{}) {
  t := time.Now()
  var meta, newArgs = l.getMetaFields(&args)
  filteredStackTrace := hlpr.GetFilteredStacktrace()
  newArgs = append(newArgs, StackTrace{filteredStackTrace})
  l.writeSwitch(t, ll.ERROR, meta, &newArgs)
}

func (l *Logger) Debug(args ...interface{}) {
  switch l.LogLevel {
  case ll.INFO, ll.WARN, ll.ERROR:
    return
  }
  t := time.Now()
  var meta, newArgs = l.getMetaFields(&args)
  l.writeSwitch(t, ll.DEBUG, meta, &newArgs)
}

func (l *Logger) Trace(args ...interface{}) {
  switch l.LogLevel {
  case ll.DEBUG, ll.INFO, ll.WARN, ll.ERROR:
    return
  }
  t := time.Now()
  var meta, newArgs = l.getMetaFields(&args)
  l.writeSwitch(t, ll.TRACE, meta, &newArgs)
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

func (l *Logger) TagPair(k string, v interface{}) *Logger {
  var z = map[string]interface{}{k: v}
  return l.Child(&z)
}

func (l *Logger) Tags(z *map[string]interface{}) *Logger {
  return l.Create(z)
}

func (l *Logger) InfoF(s string, args ...interface{}) {
  switch l.LogLevel {
  case ll.WARN, ll.ERROR:
    return
  }
  t := time.Now()
  l.writeSwitch(t, ll.INFO, nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) WarnF(s string, args ...interface{}) {
  switch l.LogLevel {
  case ll.ERROR:
    return
  }
  t := time.Now()
  l.writeSwitch(t, ll.WARN, nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

type StackTrace struct {
  ErrorTrace *[]string
}

func (l *Logger) ErrorF(s string, args ...interface{}) {
  t := time.Now()
  filteredStackTrace := hlpr.GetFilteredStacktrace()
  formattedString := fmt.Sprintf(s, args...)
  l.writeSwitch(t, ll.ERROR, nil, &[]interface{}{formattedString, StackTrace{filteredStackTrace}})
}

func (l *Logger) DebugF(s string, args ...interface{}) {
  switch l.LogLevel {
  case ll.INFO, ll.WARN, ll.ERROR:
    return
  }
  t := time.Now()
  l.writeSwitch(t, ll.DEBUG, nil, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) TraceF(s string, args ...interface{}) {
  switch l.LogLevel {
  case ll.DEBUG, ll.INFO, ll.WARN, ll.ERROR:
    return
  }
  t := time.Now()
  l.writeSwitch(t, ll.TRACE, nil, &[]interface{}{fmt.Sprintf(s, args...)})
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

var DefaultLogger = CreateLogger("Default").
  SetLogLevel(ll.TRACE)

func init() {

  //log.SetFlags(log.LstdFlags | log.Llongfile)

}
