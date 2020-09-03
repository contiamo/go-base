package handlers

import (
	"context"

	"github.com/contiamo/go-base/pkg/queue"
	"github.com/contiamo/go-base/pkg/queue/workers"
	"github.com/contiamo/go-base/pkg/tracing"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// ErrNoHandlerFound occurs when dispatcher can'f find a registered handler for a task type
	ErrNoHandlerFound = errors.New("no handler found")
)

// NewDispatchHandler creates a task handler that will dispatch tasks to other handlers
func NewDispatchHandler(handlers map[queue.TaskType]workers.TaskHandler) workers.TaskHandler {
	return &dispatchHandler{
		Tracer:   tracing.NewTracer("handlers", "DispatchHandler"),
		handlers: handlers,
	}
}

type dispatchHandler struct {
	tracing.Tracer
	handlers map[queue.TaskType]workers.TaskHandler
}

func (h *dispatchHandler) Process(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) (err error) {
	span, ctx := h.StartSpan(ctx, "Process")
	defer func() {
		h.FinishSpan(span, err)
	}()
	span.SetTag("task.id", task.ID)
	span.SetTag("task.queue", task.Queue)
	span.SetTag("task.type", task.Type)
	span.SetTag("task.spec", string(task.Spec))

	logrus := logrus.WithField("type", task.Type)

	logrus.Debug("dispatching task...")
	handler, ok := h.handlers[task.Type]
	if !ok {
		logrus.Error("there is no handler for this task type")
		close(heartbeats)
		return ErrNoHandlerFound
	}
	return handler.Process(ctx, task, heartbeats)
}
