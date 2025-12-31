package store

import (
	"time"
)

type RunStatus string

const (
	StatusQueued    RunStatus = "queued"
	StatusRunning   RunStatus = "running"
	StatusCancelled RunStatus = "cancelled"
	StatusFailed    RunStatus = "failed"
	StatusPassed    RunStatus = "passed"
)

type Run struct {
	RunID            int64 `param:"run_id"`
	RunPipelineID    int64
	Branch           string
	WorkingDirectory *string
	Output           *string
	Artifacts        *string
	Status           RunStatus
	CreatedOn        time.Time
	StartedOn        *time.Time
	EndedOn          *time.Time
	Archive          bool

	PipelineName string
}
