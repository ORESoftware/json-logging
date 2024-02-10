package main

import (
  //"github.com/oresoftware/json-logging/jlog/lib"
  "math/rand"
  "time"
  //"fmt"
  "github.com/oresoftware/json-logging/jlog/lib"
  //"fmt"
)

func randomBool() bool {
  // Seed the random number generator to get different results each run
  rand.Seed(time.Now().UnixNano())
  // Generate a random number and check if it is even or odd
  return rand.Intn(2) == 0 // rand.Intn(2) generates 0 or 1
}

func main() {

  var z = struct {
    Foo  string
    Bar  struct{}
    Zoom struct{}
  }{
    Foo: "foo",
    Bar: struct {
    }{},
    Zoom: struct {
    }{},
  }

  var b = struct {
    Foo  string
    Bar  struct{}
    Zoom struct{}
  }{
    Foo: "foo",
    Bar: struct {
    }{},
    Zoom: struct {
    }{},
  }

  for i := 0; i < 10000; i++ {

    //fmt.Println("7699c338-5a46-4106-b847-8d467bb8c268", z, b )

    if randomBool() {
    lib.DefaultLogger.Info("9acc03fa-1972-479e-a64f-598661231731", z)
    } else {
    lib.DefaultLogger.Info("2cc5b58f-dec6-4441-9ec2-550a0423578f", b)
    }
  }
}
