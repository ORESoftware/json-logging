package jlog

import (
  logger "github.com/oresoftware/json-logging/jlog/lib"
  "os"
  "strings"
  ll "github.com/oresoftware/json-logging/jlog/level"
)

var appName = func() string {
  var appName = os.Getenv("jlog_app_name")
  switch appName {
  case "":
    return "default"
  }
  return os.Getenv("jlog_app_name")
}()

var envPrefix = func() string {
  var prfx = os.Getenv("jlog_env_prefix")
  var trimmed = strings.TrimSpace(prfx)
  // explicit AF
  switch trimmed {
  case "":
    return ""
  }
  return trimmed
}()

var Stdout = logger.CreateLogger(appName).SetEnvPrefix("").SetLogLevel(ll.TRACE)
