package pool_test

import (
	"github.com/oresoftware/json-logging/jlog/pool"
	"sync"
	"testing"
	"time"
)

// Test if the worker pool can handle a simple task
func TestWorkerPoolSimpleTask(t *testing.T) {
	poolSize := 3
	p := pool.CreatePool(poolSize)

	var wg sync.WaitGroup
	wg.Add(1)

	p.Run(func(g *sync.WaitGroup) {
		// Perform a simple task
		t.Logf("Simple task executed")
		g.Done()
		wg.Done()
	})

	// Ensure the pool count is updated correctly
	if p.Count != 0 {
		t.Errorf("Expected pool count to be 0, got %d", p.Count)
	}
}

// Test if the worker pool can handle multiple concurrent tasks
func TestWorkerPoolConcurrentTasks(t *testing.T) {
	poolSize := 5
	p := pool.CreatePool(poolSize)

	numTasks := 10
	var wg sync.WaitGroup
	wg.Add(numTasks)

	for i := 0; i < numTasks; i++ {
		go func(index int) {
			p.Run(func(g *sync.WaitGroup) {
				// Perform a task and log the index
				t.Logf("Task %d executed", index)
				g.Done()
				wg.Done()
			})
		}(i)
	}

	wg.Wait()

	// Ensure the pool count is updated correctly
	if p.Count != 0 {
		t.Errorf("Expected pool count to be 0, got %d", p.Count)
	}
}

// Test if the worker pool can handle more tasks than the pool size
func TestWorkerPoolFullQueue(t *testing.T) {
	poolSize := 3
	p := pool.CreatePool(poolSize)

	numTasks := 5
	var wg sync.WaitGroup
	wg.Add(numTasks)

	for i := 0; i < numTasks; i++ {
		go func(index int) {
			p.Run(func(g *sync.WaitGroup) {
				// Perform a task and log the index
				t.Logf("Task %d executed", index)
				time.Sleep(50 * time.Millisecond) // Simulate task execution time
				g.Done()
				wg.Done()
			})
		}(i)
	}

	wg.Wait()

	// Ensure the pool count is updated correctly
	if p.Count != 0 {
		t.Errorf("Expected pool count to be 0, got %d", p.Count)
	}
}

// Test if the worker pool handles a large number of tasks without deadlocks or panics
func TestWorkerPoolStressTest(t *testing.T) {
	poolSize := 10
	p := pool.CreatePool(poolSize)

	numTasks := 1000
	var wg sync.WaitGroup
	wg.Add(numTasks)

	for i := 0; i < numTasks; i++ {
		go func(index int) {
			p.Run(func(g *sync.WaitGroup) {
				// Perform a task and log the index
				t.Logf("Task %d executed", index)
				g.Done()
				wg.Done()
			})
		}(i)
	}

	wg.Wait()

	// Ensure the pool count is updated correctly
	if p.Count != 0 {
		t.Errorf("Expected pool count to be 0, got %d", p.Count)
	}
}
