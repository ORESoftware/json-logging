package deflt

import (
	logger "github.com/oresoftware/json-logging/jlog/mult"
	"github.com/oresoftware/json-logging/jlog/shared"
	"os"
)

var Stdout = logger.New("Vibe:Chat", "", []*logger.FileLevel{})

var Stderr = logger.New("Vibe:Chat/Stderr", "", []*logger.FileLevel{{
	Level:  shared.WARN,
	File:   os.Stderr,
	IsJSON: true,
}})

func InfoWithReq(req struct{ Id string }, args ...interface{}) {
	Stdout.Info(req.Id, args)
}
