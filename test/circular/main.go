package main

import (
	jlog "github.com/oresoftware/json-logging/jlog"
	"os"
)

var log = jlog.New("vb", "vibe_", jlog.DEBUG, []*os.File{os.Stdout})

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

	//log.Info(m)
	log.Error(m)
}
