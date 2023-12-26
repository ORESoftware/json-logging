package main

import jlog "github.com/oresoftware/json-logging/jlog"

func main() {
	//jlog.DefaultLogger.Error("foo")

	var z = struct {
		Z interface{}
	}{}

	z.Z = &z

	jlog.DefaultLogger.Info(z)
}
