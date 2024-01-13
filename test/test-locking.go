package main

import (
	jlog "github.com/oresoftware/json-logging/jlog"
	"os"
	"time"
)

var log = jlog.New("foo", "", jlog.WARN, []*os.File{os.Stdout})

func main() {

	go func() {
		for {
			time.Sleep(10 * time.Millisecond)
			log.Warn("this shouldnt be in the middle")
		}
	}()

	go func() {

		for {

			//log.Info("aaaa")
			var lckLog, unlock = log.NewLoggerWithLock()
			lckLog.Warn("bbbbb")
			time.Sleep(10 * time.Millisecond)
			lckLog.Info("foo")
			lckLog.Warn("bar")
			lckLog.Warn("zzz")

			unlock()
		}
	}()

	select {}
}
