package context

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResetableTimeoutContextBasicBehavior(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := WithResetableTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	time.Sleep(50 * time.Millisecond)
	assert.Nil(t, ctx.Err())
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}

func TestResetableTimeoutContextReseting(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := WithResetableTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	time.Sleep(50 * time.Millisecond)
	assert.Nil(t, ctx.Err())
	err := ResetTimeout(ctx, 100*time.Millisecond)
	assert.Nil(t, err)
	time.Sleep(60 * time.Millisecond)
	assert.Nil(t, ctx.Err())
	time.Sleep(60 * time.Millisecond)
	assert.Equal(t, context.DeadlineExceeded, ctx.Err())
}
