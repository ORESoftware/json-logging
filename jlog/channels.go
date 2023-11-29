package json_logging

type Z = func(s int, c chan int)

func Run(list []Z) []interface{} {

	results := []interface{}{}

	c := make(chan int)

	for i := 0; i < len(list); i++ {
		results = append(results, nil)
		go list[i](i, c)
		results[i] = <-c
	}

	return results
}
