package model

import "time"

// TaskState stores runtime state of a registered task definition.
// Name is the primary key (matches TaskDef.Name from code registration).
type TaskState struct {
	Name        string    `json:"name" gorm:"primaryKey;size:100"`
	Type        string    `json:"type" gorm:"size:20;not null"`        // scheduled | async
	Description string    `json:"description" gorm:"size:500"`
	CronExpr    string    `json:"cronExpr,omitempty" gorm:"size:100"`
	TimeoutMs   int       `json:"timeoutMs" gorm:"not null;default:30000"`
	MaxRetries  int       `json:"maxRetries" gorm:"not null;default:3"`
	Status      string    `json:"status" gorm:"size:20;not null;default:active"` // active | paused
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TaskExecution records each execution of a task.
type TaskExecution struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	TaskName   string     `json:"taskName" gorm:"size:100;not null;index:idx_task_exec_name"`
	Trigger    string     `json:"trigger" gorm:"size:20;not null"`                          // cron | manual | api
	Status     string     `json:"status" gorm:"size:20;not null;index:idx_task_exec_status"` // pending | running | completed | failed | timeout | stale
	Payload    string     `json:"payload,omitempty" gorm:"type:text"`
	Result     string     `json:"result,omitempty" gorm:"type:text"`
	Error      string     `json:"error,omitempty" gorm:"type:text"`
	RetryCount int        `json:"retryCount" gorm:"not null;default:0"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	CreatedAt  time.Time  `json:"createdAt"`
}
