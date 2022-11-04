package handlers

import (
	"context"

	"github.com/contiamo/go-base/v4/pkg/queue"
	"github.com/contiamo/go-base/v4/pkg/tracing"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var (
	// ErrNoHandlerFound occurs when dispatcher can'f find a registered handler for a task type
	ErrNoHandlerFound = errors.New("no handler found")
)

// NewDispatchHandler creates a task handler that will dispatch tasks to other handlers
func NewDispatchHandler(handlers map[queue.TaskType]queue.TaskHandler) queue.TaskHandler {
	return &dispatchHandler{
		Tracer:   tracing.NewTracer("handlers", "DispatchHandler"),
		handlers: handlers,
	}
}

type dispatchHandler struct {
	tracing.Tracer
	handlers map[queue.TaskType]queue.TaskHandler
}

func (h *dispatchHandler) Process(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) (err error) {
	span, ctx := h.StartSpan(ctx, "Process")
	logger := logrus.
		WithContext(ctx).
		WithField("component", "dispatchHandler").
		WithField("type", task.Type).
		WithField("id", task.ID).
		WithField("queue", task.Queue)

	defer func() {
		logger.Debug("end")
		h.FinishSpan(span, err)
	}()
	span.SetTag("task.id", task.ID)
	span.SetTag("task.queue", task.Queue)
	span.SetTag("task.type", task.Type)
	span.SetTag("task.spec", string(task.Spec))

	logger.Debug("dispatching task")
	handler, ok := h.handlers[task.Type]
	if !ok {
		logger.Error("there is no handler for this task type")
		close(heartbeats)
		return ErrNoHandlerFound
	}

	logger.Debug("pass to handler")
	return handler.Process(ctx, task, heartbeats)
}
