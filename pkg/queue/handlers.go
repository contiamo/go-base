package queue

import "context"

// TaskHandler is a type alias for a method that parses a task and returns any processing errors
type TaskHandler interface {
	// Process implements the specific Task parsing logic
	Process(ctx context.Context, task Task, heartbeats chan<- Progress) (err error)
}

// TaskHandlerFunc is an adapter that allows the use of a normal function
// as a TaskHandler
type TaskHandlerFunc func(context.Context, Task, chan<- Progress) error

// Process implements the specific Task parsing logic
func (f TaskHandlerFunc) Process(ctx context.Context, task Task, heartbeats chan<- Progress) error {
	return f(ctx, task, heartbeats)
}
