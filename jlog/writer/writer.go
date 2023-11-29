package writer

import (
	"os"
	"sync"
)

type SafeWriter struct {
	w *os.File
	m *sync.Mutex
}

func NewSafeWriter(v *os.File) *SafeWriter {
	return &SafeWriter{w: v, m: &sync.Mutex{}}
}

func (sw *SafeWriter) Lock() {
	sw.m.Lock()
}

func (sw *SafeWriter) Unlock() {
	sw.m.Unlock()
}

func (sw *SafeWriter) WriteAllWithSelfLock(p []byte) (n int, err error) {
	sw.m.Lock()
	defer sw.m.Unlock()
	return (*sw.w).Write(p)
}

func (sw *SafeWriter) Write(p []byte) (n int, err error) {
	return (*sw.w).Write(p)
}
