package pool

import (
  "errors"
  "fmt"
  "sync"
)

// Node represents a node in the doubly-linked list
type Node struct {
  Value *Worker
  Prev  *Node
  Next  *Node
}

// DoublyLinkedList represents the doubly-linked list
type DoublyLinkedList struct {
  Head *Node
  Tail *Node
  Size int
  mu   sync.Mutex // Mutex for synchronization
}

// NewDoublyLinkedList creates a new empty doubly-linked list
func NewDoublyLinkedList() *DoublyLinkedList {
  return &DoublyLinkedList{}
}

// Enqueue adds a new node to the end of the list
func (dll *DoublyLinkedList) Enqueue(value *Worker) {
  dll.mu.Lock()
  defer dll.mu.Unlock()

  newNode := &Node{Value: value}

  if dll.Size == 0 {
    dll.Head = newNode
    dll.Tail = newNode
  } else {
    newNode.Prev = dll.Tail
    dll.Tail.Next = newNode
    dll.Tail = newNode
  }

  dll.Size++
}

// Dequeue removes and returns the node from the front of the list
func (dll *DoublyLinkedList) Dequeue() (*Worker, error) {
  dll.mu.Lock()
  defer dll.mu.Unlock()

  if dll.Size == 0 {
    return nil, errors.New("queue is empty")
  }

  value := dll.Head.Value
  dll.Head = dll.Head.Next
  dll.Size--

  if dll.Size == 0 {
    dll.Tail = nil
  } else {
    dll.Head.Prev = nil
  }

  return value, nil
}

// InsertAfter inserts a new node with the given value after the specified node
func (dll *DoublyLinkedList) InsertAfter(prevNode *Node, value *Worker) {
  dll.mu.Lock()
  defer dll.mu.Unlock()

  newNode := &Node{Value: value}

  if prevNode == nil {
    newNode.Next = dll.Head
    dll.Head.Prev = newNode
    dll.Head = newNode
  } else {
    newNode.Prev = prevNode
    newNode.Next = prevNode.Next
    prevNode.Next = newNode

    if newNode.Next != nil {
      newNode.Next.Prev = newNode
    } else {
      dll.Tail = newNode
    }
  }

  dll.Size++
}

// DeleteNode removes the specified node from the list
func (dll *DoublyLinkedList) DeleteNode(node *Node) {
  dll.mu.Lock()
  defer dll.mu.Unlock()

  if node.Prev != nil {
    node.Prev.Next = node.Next
  } else {
    dll.Head = node.Next
  }

  if node.Next != nil {
    node.Next.Prev = node.Prev
  } else {
    dll.Tail = node.Prev
  }

  dll.Size--
}

// PrintList prints the elements of the doubly-linked list
func (dll *DoublyLinkedList) PrintList() {
  dll.mu.Lock()
  defer dll.mu.Unlock()

  current := dll.Head
  for current != nil {
    fmt.Print(current.Value, " ")
    current = current.Next
  }
  fmt.Println()
}
