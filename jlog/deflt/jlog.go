package deflt

import (
	"github.com/oresoftware/json-logging/jlog/level"
	logger "github.com/oresoftware/json-logging/jlog/mult"
	"os"
)

var Stdout = logger.New("Vibe:Chat", "", []*logger.FileLevel{})

var Stderr = logger.New("Vibe:Chat/Stderr", "", []*logger.FileLevel{{
	Level:  ll.WARN,
	File:   os.Stderr,
	IsJSON: true,
}})

func InfoWithReq(req struct{ Id string }, args ...interface{}) {
	Stdout.Info(req.Id, args)
}
