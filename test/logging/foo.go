package logging

import jlog "github.com/oresoftware/json-logging"


var Log = jlog.New("Sam", false, "")

func InfoWithReq(req struct{Id string}, args ...interface{}) {
	Log.Info(req.Id, args)
}
