package pool

import (
  "sync"
)

type ChanMessage struct {
  f func(*sync.WaitGroup)
  //wg *sync.WaitGroup
}

type Worker struct {
  c      chan *ChanMessage
  mtx    sync.Mutex
  isBusy bool
}

type Pool struct {
  once              *sync.Once
  mtx               *sync.Mutex
  Size              int
  workers           []*Worker
  Count             int
  RoundRobinCounter int
}

func (p *Pool) incrementCount() {
  p.mtx.Lock()
  p.Count++
  p.mtx.Unlock()
}

func (p *Pool) decrementCount() {
  p.mtx.Lock()
  p.Count--
  p.mtx.Unlock()
}

func (p *Pool) createWorkers() {

  p.once.Do(func() {

    for i := 0; i < p.Size; i++ {

      var w = &Worker{
        c:      make(chan *ChanMessage, 0),
        mtx:    sync.Mutex{},
        isBusy: false,
      }

      go func(w *Worker) {
        for {
          var m = <-w.c
          p.incrementCount()
          var wg = &sync.WaitGroup{}
          wg.Add(1)
          m.f(wg)
          wg.Wait()
          p.decrementCount()
        }
      }(w)

      p.workers = append(p.workers, w)
    }
  })

}

func CreatePool(size int) *Pool {

  var p = &Pool{
    mtx:               &sync.Mutex{},
    Size:              size,
    Count:             0,
    RoundRobinCounter: size + 1,
    once:              &sync.Once{},
  }

  p.createWorkers()

  return p
}

func (p *Pool) Run(z func(*sync.WaitGroup)) {

  p.mtx.Lock()
  //     pool_test.go:14: Alloc: 261 MB, TotalAlloc: 727 MB, Sys: 1457 MB

  if p.Count >= p.Size+10 {
    //fmt.Println("unlocked 0")
    p.mtx.Unlock()
    // queue is pretty full, so just create a new goroutine here
    go z(nil)
    return
  }

  var m = &ChanMessage{
    f: z,
  }

  for _, v := range p.workers {
    select {
    case v.c <- m:
      p.mtx.Unlock()
      return
    default:
      continue
    }
  }

  // couldn't find a non-busy one, so just round robin to next
  p.RoundRobinCounter = (p.RoundRobinCounter + 1) % p.Size
  var v = p.workers[p.RoundRobinCounter]
  p.mtx.Unlock()

  go func() {
    v.c <- m
  }()

}
