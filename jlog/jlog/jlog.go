package jlog

import (
	logger "github.com/oresoftware/json-logging/jlog"
	"os"
)

var appName = func() string {
	var appName = os.Getenv("jlog_app_name")
	switch appName {
	case "":
		return "default"
	}
	return os.Getenv("jlog_app_name")
}()

var stdout = logger.New(appName, false, "")

var Info = stdout.Info

var InfoF = stdout.InfoF

var Warning = stdout.Warning

var WarningF = stdout.WarningF

var Error = stdout.Error

var Errorf = stdout.ErrorF

var Trace = stdout.Trace

var Tracef = stdout.TraceF

var Debug = stdout.Debug

var Debugf = stdout.DebugF
