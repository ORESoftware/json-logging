package logging

import (
	jlog "github.com/oresoftware/json-logging/jlog/default"
)

func InfoWithReq(req struct{ Id string }, args ...interface{}) {
	jlog.Stdout.Info(req.Id, args)
}
