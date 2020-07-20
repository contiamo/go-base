package workers

import "context"

// Worker provides methods to do some kind of work
type Worker interface {
	// Work is responsible for getting and processing tasks
	// It should run continuously or until the context is cancelled
	Work(context.Context) error
}
