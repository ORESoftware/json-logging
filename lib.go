package json_logging

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/logrusorgru/aurora"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

var isTerminal = terminal.IsTerminal(int(os.Stdout.Fd()))
var pid = os.Getpid()

type jsons struct {
	Time int32
}

type Logger struct {
	AppName       string
	IsLoggingJSON bool
	HostName      string
	ForceJSON     bool
	ForceNonJSON  bool
	TimeZone      string
}

type loggingTypeInternal struct {
	JSON   string
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
	MetaFields    MetaFields
	TimeZone      string
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

func (l Logger) Create(m *MetaFields) *Logger {
	return &Logger{
		IsLoggingJSON: l.IsLoggingJSON,
		AppName:       l.AppName,
		HostName:      l.HostName,
	}
}

func (l Logger) writePretty(level string, m *MetaFields, args *[]interface{}) {

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
		stylizedLevel = aurora.BgBrightYellow(level).String()
		break

	case "INFO":
		stylizedLevel = aurora.BrightBlue(level).String()
		break
	}

	buf := []string{
		aurora.Gray(9, date).String(), " ",
		stylizedLevel, " ",
		"app:" + aurora.Cyan(l.AppName).String(), " ",
	}

	for _, v := range buf {
		os.Stdout.Write([]byte(v))
	}

	for _, v := range *args {

		if reflect.TypeOf(v).Name() == "string" {
			os.Stdout.Write([]byte(v.(string) + " "))
			continue
		}

		if reflect.TypeOf(v).Name() == "bool" {
			os.Stdout.Write([]byte(strconv.FormatBool(v.(bool)) + " "))
			continue
		}

		json, err := json.Marshal(v)
		if err != nil {
			z := fmt.Sprintf("%v", v)
			os.Stdout.Write([]byte(z + " "))
			continue
		}

		os.Stdout.Write(json)
	}

	os.Stdout.Write([]byte("\n"))
}

func (l Logger) writeJSON(level string, m *MetaFields, args *[]interface{}) {

	date := time.Now().UTC().String()
	date = date[:26]
	buf, err := json.Marshal([8]interface{}{"@bunion", l.AppName, level, pid, l.HostName, date, m, args})

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

type MetaFields = map[string]interface{}

func MetaPairs(
	k1 string, v1 interface{},
	args ...interface{}) MetaFields {

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

	return m

	//return MetaFields{
	//	Meta: m,
	//}
}

func (l Logger) Infox(m MetaFields, args ...interface{}) {
	l.writeSwitch("INFO", &m, &args)
}

func (l Logger) Warnx(m MetaFields, args ...interface{}) {
	l.writeSwitch("WARN", &m, &args)
}

func (l Logger) Errorx(m MetaFields, args ...interface{}) {
	l.writeSwitch("ERROR", &m, &args)
}

func (l Logger) Fatalx(m MetaFields, args ...interface{}) {
	l.writeSwitch("FATAL", &m, &args)
}

func (l Logger) Debugx(m MetaFields, args ...interface{}) {
	l.writeSwitch("DEBUG", &m, &args)
}

func (l Logger) Tracex(m MetaFields, args ...interface{}) {
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
