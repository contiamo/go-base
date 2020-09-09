package queue

import (
	"context"

	cdb "github.com/contiamo/go-base/v2/pkg/db"
)

// SchedulerMock implement scheduler with very simple counters for testing
type SchedulerMock struct {
	ScheduleErr   error
	ScheduleCount int

	EnsureError error
	EnsureCount int

	AssertError error
	AssertCount int
}

func (s *SchedulerMock) Schedule(ctx context.Context, builder cdb.SQLBuilder, task TaskScheduleRequest) (err error) {
	s.ScheduleCount = s.ScheduleCount + 1
	return s.ScheduleErr
}

func (s *SchedulerMock) EnsureSchedule(ctx context.Context, builder cdb.SQLBuilder, task TaskScheduleRequest) error {
	s.EnsureCount = s.EnsureCount + 1
	return s.EnsureError
}

func (s *SchedulerMock) AssertSchedule(ctx context.Context, task TaskScheduleRequest) error {
	s.AssertCount = s.AssertCount + 1
	return s.AssertError
}
