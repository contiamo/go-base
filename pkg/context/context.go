package context

import (
	"context"
	"errors"
	"sync"
	"time"
)

type resetableTimeoutContext struct {
	mutex   sync.RWMutex
	parent  context.Context
	current context.Context
	cancel  context.CancelFunc
}

// WithResetableTimeout acts like context.WithTimeout, but enables the ResetTimeout() function on the returned context
func WithResetableTimeout(ctx context.Context, dur time.Duration) (context.Context, context.CancelFunc) {
	res := &resetableTimeoutContext{
		parent: ctx,
	}
	res.current, res.cancel = context.WithTimeout(ctx, dur)
	return res, func() {
		res.mutex.RLock()
		defer res.mutex.RUnlock()
		res.cancel()
	}
}

// ResetTimeout resets the timeout of the given context if it was created using WithResetableTimeout
// With this function you can extend the timeout of a context
func ResetTimeout(ctx context.Context, dur time.Duration) error {
	resetable, ok := ctx.(*resetableTimeoutContext)
	if !ok {
		return errors.New("context timeout is not resetable")
	}
	resetable.mutex.Lock()
	defer resetable.mutex.Unlock()
	resetable.current, resetable.cancel = context.WithTimeout(resetable.parent, dur)
	return nil
}

func (ctx *resetableTimeoutContext) Deadline() (deadline time.Time, ok bool) {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.current.Deadline()
}

func (ctx *resetableTimeoutContext) Done() <-chan struct{} {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.current.Done()
}

func (ctx *resetableTimeoutContext) Err() error {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.current.Err()
}

func (ctx *resetableTimeoutContext) Value(key interface{}) interface{} {
	ctx.mutex.RLock()
	defer ctx.mutex.RUnlock()
	return ctx.current.Value(key)
}
