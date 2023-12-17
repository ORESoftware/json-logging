package stack

import (
	"errors"
	"sync"
)

// Stack represents a stack data structure for strings.

type StackItem struct {
	Id  string
	Lck *sync.Mutex
}

type Stack struct {
	elements []*StackItem
}

// NewStack creates a new Stack.
func NewStack() *Stack {
	return &Stack{elements: []*StackItem{}}
}

// Push adds an element to the top of the stack.
func (s *Stack) Push(element *StackItem) {
	s.elements = append(s.elements, element)
}

// Pop removes and returns the top element of the stack. If the stack is empty, an error is returned.
func (s *Stack) Pop() (*StackItem, error) {
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
	if len(s.elements) == 0 {
		return nil, errors.New("stack is empty")
	}

	return s.elements[len(s.elements)-1], nil
}

// IsEmpty checks whether the stack is empty.
func (s *Stack) IsEmpty() bool {
	return len(s.elements) == 0
}
