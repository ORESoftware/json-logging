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

var Infof = stdout.Infof

var Warning = stdout.Warning

var Warningf = stdout.Warningf

var Error = stdout.Error

var Errorf = stdout.Errorf

var Trace = stdout.Trace

var Tracef = stdout.Tracef

var Debug = stdout.Debug

var Debugf = stdout.Debugf
