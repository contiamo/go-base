package config

import (
	"time"
)

// Queue contains configuation for task queueing
type Queue struct {
	// HeartbeatTTL is the max time a worker is expected to call queue.Heartbeat when processing a task
	HeartbeatTTL time.Duration `json:"heartbeatTTL"`
	// PollFrequency is the polling frequency when the queue will check for a new task for a waiting worker
	PollFrequency time.Duration `json:"pollFrequency"`
	// JobSchedulingInterval is the interval between iterations of the worker (e.g. job scheduling)
	SchedulingInterval time.Duration `json:"schedulingInterval"`
	// Minimal time between two listener reconnects
	MinReconnectTimeout time.Duration `json:"minReconnectTimeout"`
	// Maximal time between two listener reconnects
	MaxReconnectTimeout time.Duration `json:"maxReconnectTimeout"`
}
