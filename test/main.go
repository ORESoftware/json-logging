package main

import (
	. "ores/json-logging/test/logging"
	)

func main() {


	type Buff struct{ Bagel bool }

	Log.Info("foo", struct {
		Foo  string
		Butt Buff
	}{"bar", Buff{
		Bagel: true,
	},
	})

}
