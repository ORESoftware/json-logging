package pool

import (
	"fmt"
	"sync"
)

type ChanMessage struct {
	f  func(*sync.WaitGroup)
	wg *sync.WaitGroup
}
type Worker struct {
	c      chan *ChanMessage
	mtx    sync.Mutex
	isBusy bool
}

type Pool struct {
	mtx               *sync.Mutex
	Size              int
	workers           []*Worker
	Count             int
	RoundRobinCounter int
}

func (p *Pool) createWorkers() {

	for i := 0; i < p.Size; i++ {

		var w = &Worker{
			c:      make(chan *ChanMessage, 1),
			mtx:    sync.Mutex{},
			isBusy: false,
		}

		go func(w *Worker) {
			for {
				var m = <-w.c
				w.mtx.Lock()
				w.isBusy = true
				p.Count++
				fmt.Println("pool count:", p.Count, p.Size)
				m.f(m.wg)
				p.Count--
				w.isBusy = false
				w.mtx.Unlock()
			}
		}(w)

		p.workers = append(p.workers, w)
	}
}

func CreatePool(size int) *Pool {

	var p = &Pool{
		mtx:               &sync.Mutex{},
		Size:              size,
		Count:             0,
		RoundRobinCounter: size + 1,
	}

	p.createWorkers()

	return p
}

func (p *Pool) Run(z func(*sync.WaitGroup)) {

	p.mtx.Lock()

	var wg = &sync.WaitGroup{}
	wg.Add(1)

	var m = &ChanMessage{
		f:  z,
		wg: wg,
	}

	if p.Count >= p.Size {
		p.mtx.Unlock()
		// queue is full, so just create a new goroutine here
		go z(wg)
		return
	}

	for _, v := range p.workers {
		if !v.isBusy {
			v.mtx.Lock()
			p.mtx.Unlock()
			v.isBusy = true
			v.mtx.Unlock()
			v.c <- m
			return
		}
	}

	// couldn't find a non-busy one, so just round robin to next
	p.RoundRobinCounter = (p.RoundRobinCounter + 1) % p.Size
	var v = p.workers[p.RoundRobinCounter]
	p.mtx.Unlock()

	v.mtx.Lock()
	v.isBusy = true
	v.mtx.Unlock()
	v.c <- m

}
