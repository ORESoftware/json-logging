package lib

import (
  "encoding/base64"
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
  "math"
  "os"
  "reflect"
  "runtime"
  "runtime/debug"
  "sort"
  "strconv"
  "strings"
  "sync"
  "time"
  // "unsafe"
  "unsafe"
  "github.com/logrusorgru/aurora/v4"
  //jsoniter "github.com/json-iterator/go"
  "github.com/mailru/easyjson"
  "bytes"
)

// TODO: use a better json lib for encoding?
//var jsn = jsoniter.ConfigCompatibleWithStandardLibrary

func writeToStderr(args ...interface{}) {
  safeStderr.Lock()
  if _, err := fmt.Fprintln(os.Stderr, args...); err != nil {
    fmt.Println("adcca45f-8d7b-4d4a-8fd2-7683b7b375b5", "could not write to stderr:", err)
  }
  safeStderr.Unlock()
}

func isBase64Perf(s string) bool {
  if len(s) > 0 && s[len(s)-1] == '=' {
    return true
  }
  return false
}

func isBase64(s string) bool {
  _, err := base64.StdEncoding.DecodeString(s)
  return err == nil
}

// base64ToString decodes a base64-encoded string to a regular string
func base64ToString(s string) (string, error) {
  decodedBytes, err := base64.StdEncoding.DecodeString(s)
  if err != nil {
    return "", err
  }
  return string(decodedBytes), nil
}

var safeStdout = writer.NewSafeWriter(os.Stdout)
var safeStderr = writer.NewSafeWriter(os.Stderr)

var lockStack = stack.NewStack()

