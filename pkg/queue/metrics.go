package queue

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// largest bucket is 5 seconds
var durationMsBuckets = []float64{10, 50, 100, 200, 300, 500, 1000, 2000, 3000, 5000}

var (
	queueMetricLabels        = []string{"queue"}
	taskMetricLabels         = []string{"queue", "type"}
	oneTaskQueueMetricsSetup sync.Once
	oneSchedulerMetricsSetup sync.Once
)

// TaskQueueMetricsType provides access to the prometheus metric objects for the task queue
type TaskQueueMetricsType struct {
	Labels          []string
	TaskCounter     *prometheus.CounterVec
	EnqueueDuration *prometheus.HistogramVec
}

// TaskQueueMetrics is the global metrics instance for the task queue of this instance
var TaskQueueMetrics TaskQueueMetricsType

// SetupTaskQueueMetrics must be called before any other call to the metric subsystem happens
func SetupTaskQueueMetrics(namespace string) {
	oneTaskQueueMetricsSetup.Do(func() {
		TaskQueueMetrics = TaskQueueMetricsType{
			Labels: queueMetricLabels,
			TaskCounter: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "queue",
					Name:      "task",
					Help:      "count of tasks that have been enqueued",
				},
				taskMetricLabels,
			),
			EnqueueDuration: promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Namespace: namespace,
					Subsystem: "queue",
					Name:      "enqueue_duration_ms",
					Help:      "duration the enqueue action in ms",
					Buckets:   durationMsBuckets,
				},
				queueMetricLabels,
			),
		}
	})
}

// TaskQueueMetricsType provides access to the prometheus metric objects for the scheduler
type SchedulerMetricsType struct {
	Labels          []string
	ScheduleCounter *prometheus.CounterVec
	ErrorCounter    *prometheus.CounterVec
}

// SchedulerMetrics is the global metrics instance for the scheduler of this instance
var SchedulerMetrics SchedulerMetricsType

// SetupSchedulerMetrics must be called before any other call to the metric subsystem happens
func SetupSchedulerMetrics(namespace string) {
	oneSchedulerMetricsSetup.Do(func() {
		SchedulerMetrics = SchedulerMetricsType{
			Labels: queueMetricLabels,
			ScheduleCounter: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "scheduler",
					Name:      "task",
					Help:      "count of tasks that have been scheduled",
				},
				taskMetricLabels,
			),
			ErrorCounter: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: namespace,
					Subsystem: "scheduler",
					Name:      "error",
					Help:      "count of errors while scheduling",
				},
				taskMetricLabels,
			),
		}
	})
}
