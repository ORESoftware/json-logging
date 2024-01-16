package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// Set a timeout of 2 seconds
	timeout := 2 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel() // Ensure cancel is called to release resources associated with the context

	// Call a function with the timeout
	result, err := performTaskWithContext(ctx)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Result:", result)
}

func performTaskWithContext(ctx context.Context) (string, error) {
	// Simulate a task that may take some time
	select {
	case <-time.After(3 * time.Second): // Simulate a task that takes 3 seconds
		return "Task completed", nil
	case <-ctx.Done():
		return "", ctx.Err() // Return an error if the context is canceled or times out
	}
}