type Logger struct {
  Mtx           sync.Mutex
  AppName       string
  IsLoggingJSON bool
  HighPerf      bool
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
    Mtx:           sync.Mutex{},
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

// func NewLogger(AppName string, forceJSON bool, hostName string, envTokenPrefix string) *Logger {
//	return New(AppName, forceJSON, hostName, envTokenPrefix)
// }

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

func (x *LogId) GetLogId(isHyperLink bool) string {
  // fmt.Println("\\e]8;;http://example.com\aThis is the link\\e]8;;\\e\\")
  // fmt.Println(fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", "https://linkedin.com", "Go to Linked"))

  // fmt.Println(fmt.Sprintf("\033]8;;%s\033\\%s\033]8;;\033\\", "Go to x", "Go to XX"))

  if false && isHyperLink {
    // return au.Col.Blue("xyz1").Hyperlink(fmt.Sprintf("foobarbas", x.Val)).HyperlinkTarget()
    return au.Col.Hyperlink("foo", "https://foo.com").String()
    return au.Col.Hyperlink("foo", "https://foo.com").HyperlinkTarget()
    // return au.Col.Blue("(Goto -> LogId)").Hyperlink(fmt.Sprintf("http://vibeirl.com/dev/links?%s", x.Val)).HyperlinkTarget()
  } else {
    // return fmt.Sprintf("(log-id:'%s')", x.Val)
    return x.Val
  }
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
    Mtx:           sync.Mutex{},
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

func (l *Logger) SetHighPerf(b bool) *Logger {
  l.Mtx.Lock()
  defer l.Mtx.Unlock()
  l.HighPerf = b
  return l
}

func (l *Logger) SetEnvPrefix(s string) *Logger {
  l.Mtx.Lock()
  defer l.Mtx.Unlock()

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
  l.Mtx.Lock()
  defer l.Mtx.Unlock()
  l.IsShowLocalTZ = false
  return l
}

func (l *Logger) SetToUseTZ() *Logger {
  // TODO: use lock to set this
  l.Mtx.Lock()
  defer l.Mtx.Unlock()
  l.IsShowLocalTZ = false
  return l
}

func (l *Logger) SetToDisplayLocalTZ() *Logger {
  // TODO: use lock to set this
  l.Mtx.Lock()
  defer l.Mtx.Unlock()
  l.IsShowLocalTZ = true
  return l
}

func (l *Logger) SetOutputFile(f *os.File) *Logger {
  // TODO: use lock to set this
  l.Mtx.Lock()
  defer l.Mtx.Unlock()
  l.File = f
  return l
}

func (l *Logger) SetLogLevel(f ll.LogLevel) *Logger {
  l.Mtx.Lock()
  defer l.Mtx.Unlock()
  l.LogLevel = f
  return l
}

func (l *Logger) AddMetaField(s string, v interface{}) *Logger {
  l.Mtx.Lock()
  defer l.Mtx.Unlock()
  (*l.MetaFields.m)[s] = v
  return l
}

func (l *Logger) SetToJSONOutput() *Logger {
  l.Mtx.Lock()
  defer l.Mtx.Unlock()
  l.IsLoggingJSON = true
  return l
}

func (l *Logger) SetAppName(h string) *Logger {
  l.Mtx.Lock()
  defer l.Mtx.Unlock()
  l.AppName = h
  return l
}

func (l *Logger) SetTimeZone(h time.Location) *Logger {
  l.Mtx.Lock()
  defer l.Mtx.Unlock()
  l.TimeZone = &h
  return l
}

func (l *Logger) SetHostName(h string) *Logger {
  l.Mtx.Lock()
  defer l.Mtx.Unlock()
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

  l.Mtx.Lock()
  defer l.Mtx.Unlock()

  var z = make(map[string]interface{})
  for k, v := range *l.MetaFields.m {
    z[k] = hlpr.CopyAndDereference(v)
  }

  for k, v := range *m {
    z[k] = hlpr.CopyAndDereference(v)
  }

  return &Logger{
    Mtx:           sync.Mutex{},
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

func getLastXChars(count int, s interface{}) string {
  // Check if the string length is less than 13

  if s, ok := s.(string); ok {
    if len(s) < count {
      return s // Return the original string if it's too short
    }
    // Return the last 13 characters
    return s[len(s)-count:]
  }

  return "<(not a string)>"
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

  b.WriteString(au.Col.Gray(9, date).String())
  b.WriteString(" ")
  b.WriteString(stylizedLevel)
  b.WriteString(" ")
  b.WriteString(au.Col.Gray(12, "app:").String())
  b.WriteString(au.Col.Italic(l.AppName).String())
  b.WriteString(" ")
  if v, ok := (*m.m)["log_id"]; ok {
    //b.WriteString(fmt.Sprintf("(log-id:%s) ", v))
    b.WriteString(fmt.Sprintf("(%s%s) ", aurora.Bold("log-id:").String(), getLastXChars(12, v)))
  }

  if v, ok := (*m.m)["log_num"]; ok {
    //b.WriteString(fmt.Sprintf("(log-id:%s) ", v))
    b.WriteString(fmt.Sprintf("(%s%v) ", aurora.Bold("log-num:").String(), v))
  }

  size := 0

  for _, v := range *args {

    var primitive = true

    if v == nil {
      b.WriteString(fmt.Sprintf("<nil 1> - (%T)", v))
      continue
    }

    if &v == nil {
      b.WriteString(fmt.Sprintf("<nil 2> (%T)", v))
      continue
    }

    var rv = reflect.ValueOf(v)
    var t = reflect.TypeOf(v)
    var kind = t.Kind()

    if !rv.IsValid() {
      b.WriteString(fmt.Sprintf("%v", v))
      continue
    }

    var counter = 0

    for {

      if !(kind == reflect.Ptr || kind == reflect.Interface) {
        break
      }

      if !rv.IsNil() {
        b.WriteString(fmt.Sprintf("%v", v))
        break
      }

      if !rv.IsValid() { // Check if the dereferenced value is valid
        v = nil
        break
      }

      rv = rv.Elem()

      if !rv.IsValid() { // Check if the dereferenced value is valid
        v = nil
        break
      }

      t = rv.Type()
      kind = rv.Kind()
      v = rv.Interface()
      if counter++; counter > 6 {
        break
      }

    }

    if !rv.IsValid() {
      b.WriteString(fmt.Sprintf("(%v)) - (%T)", v, v))
    }

    if kind == reflect.Ptr || kind == reflect.Interface {
      b.WriteString(fmt.Sprintf("(%v) - (%T)", v, v))
      continue
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

func marshalCustomArray2(arr []interface{}) ([]byte, error) {
  var b bytes.Buffer
  b.WriteByte('[')

  for i, v := range arr {
    var buf []byte
    var err error

    // Check if the value implements easyjson.Marshaler
    if m, ok := v.(easyjson.Marshaler); ok {
      buf, err = easyjson.Marshal(m)
      fmt.Println(fmt.Sprintf(`["20095e96-d393-456a-9a9a-3e0c47ce760c","%s"]`, err.Error()));
    } else {
      // Fallback to encoding/json
      buf, err = json.Marshal(v)
    }

    if err != nil {
      return nil, err
    }

    b.Write(buf)

    // Add comma between elements, but not after the last element
    if i < len(arr)-1 {
      b.WriteByte(',')
    }
  }

  b.WriteByte(']')

  return b.Bytes(), nil
}

var bufferPool = sync.Pool{
  New: func() interface{} {
    return new(bytes.Buffer)
  },
}

var bufferPool2 = sync.Pool{
  New: func() interface{} {
    return new(bytes.Buffer)
  },
}

var bufferPool3 = sync.Pool{
  New: func() interface{} {
    return new(bytes.Buffer)
  },
}

func marshalToJSON(v interface{}) ([]byte, error) {
  buf := bufferPool.Get().(*bytes.Buffer)
  buf.Reset()
  defer bufferPool.Put(buf)

  encoder := json.NewEncoder(buf)
  if err := encoder.Encode(v); err != nil {
    return nil, err
  }

  // Need to remove the trailing newline added by Encode
  if buf.Len() > 0 {
    buf.Truncate(buf.Len() - 1)
  }

  return buf.Bytes(), nil
}

func marshalCustomArray(arr []interface{}) ([]byte, error) {
  ///
  b := bufferPool.Get().(*bytes.Buffer)
  b.Reset() // Reset the buffer for reuse
  defer bufferPool.Put(b)

  b.WriteByte('[')

  for i, v := range arr {
    var buf []byte
    var err error

    // Check if the value implements easyjson.Marshaler
    if m, ok := v.(easyjson.Marshaler); ok {
      buf, err = easyjson.Marshal(m)
    } else {
      // Fallback to encoding/json
      buf, err = json.Marshal(v)
    }

    if err != nil {
      return nil, err
    }

    b.Write(buf)

    // Add comma between elements, but not after the last element
    if i < len(arr)-1 {
      b.WriteByte(',')
    }
  }

  b.WriteByte(']')

  return b.Bytes(), nil // Convert buffer to bytes and return
}

var ioChan = make(chan func())
var ioChan2 = make(chan func())

func init(){
  go func() {
    for {
      fn := <- ioChan
      fn()
    }
  }()

  go func() {
    for {
      fn := <- ioChan2
      fn()
    }
  }()

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

  // TODO: use a single channel for all calls, instead of different waitgroups per call?
  //var wg = sync.WaitGroup{}
  //wg.Add(1)

  // shared.StdioPool.Run(func(g *sync.WaitGroup) {

  ioChan <- (func() {

    //defer wg.Done()

    // TODO: maybe manually generating JSON is better? prob not worth it
    //buf, err := json.Marshal([8]interface{}{"@bunion:v1", l.AppName, strLevel, pid, l.HostName, date, mf.m, args})

    jj, err := json.Marshal(mf.m)

    jjj, err := marshalCustomArray(*args)

    buf := []byte(fmt.Sprintf(`["@bunion:v1","%s","%s","%d","%s","%s", %s, %s]`,
      l.AppName, strLevel, pid, l.HostName, date, string(jj), string(jjj)))

    if false {
      _ = func() []byte {
        buf := bufferPool3.Get().(*bytes.Buffer)
        buf.Reset()                // Reset the buffer for reuse
        defer bufferPool3.Put(buf) // Make sure to return the buffer to the pool

        // Use the buffer to construct the string
        buf.WriteString(`["@bunion:v1","`)
        buf.WriteString(l.AppName)
        buf.WriteString(`","`)
        buf.WriteString(strLevel)
        buf.WriteString(`",`)
        buf.WriteString(fmt.Sprint(pid))
        buf.WriteString(`,"`)
        buf.WriteString(l.HostName)
        buf.WriteString(`","`)
        buf.WriteString(date)
        buf.WriteString(`", `)
        buf.Write(jj) // Assuming jj is already a []byte representing JSON
        buf.WriteString(", ")
        buf.Write(jjj) // Assuming jjj is also a []byte representing JSON
        buf.WriteString("]")

        // Convert the buffer's contents to a []byte if needed
        result := buf.Bytes()

        return result
      }()
    }

    if err != nil {

      _, file, line, _ := runtime.Caller(3)
      DefaultLogger.Warn("json-logging: 1: could not marshal the slice:", err.Error(), "file://"+file+":"+strconv.Itoa(line))

      // cleaned := make([]interface{},0)

      var cache = map[*interface{}]*interface{}{}
      var cleaned = make([]interface{}, 0, len(*args))

      for i := 0; i < len(*args); i++ {
        // TODO: for now instead of cleanUp, we can ust fmt.Sprintf()
        v := &(*args)[i]
        c := hlpr.CleanUp(v, &cache)
        // debug.PrintStack()
        cleaned = append(cleaned, c)
      }

      buf, err = json.Marshal([8]interface{}{"@bunion:v1", l.AppName, level, pid, l.HostName, date, mf.m, cleaned})

      if err != nil {
        fmt.Println(errors.New("Json-Logging: 2: could not marshal the slice: " + err.Error()))
        return
      }

      // fmt.Println("json-logging: cleaned:", cleaned)
    }

    ioChan2 <- func() {
      shared.M1.Lock()
      // TODO: if the user selects stderr or non-stdout then need to lock on that
      safeStdout.Write(buf)
      safeStdout.Write([]byte("\n"))
      shared.M1.Unlock()
    }

  });

  //wg.Wait()
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

type ArrayVal struct {
  GoType      string
  TrueLen     int
  IsTruncated bool
  Val         []interface{}
}

type MapVal struct {
  GoType       string
  TrueKeyCount int
  IsTruncated  bool
  Val          map[string]interface{}
}

type EmptyVal struct {
  EmptyVal bool
}

func doMap(v interface{}, val reflect.Value) *MapVal {

  if x, ok := v.(MapVal); ok {
    return &x
  }

  if x, ok := v.(*MapVal); ok {
    return x
  }

  if !val.IsValid() {
    return nil
  }

  var z = MapVal{
    GoType:       "<unknown>",
    TrueKeyCount: 0,
    IsTruncated:  false,
    Val:          nil,
  }

  len := val.Len()
  z.TrueKeyCount = len
  z.GoType = val.Type().String()

  keyToRetrieve := "JLogMarker"

  // Get the value associated with the key
  // TODO: ??
  // panic: reflect.Value.MapIndex: value of type string is not assignable to type http.connectMethodKey
  keyValue := val.MapIndex(reflect.ValueOf(keyToRetrieve))

  // Check if the key exists
  if keyValue.IsValid() {
    // Print the value
    return &z
  }

  keys := val.MapKeys()

  min := int(math.Min(float64(len), float64(25)))
  if min < len {
    z.IsTruncated = true
  }

  z.Val = map[string]interface{}{}

  i := 0
  for _, key := range keys {
    if i++; i > min {
      break
    }
    var el = val.MapIndex(key)
    if el.IsValid() && el.CanInterface() {
      z.Val[fmt.Sprintf("%v", key)] = getInspectableVal(el.Interface(), el, 0, 1)
    } else {
      z.Val[fmt.Sprintf("%v", key)] = nil
    }
  }

  // for i := 0; i < min; i++ {
  //  el := val.Index(i)
  //  if el.IsValid() {
  //    z.Val[i] = getInspectableVal(el.Interface(), el, 0, 1)
  //  } else {
  //    // Handle the case where the value is nil
  //    z.Val[i] = nil // or any default value you want
  //  }
  // }

  return &z
}

func doArray(v interface{}, rv reflect.Value) *ArrayVal {

  if x, ok := v.(ArrayVal); ok {
    return &x
  }

  if x, ok := v.(*ArrayVal); ok {
    return x
  }

  var z = ArrayVal{
    TrueLen:     0,
    IsTruncated: false,
    Val:         nil,
    GoType:      "<unknown>",
  }

  len := rv.Len()
  z.TrueLen = len

  min := int(math.Min(float64(len), float64(40)))
  if min < len {
    z.IsTruncated = true
  }

  z.Val = make([]interface{}, min)

  for i := 0; i < min; i++ {
    el := rv.Index(i)
    if el.IsValid() {
      z.GoType = fmt.Sprintf("%s", el.Type().String())
      inf := el.Interface()
      // z.GoType = fmt.Sprintf("%T", inf)
      z.Val[i] = getInspectableVal(inf, el, 0, 1)
    } else {
      // Handle the case where the value is nil
      z.Val[i] = nil // or any default value you want
    }
  }

  // TODO: add the 3 last original elements to end of new list, if space permits

  // for i := 0; i < 3; i++ {
  //  z.Val = append(z.Val, EmptyVal{EmptyVal: true})
  // }
  //
  // var b = math.Max(3, float64(min-len))
  //
  // for i := int(b); i >= 0; i-- {
  //  z.Val = append(z.Val, getInspectableVal(rv.Index(len-1-i).Interface(), 0))
  // }

  return &z
}

type VibeInspectStr interface {
  ToString() string
}

type VibeInspectInt interface {
  ToInt() int
}

type VibeInspectBool interface {
  ToBool() bool
}

type UnkVal struct {
  GoType   string
  Val      interface{}
  ValAsStr string
}

func getInspectableVal(obj interface{}, rv reflect.Value, depth int, count int) interface{} {
  // /
  // var rv = reflect.ValueOf(obj)

  if count > 11 {
    return obj
  }

  if !rv.IsValid() {
    // Handle invalid reflection value (e.g., nil pointer)
    return nil
  }

  var v = rv.Interface()

  if v == nil {
    return v
  }

  if rv.Kind() == reflect.Ptr || rv.Kind() == reflect.Interface {

    if rv.IsNil() {
      // Handle nil interface value
      return nil
    }

    if !rv.IsValid() {
      return nil
    }

    rv = rv.Elem()

    if !rv.IsValid() {
      return nil
    }

    if rv.CanInterface() {
      return getInspectableVal(rv.Interface(), rv, depth, count+1)
    }

  }

  if !rv.CanInterface() {
    return v
  }

  v = rv.Interface()

  if v == nil {
    return nil
  }

  if x, ok := v.(string); ok {
    return x
  }

  if x, ok := v.(int); ok {
    return x
  }

  if x, ok := v.(bool); ok {
    return x
  }

  // if depth > 3 {
  //  return v
  // }

  if !rv.IsValid() {
    return nil
  }

  switch rv.Kind() {
  //
  case reflect.Slice:
    if rv.Type().Elem().Kind() == reflect.Uint8 {
      if z, ok := v.([]byte); ok {
        return string(z)
      }
      return v
    }
    return doArray(v, rv)

  case reflect.Array:
    if rv.Type().Elem().Kind() == reflect.Uint8 {
      if z, ok := v.([]byte); ok {
        return string(z)
      }
      return v
    }
    return doArray(v, rv)
  }

  if rv.Kind() == reflect.Func {
    return fmt.Sprintf("(func())")
  }

  if rv.Kind() == reflect.Chan {
    if rv.IsValid() && rv.CanInterface() {
      return fmt.Sprintf("(chan (%v) %v)", rv.Type(), rv.Interface())
    }
    return fmt.Sprintf("(chan (%v) (%v) %+v)", rv.Type(), v, v)
  }

  if rv.Kind() == reflect.Map {
    return doMap(v, rv)
  }

  if rv.Kind() != reflect.Struct {
    // if it's a not a struct now
    var str = fmt.Sprintf("(%v / %v)", v, rv.Type().String())
    var t = fmt.Sprintf("(%T / %v)", v, rv.Type().String())
    return &UnkVal{
      GoType:   t,
      Val:      v,
      ValAsStr: str,
    }
  }

  // it's a struct, so we can add metadata to it
  var errStr = ""
  var toString = ""

  var typeStr = fmt.Sprintf("%T", v)

  if rv.IsValid() {
    typ := rv.Type()
    z := typ.String()
    if z != typeStr {
      typeStr = fmt.Sprintf("(%s / %v / %s)", typeStr, typ, z)
    }
  }

  if z, ok := v.(error); ok {
    errStr = z.Error()
  }

  if z, ok := v.(Stringer); ok {
    toString = z.String()
  }

  outResult := make(map[string]interface{})

  if typeStr != "" {
    outResult["+(GoType):"] = typeStr
  }

  if errStr != "" {
    outResult["+(ErrStr):"] = errStr
  }

  if toString != "" && toString != errStr {
    outResult["+(ToStr):"] = toString
  }

  innerResult := make(map[string]interface{})
  outResult["+(Val):"] = innerResult

  typ := rv.Type()

  for i := 0; i < rv.NumField(); i++ {

    field := rv.Field(i)
    fieldName := typ.Field(i).Name

    if !field.IsValid() {
      innerResult[fieldName] = nil
      continue
    }

    j := 0

    for {

      if j++; j > 9 {
        // only try to deref so many times - perhaps it's a ptr to a ptr, etc
        break
      }

      if !(field.Kind() == reflect.Ptr || field.Kind() == reflect.Interface) {
        break
      }

      if !field.IsValid() {
        innerResult[fieldName] = nil
        break
      }

      if field.IsNil() {
        innerResult[fieldName] = nil
        break
      }

      field = field.Elem()

      if !field.IsValid() {
        innerResult[fieldName] = nil
        break
      }

    }

    if _, ok := innerResult[fieldName]; ok {
      continue
    }

    if field.Kind() == reflect.Interface || field.Kind() == reflect.Ptr {

      if field.IsNil() {
        innerResult[fieldName] = nil
        continue
      }

      if !field.IsValid() {
        innerResult[fieldName] = nil
        continue
      }

      field = field.Elem()

      if !field.IsValid() {
        innerResult[fieldName] = nil
        continue
      }

      return fmt.Sprintf("yyy %T // %v", field.Interface(), field.Interface())
    }

    if field.CanInterface() {
      innerResult[fieldName] = getInspectableVal(field.Interface(), field, depth+1, 1)
      continue
    }

    innerResult[fieldName] = fmt.Sprintf("%v (Type: %s)", field.String(), field.Type().String())
    continue

    // f := rv.FieldByName(fieldName)

    if field.CanAddr() {
      field = reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem()
      //field = reflect.NewAt(field.Type(), field.Addr().UnsafePointer())
    }

    if field.IsValid() && field.CanInterface() {
      innerResult[fieldName] = getInspectableVal(field.Interface(), field, depth+1, 1)
    } else {
      innerResult[fieldName] = fmt.Sprintf("%s", field.Type().String())
    }

    continue
  }

  return outResult
}

func (l *Logger) getMetaFields(args *[]interface{}) (*MetaFields, []interface{}) {
  // //
  var newArgs = []interface{}{}
  var mf = NewMetaFields(&MF{})

  for k, v := range *l.MetaFields.m {
    (*mf.m)[k] = v
  }

  var hasLogId = false

  for _, x := range *args {
    if z, ok := x.(MetaFields); ok {
      for k, v := range *z.m {
        (*mf.m)[k] = v
      }
    } else if z, ok := x.(*MetaFields); ok {
      for k, v := range *z.m {
        (*mf.m)[k] = v
      }
    } else if z, ok := x.(*LogId); ok {
      (*mf.m)["log_id"] = z.GetLogId(true)
      hasLogId = true
      // newArgs = append(newArgs, z.GetLogId(true))
    } else if z, ok := x.(LogId); ok {
      (*mf.m)["log_id"] = z.GetLogId(true)
      // newArgs = append(newArgs, z.GetLogId(true))
      hasLogId = true
    } else {

      if l.IsLoggingJSON && true || !l.HighPerf {
        var xx = reflect.ValueOf(x)
        newArgs = append(newArgs, getInspectableVal(x, xx, 0, 1))
      } else {
        newArgs = append(newArgs, x)
      }

    }
  }

  if false && !hasLogId {
    fmt.Println("missing log id:", string(debug.Stack()))
  }

  return mf, newArgs
}

func (l *Logger) Trace(args ...interface{}) {
  switch l.LogLevel {
  case ll.DEBUG, ll.INFO, ll.WARN, ll.ERROR, ll.CRITICAL:
    return
  }
  t := time.Now()
  n := shared.GetNextLogNum()
  var meta, newArgs = l.getMetaFields(&args)
  (*meta.m)["log_num"] = n
  l.writeSwitch(t, ll.TRACE, meta, &newArgs)
}

func (l *Logger) Debug(args ...interface{}) {
  switch l.LogLevel {
  case ll.INFO, ll.WARN, ll.ERROR, ll.CRITICAL:
    return
  }
  t := time.Now()
  n := shared.GetNextLogNum()
  var meta, newArgs = l.getMetaFields(&args)
  (*meta.m)["log_num"] = n
  l.writeSwitch(t, ll.DEBUG, meta, &newArgs)
}

func (l *Logger) Info(args ...interface{}) {
  switch l.LogLevel {
  case ll.WARN, ll.ERROR, ll.CRITICAL:
    return
  }
  t := time.Now()
  n := shared.GetNextLogNum()
  var meta, newArgs = l.getMetaFields(&args)
  (*meta.m)["log_num"] = n
  l.writeSwitch(t, ll.INFO, meta, &newArgs)
}

func (l *Logger) Warn(args ...interface{}) {
  switch l.LogLevel {
  case ll.ERROR, ll.CRITICAL:
    return
  }
  t := time.Now()
  n := shared.GetNextLogNum()
  var meta, newArgs = l.getMetaFields(&args)
  (*meta.m)["log_num"] = n
  l.writeSwitch(t, ll.WARN, meta, &newArgs)
}

func (l *Logger) Error(args ...interface{}) {
  switch l.LogLevel {
  case ll.CRITICAL:
    return
  }
  t := time.Now()
  n := shared.GetNextLogNum()
  var meta, newArgs = l.getMetaFields(&args)
  (*meta.m)["log_num"] = n
  filteredStackTrace := hlpr.GetFilteredStacktrace()
  newArgs = append(newArgs, StackTrace{filteredStackTrace})
  l.writeSwitch(t, ll.ERROR, meta, &newArgs)
}

func (l *Logger) Critical(args ...interface{}) {
  t := time.Now()
  n := shared.GetNextLogNum()
  var meta, newArgs = l.getMetaFields(&args)
  (*meta.m)["log_num"] = n
  filteredStackTrace := hlpr.GetFilteredStacktrace()
  newArgs = append(newArgs, StackTrace{filteredStackTrace})
  l.writeSwitch(t, ll.CRITICAL, meta, &newArgs)
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

func (l *Logger) TraceF(s string, args ...interface{}) {
  switch l.LogLevel {
  case ll.DEBUG, ll.INFO, ll.WARN, ll.ERROR, ll.CRITICAL:
    return
  }
  t := time.Now()
  var meta = l.MetaFields
  l.writeSwitch(t, ll.TRACE, meta, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) DebugF(s string, args ...interface{}) {
  switch l.LogLevel {
  case ll.INFO, ll.WARN, ll.ERROR, ll.CRITICAL:
    return
  }
  t := time.Now()
  var meta = l.MetaFields
  l.writeSwitch(t, ll.DEBUG, meta, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) InfoF(s string, args ...interface{}) {
  switch l.LogLevel {
  case ll.WARN, ll.ERROR, ll.CRITICAL:
    return
  }
  t := time.Now()
  var meta = l.MetaFields
  l.writeSwitch(t, ll.INFO, meta, &[]interface{}{fmt.Sprintf(s, args...)})
}

func (l *Logger) WarnF(s string, args ...interface{}) {
  switch l.LogLevel {
  case ll.ERROR, ll.CRITICAL:
    return
  }
  t := time.Now()
  var meta = l.MetaFields
  l.writeSwitch(t, ll.WARN, meta, &[]interface{}{fmt.Sprintf(s, args...)})
}

type StackTrace struct {
  ErrorTrace *[]string
}

func (l *Logger) ErrorF(s string, args ...interface{}) {
  switch l.LogLevel {
  case ll.CRITICAL:
    // only logging critical level messages!
    return
  }
  t := time.Now()
  filteredStackTrace := hlpr.GetFilteredStacktrace()
  formattedString := fmt.Sprintf(s, args...)
  var meta = l.MetaFields
  l.writeSwitch(t, ll.ERROR, meta, &[]interface{}{formattedString, StackTrace{filteredStackTrace}})
}

func (l *Logger) CriticalF(s string, args ...interface{}) {
  t := time.Now()
  filteredStackTrace := hlpr.GetFilteredStacktrace()
  formattedString := fmt.Sprintf(s, args...)
  var meta = l.MetaFields
  l.writeSwitch(t, ll.CRITICAL, meta, &[]interface{}{formattedString, StackTrace{filteredStackTrace}})
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
  SetToJSONOutput().
  SetLogLevel(ll.TRACE)

func init() {

  // log.SetFlags(log.LstdFlags | log.Llongfile)

}
