package stack

import (
	"errors"
	"fmt"
	"sync"
)

// Stack represents a stack data structure for strings.

type StackItem struct {
	Id  string
	Lck *sync.Mutex
}

type Stack struct {
	mtx      sync.Mutex
	elements []*StackItem
}

// NewStack creates a new Stack.
func NewStack() *Stack {
	return &Stack{elements: []*StackItem{}}
}

func (s *Stack) Print(z string) {
	s.mtx.Lock()
	fmt.Println(z)
	for i := 0; i < len(s.elements); i++ {
		fmt.Println(fmt.Sprintf("%v %+v", i, s.elements[i]))
	}
	s.mtx.Unlock()
}

// Push adds an element to the top of the stack.
func (s *Stack) Push(element *StackItem) {
	//
	s.mtx.Lock()
	defer s.mtx.Unlock()
	s.elements = append(s.elements, element)
}

// Pop removes and returns the top element of the stack. If the stack is empty, an error is returned.
func (s *Stack) Pop() (*StackItem, error) {
	//
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if len(s.elements) == 0 {
		return nil, errors.New("stack is empty")
	}

	// Get the top element
	topIndex := len(s.elements) - 1
	topElement := s.elements[topIndex]

	// Remove the top element
	s.elements = s.elements[:topIndex]

	return topElement, nil
}

// Peek returns the top element of the stack without removing it. If the stack is empty, an error is returned.
func (s *Stack) Peek() (*StackItem, error) {
	//
	s.mtx.Lock()
	defer s.mtx.Unlock()

	if len(s.elements) == 0 {
		return nil, errors.New("stack is empty")
	}

	return s.elements[len(s.elements)-1], nil
}

// IsEmpty checks whether the stack is empty.
func (s *Stack) IsEmpty() bool {
	//
	s.mtx.Lock()
	defer s.mtx.Unlock()
	return len(s.elements) == 0
}
