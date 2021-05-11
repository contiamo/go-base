package handlers

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/contiamo/go-base/v3/pkg/queue"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

type errHandler struct {
	t       *testing.T
	expTask queue.Task
}

func (h errHandler) Process(ctx context.Context, task queue.Task, heartbeats chan<- queue.Progress) (err error) {
	require.Equal(h.t, h.expTask, task)
	return errors.New("invalid")
}

func TestDispatcherProcess(t *testing.T) {
	logrus.SetOutput(ioutil.Discard)
	defer logrus.SetOutput(os.Stdout)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	t.Run("propagates the underlying handler's error", func(t *testing.T) {
		task := queue.Task{TaskBase: queue.TaskBase{Type: "test"}}

		h := NewDispatchHandler(map[queue.TaskType]queue.TaskHandler{
			"test": errHandler{t: t, expTask: task},
		})

		err := h.Process(ctx, task, nil)
		require.Error(t, err)
		require.Equal(t, "invalid", err.Error())
	})

	t.Run("returns ErrNoHandlerFound when there is no handler and closes heartbeats", func(t *testing.T) {
		task := queue.Task{TaskBase: queue.TaskBase{Type: "test"}}

		h := NewDispatchHandler(map[queue.TaskType]queue.TaskHandler{})

		heartbeats := make(chan queue.Progress)
		err := h.Process(ctx, task, heartbeats)
		<-heartbeats
		require.PanicsWithError(t, "send on closed channel", func() {
			heartbeats <- queue.Progress{}
		})
		require.Error(t, err)
		require.Equal(t, ErrNoHandlerFound.Error(), err.Error())
	})
}
