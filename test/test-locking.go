package main

import (
	jlog "github.com/oresoftware/json-logging/jlog"
	"time"
)

var log = jlog.New("foo", false, "")

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
			lckLog.Warning("bar")
			lckLog.Warn("zzz")

			unlock()
		}
	}()

	select {}
}
