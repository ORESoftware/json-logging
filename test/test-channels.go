package main

import (
	"errors"
	"fmt"
	chns "github.com/oresoftware/json-logging/jlog"
	jlog "github.com/oresoftware/json-logging/jlog/lib"
	"github.com/oresoftware/json-logging/test/logging"
	"sync"
	"time"
)

func do() (string, error) {
	return " ", errors.New("agag")
}

func main() {

	var d, err = do()

	logging.InfoWithReq(struct{ Id string }{Id: ""}, err, d)

	var wg = sync.WaitGroup{}
	words := []string{"foo", "bar", "baz"}
	for _, word := range words {
		wg.Add(1)
		go func(word string) {
			time.Sleep(1 * time.Second)
			defer wg.Done()
			fmt.Println(word)
		}(word)
	}
	wg.Wait() //     // blocks/waits for waitgroup

}

func main2() {

	inputs := []chns.Z{
		func(s int, c chan int) {
			time.Sleep(2 * time.Second)
			c <- s + 100
		},
		func(s int, c chan int) {
			c <- s + 101
		},
		func(s int, c chan int) {
			c <- s + 102
		},
	}
	results := chns.Run(inputs)

	for i := 0; i < len(results); i++ {
		jlog.DefaultLogger.Info(i, results[i])
	}
}
