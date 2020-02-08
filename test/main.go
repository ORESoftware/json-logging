package main

import (
	jlog "github.com/oresoftware/json-logging"
	. "github.com/oresoftware/json-logging/test/logging"
)

func main() {

	type Buff struct{ Bagel bool }

	Log.Info(Buff{}, struct {
		Foo  string `json:"foo"`
		Butt Buff   `json:"buff"`
	}{"bar", Buff{
		Bagel: true,
	},
	})

	Log.Infox(map[string]interface{}{"foo": 5}, "bar", struct {
		Foo  string `json:"foo"`
		Butt Buff   `json:"buff"`
	}{"bar", Buff{
		Bagel: true,
	},
	})

	m := jlog.MetaPairs("foo", 5, "zgage", "vv")

	Log.Infox(m, "bar", struct {
		Foo  string `json:"foo"`
		Butt Buff   `json:"buff"`
	}{"bar", Buff{
		Bagel: true,
	},
	})

	Log.Info("foo")


}
