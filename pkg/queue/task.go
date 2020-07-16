package queue

import "time"

// References is a dictionary of additinal SQL columns to set.
// Normally this dictionary contains referencies to other entities
// e.g. datasource_id, table_id, schedule_id, etc.
// So, the SQL table can have the `DELETE CASCADE` setting.
type References map[string]interface{}

// GetNamesAndValues returns a slice of column names and a slice of values accordingly.
// It's to easily use it in the SQL builder.
func (a References) GetNamesAndValues() (keys []string, values []interface{}) {
	if a == nil {
		return keys, values
	}
	l := len(a)
	keys, values = make([]string, 0, l), make([]interface{}, 0, l)
	for k, v := range a {
		keys = append(keys, k)
		values = append(values, v)
	}

	return keys, values
}

// Progress is a serialized status object that indicates the status of the task
// For example, it can be an object that contains the amount of processed bytes.
// The actual underlying type depends on the task type and consumers must serialize the value.
type Progress []byte

// Spec is is a serialized specification object used to set the task parameters.
// The actual underlying type depends on the task type and consumers must serialize the value.
type Spec []byte

// TaskType is a type which specifies what a task it is
type TaskType string

// String returns a string value of the task type
func (t TaskType) String() string {
	return string(t)
}

// TaskStatus : Current state of the task
type TaskStatus string

// List of TaskStatus
const (
	// Waiting task is waiting for being picked up
	Waiting TaskStatus = "waiting"
	// Running task is currently running
	Running TaskStatus = "running"
	// Cancelled task is cancelled by the user
	Cancelled TaskStatus = "cancelled"
	// Finished task is successfully finished
	Finished TaskStatus = "finished"
	// Failed task is failed with an error
	Failed TaskStatus = "failed"
)

// TaskBase contains basic fields for a task
type TaskBase struct {
	// Queue is the queue to which the task belongs
	Queue string
	// Type holds a task type identifier
	Type TaskType
	// Spec contains the task specification based on type of the task.
	Spec Spec
}

// TaskEnqueueRequest contains fields required for adding a task to the queue
type TaskEnqueueRequest struct {
	TaskBase
	// References contain names and values for additinal
	// SQL columns to set external references for a task for easy clean up
	References References
}

// TaskScheduleRequest contains fields required for scheduling a task
type TaskScheduleRequest struct {
	TaskBase
	// CronSchedule is the schedule impression in cron syntax that defines
	// when the task should be executed.
	CronSchedule string
	// References contain names and values for additinal
	// SQL columns to set external references for a schedule for easy clean up
	References References
}

// Task represents a task in the queue
type Task struct {
	TaskBase
	// ID is the id of the task
	ID string
	// Status is the current status of the task
	Status TaskStatus
	// Progress is used for workers to update the status of the task.
	// For example, bytes processed.
	// The actual underlying type is specific for each task type.
	Progress Progress
	// CreatedAt is when the task was initially put in the queue
	CreatedAt time.Time
	// CreatedAt is when the task was initially put in the queue
	UpdatedAt time.Time
	// StartedAt is when a worker picked up the task
	StartedAt *time.Time
	// FinishedAt is when the task was finished being processed
	FinishedAt *time.Time
	// LastHeartbeat provides a way to ensure the task is still being processed and hasn't failed.
	LastHeartbeatAt *time.Time
}
