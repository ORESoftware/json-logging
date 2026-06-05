package writer

import (
	"fmt"
	"os"
	"sync"
	"syscall"
)

type SafeWriter struct {
	w *os.File
	m *sync.RWMutex
}

var fileLocks sync.Map

func lockKey(v *os.File) string {
	if v == nil {
		return "fd:stdout"
	}

	var st syscall.Stat_t
	if err := syscall.Fstat(int(v.Fd()), &st); err == nil {
		return fmt.Sprintf("inode:%d:%d", st.Dev, st.Ino)
	}

	return fmt.Sprintf("fd:%d", v.Fd())
}

func lockFor(v *os.File) *sync.RWMutex {
	key := lockKey(v)
	actual, _ := fileLocks.LoadOrStore(key, &sync.RWMutex{})
	return actual.(*sync.RWMutex)
}

func NewSafeWriter(v *os.File) *SafeWriter {
	if v == nil {
		v = os.Stdout
	}
	return &SafeWriter{w: v, m: lockFor(v)}
}

func (sw *SafeWriter) Lock() {
	sw.m.Lock()
}

func (sw *SafeWriter) Unlock() {
	sw.m.Unlock()
}

func (sw *SafeWriter) WriteString(p string) (n int, err error) {
	sw.Lock()
	defer sw.Unlock()
	return sw.w.WriteString(p)
}

func (sw *SafeWriter) Write(p []byte) (n int, err error) {
	sw.Lock()
	defer sw.Unlock()
	return sw.w.Write(p)
}

func WriteString(w *os.File, p string) (n int, err error) {
	return NewSafeWriter(w).WriteString(p)
}

func Write(w *os.File, p []byte) (n int, err error) {
	return NewSafeWriter(w).Write(p)
}
