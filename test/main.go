package main

import (
	log "json-logging"
)

func main() {

	type Buff struct{ Bagel bool }

	log.DefaultLogger.Info("foo", struct {
		Foo  string
		Butt Buff
	}{"bar", Buff{
		Bagel: true,
	},
	})

}
