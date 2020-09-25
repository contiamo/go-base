package queue

import (
	"os"
	"path/filepath"
	"sync"

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

// WorkerMetricsType provides access to the prometheus metric objects for a worker
type WorkerMetricsType struct {
	Labels []string

	ActiveGauge     prometheus.Gauge
	WorkingGauge    prometheus.Gauge
	WaitingGauge    prometheus.Gauge
	DequeueingGauge prometheus.Gauge

	ProcessingDuration prometheus.Histogram
	DequeueingDuration prometheus.Histogram

	ProcessedCounter    *prometheus.CounterVec
	DequeueErrorCounter prometheus.Counter
	ErrorsCounter       prometheus.Counter
}

const (
	instanceKey = "instance"
	serviceKey  = "service"
)

var (
	// largest bucket is 5 seconds
	durationMsBuckets = []float64{10, 50, 100, 200, 300, 500, 1000, 2000, 3000, 5000}
	// largest bucket is 83 min
	durationSBuckets = []float64{1, 10, 50, 100, 200, 300, 500, 1000, 2000, 3000, 5000}
	mutex            = &sync.Mutex{}
	processName      = filepath.Base(os.Args[0])
	constLabels      = prometheus.Labels{
		serviceKey:  processName,
		instanceKey: getHostname(),
	}
	queueMetricLabels = []string{"queue"}
	taskMetricLabels  = []string{"queue", "type"}

	// handler metrics definitions

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
		Help:        "duration of enqueueing in ms",
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

	// worker metrics definitions

	// task worker

	defTaskWorkerActiveGaugeOpts = prometheus.GaugeOpts{
		Namespace:   "queue",
		Subsystem:   "task_worker",
		Name:        "active_gauge",
		Help:        "gauge of active instances",
		ConstLabels: constLabels,
	}
	defTaskWorkerWorkingGaugeOpts = prometheus.GaugeOpts{
		Namespace:   "queue",
		Subsystem:   "task_worker",
		Name:        "working_gauge",
		Help:        "gauge of working instances",
		ConstLabels: constLabels,
	}
	defTaskWorkerWaitingGaugeOpts = prometheus.GaugeOpts{
		Namespace:   "queue",
		Subsystem:   "task_worker",
		Name:        "waiting_gauge",
		Help:        "gauge of waiting instances",
		ConstLabels: constLabels,
	}
	defTaskWorkerDequeueingGaugeOpts = prometheus.GaugeOpts{
		Namespace:   "queue",
		Subsystem:   "task_worker",
		Name:        "dequeueing_gauge",
		Help:        "gauge of instances that are trying to dequeue a task",
		ConstLabels: constLabels,
	}
	defTaskWorkerProcessingDurationOpts = prometheus.HistogramOpts{
		Namespace:   "queue",
		Subsystem:   "task_worker",
		Name:        "processing_duration_s",
		Help:        "total duration of tasks being processed in seconds",
		Buckets:     durationSBuckets,
		ConstLabels: constLabels,
	}
	defTaskWorkerDequeueingDurationOpts = prometheus.HistogramOpts{
		Namespace:   "queue",
		Subsystem:   "task_worker",
		Name:        "dequeueing_duration_s",
		Help:        "total duration spent waiting for a new task in seconds",
		Buckets:     durationSBuckets,
		ConstLabels: constLabels,
	}
	defTaskWorkerProcessedCounterOpts = prometheus.CounterOpts{
		Namespace:   "queue",
		Subsystem:   "task_worker",
		Name:        "task_count",
		Help:        "total count of processed tasks",
		ConstLabels: constLabels,
	}
	defTaskWorkerDequeueErrorsCounterOpts = prometheus.CounterOpts{
		Namespace:   "queue",
		Subsystem:   "task_worker",
		Name:        "dequeue_error_count",
		Help:        "total count of dequeueing errors",
		ConstLabels: constLabels,
	}
	defTaskWorkerErrorsCounterOpts = prometheus.CounterOpts{
		Namespace:   "queue",
		Subsystem:   "task_worker",
		Name:        "error_count",
		Help:        "total count of errors",
		ConstLabels: constLabels,
	}

	// schedule worker

	defScheduleWorkerActiveGaugeOpts = prometheus.GaugeOpts{
		Namespace:   "queue",
		Subsystem:   "schedule_worker",
		Name:        "active_gauge",
		Help:        "gauge of active instances",
		ConstLabels: constLabels,
	}
	defScheduleWorkerWorkingGaugeOpts = prometheus.GaugeOpts{
		Namespace:   "queue",
		Subsystem:   "schedule_worker",
		Name:        "working_gauge",
		Help:        "gauge of working instances",
		ConstLabels: constLabels,
	}
	defScheduleWorkerWaitingGaugeOpts = prometheus.GaugeOpts{
		Namespace:   "queue",
		Subsystem:   "schedule_worker",
		Name:        "waiting_gauge",
		Help:        "gauge of waiting instances",
		ConstLabels: constLabels,
	}
	defScheduleWorkerDequeueingGaugeOpts = prometheus.GaugeOpts{
		Namespace:   "queue",
		Subsystem:   "schedule_worker",
		Name:        "dequeueing_gauge",
		Help:        "gauge of instances that are trying to dequeue a schedule",
		ConstLabels: constLabels,
	}
	defScheduleWorkerProcessingDurationOpts = prometheus.HistogramOpts{
		Namespace:   "queue",
		Subsystem:   "schedule_worker",
		Name:        "processing_duration_s",
		Help:        "total duration of tasks being scheduled in seconds",
		Buckets:     durationSBuckets,
		ConstLabels: constLabels,
	}
	defScheduleWorkerDequeueingDurationOpts = prometheus.HistogramOpts{
		Namespace:   "queue",
		Subsystem:   "schedule_worker",
		Name:        "dequeueing_duration_s",
		Help:        "total duration spent waiting for a new task to schedule in seconds",
		Buckets:     durationSBuckets,
		ConstLabels: constLabels,
	}
	defScheduleWorkerProcessedCounterOpts = prometheus.CounterOpts{
		Namespace:   "queue",
		Subsystem:   "schedule_worker",
		Name:        "task_count",
		Help:        "total count of processed schedules",
		ConstLabels: constLabels,
	}
	defScheduleWorkerDequeueErrorsCounterOpts = prometheus.CounterOpts{
		Namespace:   "queue",
		Subsystem:   "schedule_worker",
		Name:        "dequeue_error_count",
		Help:        "total count of schedule dequeueing errors",
		ConstLabels: constLabels,
	}
	defScheduleWorkerErrorsCounterOpts = prometheus.CounterOpts{
		Namespace:   "queue",
		Subsystem:   "schedule_worker",
		Name:        "error_count",
		Help:        "total count of errors",
		ConstLabels: constLabels,
	}

	// public interface

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

	// TaskWorkerMetrics is the global metrics instance for the task worker of this instance
	TaskWorkerMetrics = WorkerMetricsType{
		Labels: taskMetricLabels,

		ActiveGauge:     promauto.NewGauge(defTaskWorkerActiveGaugeOpts),
		WorkingGauge:    promauto.NewGauge(defTaskWorkerWorkingGaugeOpts),
		WaitingGauge:    promauto.NewGauge(defTaskWorkerWaitingGaugeOpts),
		DequeueingGauge: promauto.NewGauge(defTaskWorkerDequeueingGaugeOpts),

		ProcessingDuration: promauto.NewHistogram(defTaskWorkerProcessingDurationOpts),
		DequeueingDuration: promauto.NewHistogram(defTaskWorkerDequeueingDurationOpts),

		ProcessedCounter:    promauto.NewCounterVec(defTaskWorkerProcessedCounterOpts, taskMetricLabels),
		DequeueErrorCounter: promauto.NewCounter(defTaskWorkerDequeueErrorsCounterOpts),
		ErrorsCounter:       promauto.NewCounter(defTaskWorkerErrorsCounterOpts),
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

	// ScheduleWorkerMetrics is the global metrics instance for the schedule worker of this instance
	ScheduleWorkerMetrics = WorkerMetricsType{
		Labels: taskMetricLabels,

		ActiveGauge:     promauto.NewGauge(defScheduleWorkerActiveGaugeOpts),
		WorkingGauge:    promauto.NewGauge(defScheduleWorkerWorkingGaugeOpts),
		WaitingGauge:    promauto.NewGauge(defScheduleWorkerWaitingGaugeOpts),
		DequeueingGauge: promauto.NewGauge(defScheduleWorkerDequeueingGaugeOpts),

		ProcessingDuration: promauto.NewHistogram(defScheduleWorkerProcessingDurationOpts),
		DequeueingDuration: promauto.NewHistogram(defScheduleWorkerDequeueingDurationOpts),

		ProcessedCounter:    promauto.NewCounterVec(defScheduleWorkerProcessedCounterOpts, taskMetricLabels),
		DequeueErrorCounter: promauto.NewCounter(defScheduleWorkerDequeueErrorsCounterOpts),
		ErrorsCounter:       promauto.NewCounter(defScheduleWorkerErrorsCounterOpts),
	}
)

// SwitchMetricsServiceName changes the service label used in the metrics,
// so it can be customized with a different name
func SwitchMetricsServiceName(serviceName string) {
	newConstLabels := prometheus.Labels{
		instanceKey: constLabels[instanceKey],
		serviceKey:  serviceName,
	}

	// task queue

	newTaskCounterOpts := defTaskCounterOpts
	newTaskCounterOpts.ConstLabels = newConstLabels

	newEnqueueDurationOpts := defEnqueueDurationOpts
	newEnqueueDurationOpts.ConstLabels = newConstLabels

	// scheduler queue

	newScheduleCounterOpts := defScheduleCounterOpts
	newScheduleCounterOpts.ConstLabels = newConstLabels

	newSchedulerErrorCounterOpts := defSchedulerErrorCounterOpts
	newSchedulerErrorCounterOpts.ConstLabels = newConstLabels

	// task worker

	newTaskWorkerActiveGaugeOpts := defTaskWorkerActiveGaugeOpts
	newTaskWorkerActiveGaugeOpts.ConstLabels = newConstLabels

	newTaskWorkerWorkingGaugeOpts := defTaskWorkerWorkingGaugeOpts
	newTaskWorkerWorkingGaugeOpts.ConstLabels = newConstLabels

	newTaskWorkerWaitingGaugeOpts := defTaskWorkerWaitingGaugeOpts
	newTaskWorkerWaitingGaugeOpts.ConstLabels = newConstLabels

	newTaskWorkerDequeueingGaugeOpts := defTaskWorkerDequeueingGaugeOpts
	newTaskWorkerDequeueingGaugeOpts.ConstLabels = newConstLabels

	newTaskWorkerProcessingDurationOpts := defTaskWorkerProcessingDurationOpts
	newTaskWorkerProcessingDurationOpts.ConstLabels = newConstLabels

	newTaskWorkerDequeueingDurationOpts := defTaskWorkerDequeueingDurationOpts
	newTaskWorkerDequeueingDurationOpts.ConstLabels = newConstLabels

	newTaskWorkerProcessedCounterOpts := defTaskWorkerProcessedCounterOpts
	newTaskWorkerProcessedCounterOpts.ConstLabels = newConstLabels

	newTaskWorkerDequeueErrorsCounterOpts := defTaskWorkerDequeueErrorsCounterOpts
	newTaskWorkerDequeueErrorsCounterOpts.ConstLabels = newConstLabels

	newTaskWorkerErrorsCounterOpts := defTaskWorkerErrorsCounterOpts
	newTaskWorkerErrorsCounterOpts.ConstLabels = newConstLabels

	// schedule worker

	newScheduleWorkerActiveGaugeOpts := defScheduleWorkerActiveGaugeOpts
	newScheduleWorkerActiveGaugeOpts.ConstLabels = newConstLabels

	newScheduleWorkerWorkingGaugeOpts := defScheduleWorkerWorkingGaugeOpts
	newScheduleWorkerWorkingGaugeOpts.ConstLabels = newConstLabels

	newScheduleWorkerWaitingGaugeOpts := defScheduleWorkerWaitingGaugeOpts
	newScheduleWorkerWaitingGaugeOpts.ConstLabels = newConstLabels

	newScheduleWorkerDequeueingGaugeOpts := defScheduleWorkerDequeueingGaugeOpts
	newScheduleWorkerDequeueingGaugeOpts.ConstLabels = newConstLabels

	newScheduleWorkerProcessingDurationOpts := defScheduleWorkerProcessingDurationOpts
	newScheduleWorkerProcessingDurationOpts.ConstLabels = newConstLabels

	newScheduleWorkerDequeueingDurationOpts := defScheduleWorkerDequeueingDurationOpts
	newScheduleWorkerDequeueingDurationOpts.ConstLabels = newConstLabels

	newScheduleWorkerProcessedCounterOpts := defScheduleWorkerProcessedCounterOpts
	newScheduleWorkerProcessedCounterOpts.ConstLabels = newConstLabels

	newScheduleWorkerDequeueErrorsCounterOpts := defScheduleWorkerDequeueErrorsCounterOpts
	newScheduleWorkerDequeueErrorsCounterOpts.ConstLabels = newConstLabels

	newScheduleWorkerErrorsCounterOpts := defScheduleWorkerErrorsCounterOpts
	newScheduleWorkerErrorsCounterOpts.ConstLabels = newConstLabels

	mutex.Lock()
	defer mutex.Unlock()

	// task queue

	prometheus.Unregister(TaskQueueMetrics.TaskCounter)
	prometheus.Unregister(TaskQueueMetrics.EnqueueDuration)
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

	// scheduler queue

	prometheus.Unregister(SchedulerMetrics.ScheduleCounter)
	prometheus.Unregister(SchedulerMetrics.ErrorCounter)
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

	// task worker

	prometheus.Unregister(TaskWorkerMetrics.ProcessingDuration)
	prometheus.Unregister(TaskWorkerMetrics.DequeueingDuration)
	prometheus.Unregister(TaskWorkerMetrics.ActiveGauge)
	prometheus.Unregister(TaskWorkerMetrics.WorkingGauge)
	prometheus.Unregister(TaskWorkerMetrics.WaitingGauge)
	prometheus.Unregister(TaskWorkerMetrics.DequeueingGauge)
	prometheus.Unregister(TaskWorkerMetrics.ProcessedCounter)
	prometheus.Unregister(TaskWorkerMetrics.ErrorsCounter)
	prometheus.Unregister(TaskWorkerMetrics.DequeueErrorCounter)
	TaskWorkerMetrics = WorkerMetricsType{
		Labels: taskMetricLabels,

		ActiveGauge:     promauto.NewGauge(newTaskWorkerActiveGaugeOpts),
		WorkingGauge:    promauto.NewGauge(newTaskWorkerWorkingGaugeOpts),
		WaitingGauge:    promauto.NewGauge(newTaskWorkerWaitingGaugeOpts),
		DequeueingGauge: promauto.NewGauge(newTaskWorkerDequeueingGaugeOpts),

		ProcessingDuration: promauto.NewHistogram(newTaskWorkerProcessingDurationOpts),
		DequeueingDuration: promauto.NewHistogram(newTaskWorkerDequeueingDurationOpts),

		ProcessedCounter:    promauto.NewCounterVec(newTaskWorkerProcessedCounterOpts, taskMetricLabels),
		DequeueErrorCounter: promauto.NewCounter(newTaskWorkerDequeueErrorsCounterOpts),
		ErrorsCounter:       promauto.NewCounter(newTaskWorkerErrorsCounterOpts),
	}

	// schedule worker

	prometheus.Unregister(ScheduleWorkerMetrics.ProcessingDuration)
	prometheus.Unregister(ScheduleWorkerMetrics.DequeueingDuration)
	prometheus.Unregister(ScheduleWorkerMetrics.ActiveGauge)
	prometheus.Unregister(ScheduleWorkerMetrics.WorkingGauge)
	prometheus.Unregister(ScheduleWorkerMetrics.WaitingGauge)
	prometheus.Unregister(ScheduleWorkerMetrics.DequeueingGauge)
	prometheus.Unregister(ScheduleWorkerMetrics.ProcessedCounter)
	prometheus.Unregister(ScheduleWorkerMetrics.ErrorsCounter)
	prometheus.Unregister(ScheduleWorkerMetrics.DequeueErrorCounter)
	ScheduleWorkerMetrics = WorkerMetricsType{
		Labels: taskMetricLabels,

		ActiveGauge:     promauto.NewGauge(newScheduleWorkerActiveGaugeOpts),
		WorkingGauge:    promauto.NewGauge(newScheduleWorkerWorkingGaugeOpts),
		WaitingGauge:    promauto.NewGauge(newScheduleWorkerWaitingGaugeOpts),
		DequeueingGauge: promauto.NewGauge(newScheduleWorkerDequeueingGaugeOpts),

		ProcessingDuration: promauto.NewHistogram(newScheduleWorkerProcessingDurationOpts),
		DequeueingDuration: promauto.NewHistogram(newScheduleWorkerDequeueingDurationOpts),

		ProcessedCounter:    promauto.NewCounterVec(newScheduleWorkerProcessedCounterOpts, taskMetricLabels),
		DequeueErrorCounter: promauto.NewCounter(newScheduleWorkerDequeueErrorsCounterOpts),
		ErrorsCounter:       promauto.NewCounter(newScheduleWorkerErrorsCounterOpts),
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
