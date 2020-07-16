package workers

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// largest bucket is 83 min
var (
	durationSBuckets  = []float64{1, 10, 50, 100, 200, 300, 500, 1000, 2000, 3000, 5000}
	durationMsBuckets = []float64{10, 50, 100, 200, 300, 500, 1000, 2000, 3000, 5000}
	queueMetricLabels = []string{"queue"}
	taskMetricLabels  = []string{"queue", "type"}
)

// TaskQueueMetrics provides access to the prometheus metric objects for the task queue
var TaskQueueMetrics = struct {
	Labels             []string
	TaskDuration       *prometheus.HistogramVec
	TaskWaiting        *prometheus.HistogramVec
	DequeueDuration    prometheus.Histogram
	WorkerGauge        prometheus.Gauge
	WorkerWaiting      prometheus.Counter
	WorkerWorking      *prometheus.CounterVec
	WorkerWorkingGauge *prometheus.GaugeVec
	WorkerTask         *prometheus.CounterVec
	WorkerErrors       *prometheus.CounterVec
	QueueErrors        prometheus.Counter
}{
	Labels: queueMetricLabels,
	TaskDuration: promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "hub",
			Subsystem: "worker",
			Name:      "duration_s",
			Help:      "duration of a task in seconds by queue",
			Buckets:   durationSBuckets,
		},
		taskMetricLabels,
	),
	TaskWaiting: promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "hub",
			Subsystem: "queue",
			Name:      "waiting_duration_s",
			Help:      "duration of a task waiting to start",
			Buckets:   durationSBuckets,
		},
		taskMetricLabels,
	),
	DequeueDuration: promauto.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "hub",
			Subsystem: "worker",
			Name:      "wait_duration_s",
			Help:      "duration in seconds a worker spent waiting",
			Buckets:   durationSBuckets,
		},
	),
	WorkerGauge: promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "hub",
		Subsystem: "worker",
		Name:      "count",
		Help:      "count of initialized workers",
	}),
	WorkerWaiting: promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hub",
		Subsystem: "worker",
		Name:      "waiting_count",
		Help:      "count of workers waiting for a task",
	}),
	WorkerWorking: promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "hub",
			Subsystem: "worker",
			Name:      "working_total",
			Help:      "count of workers working on a task",
		},
		taskMetricLabels,
	),
	WorkerWorkingGauge: promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "hub",
			Subsystem: "worker",
			Name:      "working",
			Help:      "count of working workers",
		},
		taskMetricLabels,
	),
	WorkerTask: promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "hub",
			Subsystem: "worker",
			Name:      "task_total",
			Help:      "count of tasks seen by the worker",
		},
		taskMetricLabels,
	),
	WorkerErrors: promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "hub",
			Subsystem: "worker",
			Name:      "error_total",
			Help:      "count of errors seen by the worker",
		},
		taskMetricLabels,
	),
	QueueErrors: promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hub",
		Subsystem: "worker",
		Name:      "queue_error_total",
		Help:      "count of dequeue errors seen by the worker",
	}),
}

// TaskSchedulingMetrics provides access to the prometheus metric objects for task scheduling
var TaskSchedulingMetrics = struct {
	Labels            []string
	IterationDuration prometheus.Histogram
	WorkerGauge,
	WorkerWorkingGauge prometheus.Gauge
	WorkerWaiting,
	WorkerWorking,
	WorkerErrors prometheus.Counter
	WorkerTaskScheduled *prometheus.CounterVec
}{
	Labels: queueMetricLabels,
	IterationDuration: promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "hub",
		Subsystem: "scheduling",
		Name:      "iteration_duration_ms",
		Help:      "duration of the scheduling iteration in ms",
		Buckets:   durationMsBuckets,
	}),
	WorkerGauge: promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "hub",
		Subsystem: "scheduling",
		Name:      "count",
		Help:      "count of initialized task scheduling workers",
	}),
	WorkerWorkingGauge: promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "hub",
		Subsystem: "scheduling",
		Name:      "working_count",
		Help:      "count of working task scheduling workers",
	}),
	WorkerWaiting: promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hub",
		Subsystem: "scheduling",
		Name:      "waiting_total",
		Help:      "count of workers waiting for a task to schedule",
	}),
	WorkerWorking: promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hub",
		Subsystem: "scheduling",
		Name:      "working_total",
		Help:      "count of workers working on a task scheduling",
	}),
	WorkerErrors: promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "hub",
		Subsystem: "scheduling",
		Name:      "error_total",
		Help:      "count of task scheduling errors seen by the worker",
	}),
	WorkerTaskScheduled: promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "hub",
			Subsystem: "scheduling",
			Name:      "task_total",
			Help:      "count of tasks scheduled by the worker",
		},
		taskMetricLabels,
	),
}
