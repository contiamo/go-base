package queue

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// largest bucket is 5 seconds
var durationMsBuckets = []float64{10, 50, 100, 200, 300, 500, 1000, 2000, 3000, 5000}

var (
	queueMetricLabels = []string{"queue"}
	taskMetricLabels  = []string{"queue", "type"}
)

// TaskQueueMetrics provides access to the prometheus metric objects for the task queue
var TaskQueueMetrics = struct {
	Labels          []string
	TaskCounter     *prometheus.CounterVec
	EnqueueDuration *prometheus.HistogramVec
}{
	Labels: queueMetricLabels,
	TaskCounter: promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "hub",
			Subsystem: "queue",
			Name:      "task",
			Help:      "count of tasks that have been enqueued",
		},
		taskMetricLabels,
	),
	EnqueueDuration: promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "hub",
			Subsystem: "queue",
			Name:      "enqueue_duration_ms",
			Help:      "duration the enqueue action in ms",
			Buckets:   durationMsBuckets,
		},
		queueMetricLabels,
	),
}

var SchedulerMetrics = struct {
	Labels          []string
	ScheduleCounter *prometheus.CounterVec
	ErrorCounter    *prometheus.CounterVec
}{
	Labels: queueMetricLabels,
	ScheduleCounter: promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "hub",
			Subsystem: "scheduler",
			Name:      "task",
			Help:      "count of tasks that have been scheduled",
		},
		taskMetricLabels,
	),
	ErrorCounter: promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "hub",
			Subsystem: "scheduler",
			Name:      "error",
			Help:      "count of errors while scheduling",
		},
		taskMetricLabels,
	),
}
