package main

import (
	json_logging "github.com/oresoftware/json-logging"
	"time"
)

func main()  {

	inputs := []json_logging.Z{
		func(s int, c chan int) {
			time.Sleep(2*time.Second)
            c <- s+100
		},
		func(s int, c chan int) {
			c <- s+101
		},
		func(s int, c chan int) {
			c <- s+102
		},
	}
	results := json_logging.Run(inputs)

	for i :=0 ; i< len(results); i++ {
		json_logging.DefaultLogger.Info(i, results[i])
	}
}