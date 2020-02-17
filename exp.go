package json_logging

import "log"

var s = struct{ Foo string }{"foo"}

func acceptStruct(a interface{}, b interface{}) bool {
	return a == b
}

func main() {
	log.Println(acceptStruct(s, &s))
}
