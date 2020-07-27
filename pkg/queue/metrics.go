package queue

import (
	"os"
	"path/filepath"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

// TaskQueueMetricsType provides access to the prometheus metric objects for the task queue
type TaskQueueMetricsType struct {
	Labels          []string
	TaskCounter     *prometheus.CounterVec
	EnqueueDuration *prometheus.HistogramVec
}

// TaskQueueMetricsType provides access to the prometheus metric objects for the scheduler
type SchedulerMetricsType struct {
	Labels          []string
	ScheduleCounter *prometheus.CounterVec
	ErrorCounter    *prometheus.CounterVec
}

const (
	instanceKey = "instance"
	serviceKey  = "service"
)

var (
	// largest bucket is 5 seconds
	durationMsBuckets = []float64{10, 50, 100, 200, 300, 500, 1000, 2000, 3000, 5000}
	processName       = filepath.Base(os.Args[0])
	constLabels       = prometheus.Labels{
		serviceKey:  processName,
		instanceKey: getHostname(),
	}
	queueMetricLabels = []string{"queue"}
	taskMetricLabels  = []string{"queue", "type"}

	defTaskCounterOpts = prometheus.CounterOpts{
		Namespace:   "queue",
		Subsystem:   "task",
		Name:        "total_count",
		Help:        "count of tasks that have been enqueued",
		ConstLabels: constLabels,
	}
	defEnqueueDurationOpts = prometheus.HistogramOpts{
		Namespace:   "queue",
		Subsystem:   "task",
		Name:        "enqueue_duration_ms",
		Help:        "duration the enqueue action in ms",
		Buckets:     durationMsBuckets,
		ConstLabels: constLabels,
	}
	defScheduleCounterOpts = prometheus.CounterOpts{
		Namespace:   "queue",
		Subsystem:   "scheduler",
		Name:        "total_scheduled",
		Help:        "count of tasks that have been scheduled",
		ConstLabels: constLabels,
	}

	defSchedulerErrorCounterOpts = prometheus.CounterOpts{
		Namespace:   "queue",
		Subsystem:   "scheduler",
		Name:        "total_errors",
		Help:        "count of errors while scheduling",
		ConstLabels: constLabels,
	}

	// TaskQueueMetrics is the global metrics instance for the task queue of this instance
	TaskQueueMetrics = TaskQueueMetricsType{
		Labels: queueMetricLabels,
		TaskCounter: promauto.NewCounterVec(
			defTaskCounterOpts,
			taskMetricLabels,
		),
		EnqueueDuration: promauto.NewHistogramVec(
			defEnqueueDurationOpts,
			queueMetricLabels,
		),
	}

	// SchedulerMetrics is the global metrics instance for the scheduler of this instance
	SchedulerMetrics = SchedulerMetricsType{
		Labels: queueMetricLabels,
		ScheduleCounter: promauto.NewCounterVec(
			defScheduleCounterOpts,
			taskMetricLabels,
		),
		ErrorCounter: promauto.NewCounterVec(
			defSchedulerErrorCounterOpts,
			taskMetricLabels,
		),
	}
)

// SwitchMetricsServiceName changes the service label used in the metrics,
// so it can be customized
func SwitchMetricsServiceName(serviceName string) {
	newConstLabels := prometheus.Labels{
		instanceKey: constLabels[instanceKey],
		serviceKey:  serviceName,
	}
	newTaskCounterOpts := defTaskCounterOpts
	newTaskCounterOpts.ConstLabels = newConstLabels

	newEnqueueDurationOpts := defEnqueueDurationOpts
	newEnqueueDurationOpts.ConstLabels = newConstLabels

	newScheduleCounterOpts := defScheduleCounterOpts
	newScheduleCounterOpts.ConstLabels = newConstLabels

	newSchedulerErrorCounterOpts := defSchedulerErrorCounterOpts
	newSchedulerErrorCounterOpts.ConstLabels = newConstLabels

	TaskQueueMetrics = TaskQueueMetricsType{
		Labels: queueMetricLabels,
		TaskCounter: promauto.NewCounterVec(
			newTaskCounterOpts,
			taskMetricLabels,
		),
		EnqueueDuration: promauto.NewHistogramVec(
			newEnqueueDurationOpts,
			queueMetricLabels,
		),
	}

	// SchedulerMetrics is the global metrics instance for the scheduler of this instance
	SchedulerMetrics = SchedulerMetricsType{
		Labels: queueMetricLabels,
		ScheduleCounter: promauto.NewCounterVec(
			newScheduleCounterOpts,
			taskMetricLabels,
		),
		ErrorCounter: promauto.NewCounterVec(
			newSchedulerErrorCounterOpts,
			taskMetricLabels,
		),
	}
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		logrus.Errorf("unable to retrieve hostname - setting to unknown")
		hostname = "unknown"
	}

	return hostname
}
