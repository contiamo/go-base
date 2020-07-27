package queue

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

var (
	// largest bucket is 5 seconds
	durationMsBuckets  = []float64{10, 50, 100, 200, 300, 500, 1000, 2000, 3000, 5000}
	queueMetricLabels  = []string{"queue"}
	taskMetricLabels   = []string{"queue", "type"}
	processName        = escapeNamespace(filepath.Base(os.Args[0]))
	defTaskCounterOpts = prometheus.CounterOpts{
		Namespace: processName,
		Subsystem: "queue",
		Name:      "task",
		Help:      "count of tasks that have been enqueued",
	}
	defEnqueueDurationOpts = prometheus.HistogramOpts{
		Namespace: processName,
		Subsystem: "queue",
		Name:      "enqueue_duration_ms",
		Help:      "duration the enqueue action in ms",
		Buckets:   durationMsBuckets,
	}
	defScheduleCounterOpts = prometheus.CounterOpts{
		Namespace: processName,
		Subsystem: "scheduler",
		Name:      "task",
		Help:      "count of tasks that have been scheduled",
	}

	defSchedulerErrorCounterOpts = prometheus.CounterOpts{
		Namespace: processName,
		Subsystem: "scheduler",
		Name:      "error",
		Help:      "count of errors while scheduling",
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

// SwitchMetricsNamespace changes the namespace used in the metrics, so it can be customized
func SwitchMetricsNamespace(namespace string) {
	newTaskCounterOpts := defTaskCounterOpts
	newTaskCounterOpts.Namespace = namespace

	newEnqueueDurationOpts := defEnqueueDurationOpts
	newEnqueueDurationOpts.Namespace = namespace

	newScheduleCounterOpts := defScheduleCounterOpts
	newScheduleCounterOpts.Namespace = namespace

	newSchedulerErrorCounterOpts := defSchedulerErrorCounterOpts
	newSchedulerErrorCounterOpts.Namespace = namespace

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

func escapeNamespace(str string) string {
	str = strings.ToLower(str)
	result := strings.Builder{}
	for _, rune := range str {
		if !unicode.IsLetter(rune) && !unicode.IsDigit(rune) {
			result.WriteRune('_')
		} else {
			result.WriteRune(rune)
		}
	}

	if result.Len() == 0 {
		return "default"
	}

	return result.String()
}
