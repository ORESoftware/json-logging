package jlog

import logger "github.com/oresoftware/json-logging/jlog"

var Stdout = logger.New("Vibe:Chat", false, "")

var Stderr = logger.New("Vibe:Chat/Stderr", false, "")

func InfoWithReq(req struct{ Id string }, args ...interface{}) {
	Stdout.Info(req.Id, args)
}
