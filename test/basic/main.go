package main

import (
	jlog "github.com/oresoftware/json-logging/jlog/lib"
)

func main() {
	jlog.DefaultLogger.Error(jlog.Id("my id 1"), "foo")
	jlog.DefaultLogger.Error("foo", jlog.Id("my id 2"))

}
