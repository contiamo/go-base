package queue

import "context"

// Worker provides methods to do some kind of work
type Worker interface {
	// Work is responsible for getting and processing tasks
	// It should run continuously or until the context is canceled
	Work(context.Context) error
}
