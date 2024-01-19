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
  readyWorkers      *DoublyLinkedList
  workers           []*Worker
  Count             int
  RoundRobinCounter int
  NewGRCount        int
  RRCount           int
  FreeWorkerCount   int
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

func (p *Pool) enqueueWorker(w *Worker) {
  p.mtx.Lock()
  p.readyWorkers.Enqueue(w)
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
          p.enqueueWorker(w)
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
    readyWorkers:      NewDoublyLinkedList(),
  }

  p.createWorkers()

  return p
}

func (p *Pool) Run(z func(*sync.WaitGroup)) {

  p.mtx.Lock()
  // pool_test.go:14: Alloc: 261 MB, TotalAlloc: 727 MB, Sys: 1457 MB

  //if p.Count >= p.Size+100 {
  //  p.NewGRCount++
  //  p.mtx.Unlock()
  //  // queue is pretty full, so just create a new goroutine here
  //  go z(nil)
  //  return
  //}

  if p.Count > p.Size {
    panic("should not happen")
  }

  var m = &ChanMessage{
    f: z,
  }

  if b, err := p.readyWorkers.Dequeue(); err == nil {
    p.mtx.Unlock()
    select {
    case b.c <- m:
      p.FreeWorkerCount++
      return
    default:
      p.mtx.Lock()
    }
  }

  //TODO make a queue of workers where the ready ones are at front of list
  for _, v := range p.workers {
    select {
    case v.c <- m:
      p.FreeWorkerCount++
      p.mtx.Unlock()
      return
    default:
      continue
    }
  }

  // couldn't find a non-busy one, so just round robin to next
  p.RoundRobinCounter = (p.RoundRobinCounter + 1) % p.Size
  var v = p.workers[p.RoundRobinCounter]
  p.RRCount++
  p.mtx.Unlock()

  go func() {
    v.c <- m
  }()

}
