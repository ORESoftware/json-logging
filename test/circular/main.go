package main

import (
	jlog "github.com/oresoftware/json-logging/jlog"
)

var log = jlog.New("vb", true, "hn", "vibe_")

func main() {
	//jlog.DefaultLogger.Error("foo")

	type M struct {
		Foo string
		Z   struct {
			Bar string
			Z   struct {
				Bzz string
				Z   interface {
				}
			}
		}
	}

	m := M{}
	m.Z.Z.Z = &m

	log.Info(m)
}
