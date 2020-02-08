package main

import (
	. "github.com/oresoftware/json-logging/test/logging"
)


type Zoom struct {
	Bagel bool
	K     struct{}
	Val   int
}

func (z Zoom) Zoom() string{
   return "bop"
}


func main() {


	Log.Warn([3]interface{}{"foo","bar", struct{foo string}{"fu"}})

	Log.Info(" ag ageg a gae")

	Log.Info("foo", 55, main, true, struct{ Boo string }{"fudge"}, map[string]string{"fpp": "age"})

	Log.Info(map[string]interface{}{"foo": 5}, "bar", struct {
		Foo  string `json:"foo"`
		Butt Zoom   `json:"buff"`
	}{"bar", Zoom{
		Bagel: true,
	},
	})

	Log.Info(Zoom{}, struct {
		Foo  string `json:"foo"`
		Butt Zoom   `json:"buff"`
	}{"bar", Zoom{
		Bagel: false,
		Val:   33,
		K: struct {
		}{},
	}})

	//m := jlog.MetaPairs("foo", 5, "zgage", "vv")
	//
	//Log.Infox(m, "bar", struct {
	//	Foo  string `json:"foo"`
	//	Butt Zoom   `json:"buff"`
	//}{"bar", Zoom{
	//	Bagel: true,
	//},
	//})
	//
	//Log.Info("foo")

}
