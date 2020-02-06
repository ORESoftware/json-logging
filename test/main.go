package main

import (
	jlog "github.com/oresoftware/json-logging"
	. "github.com/oresoftware/json-logging/test/logging"
)

func main() {

	type Buff struct{ Bagel bool }

	Log.Info("foo", struct {
		Foo  string `json:"foo"`
		Butt Buff `json:"buff"`
	}{"bar", Buff{
		Bagel: true,
	},
	})


	Log.InfoWithMeta(jlog.Meta(map[string]interface{}{"foo":5}) , "bar",struct {
		Foo  string `json:"foo"`
		Butt Buff `json:"buff"`
	}{"bar", Buff{
		Bagel: true,
	},
	})


	Log.Tabs(5)
	Log.NewLine()
	Log.Spaces(8)

}
