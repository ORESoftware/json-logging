package main

import (
	"os"
)

type Zoom struct {
	Bagel bool
	K     struct{}
	Val   int
}

func (z Zoom) Zoom() string {
	return "bop"
}

func main() {

	//Log.Warn([3]interface{}{"foo","bar", struct{foo string}{"fu"}})
	//
	//Log.Info(" ag ageg a gae")
	//

	//Log.JSON("foo", true, 44.44, make(chan int))

	//Log.Info("foo", 55, main, true, struct{ Boo string }{"fudge"}, map[string]interface{}{"fpp": "age", "boop": struct{
	//	Bop string
	//	dog int
	//}{"age", 5}})

	fog.Info("'foo'", 55, main, true, struct{ Boo string }{"fudge"}, map[int]int{}, map[string]interface{}{"fpp": "age", "boop": struct {
		Bop string
		dog int
		c   chan int
	}{"age", 5, make(chan int)}})

	os.Exit(3)

	//Log.Info(map[string]interface{}{"foo": 5}, "bar", struct {
	//	Foo  string `json:"foo"`
	//	Butt Zoom   `json:"buff"`
	//}{"bar", Zoom{
	//	Bagel: true,
	//},
	//})

	//Log.Info(Zoom{})

	fog.Info(Zoom{}, Zoom{}, Zoom{}, Zoom{}, struct {
		Foo  string `json:"foo"`
		Butt Zoom   `json:"buff"`
	}{"bar", Zoom{
		Bagel: false,
		Val:   33,
		K: struct {
		}{},
	}})

	//Log.Info(Zoom{}, struct {
	//	Foo  string `json:"foo"`
	//	Butt Zoom   `json:"buff"`
	//}{"bar", Zoom{
	//	Bagel: false,
	//	Val:   33,
	//	K: struct {
	//	}{},
	//}})

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
