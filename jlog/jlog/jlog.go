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

var Stdout = logger.New(appName, "", logger.WARN, []*logger.FileLevel{})
