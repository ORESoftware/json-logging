package main

import (
	jlog "github.com/oresoftware/json-logging/jlog/lib"
	"time"
)

var log = jlog.CreateLogger("foo")

func main3() {

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
