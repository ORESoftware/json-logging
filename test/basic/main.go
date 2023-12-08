package main

import (
	jlog "github.com/oresoftware/json-logging/jlog"
)

func main() {
	jlog.DefaultLogger.Error("foo")
}
