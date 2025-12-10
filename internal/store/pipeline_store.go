package store

import (
	"context"
)

type Pipeline struct {
	PipelineID      int64
	PipelineAgentID int64
	Name            string
	Description     string
	// Git repository path
	Repository string
	// Pipeline script path within the repository
	ScriptPath string
	// Pipeline schedule in cron syntax
	Schedule *string
	// Git branch for scheduled run
	ScheduleBranch *string
	// Schduled job ID
	ScheduleJobID *string
}

type PipelineRunData struct {
	PipelineID        int64
	AgentID           int64
	OSType            string
	CredentialID      *int64
	Repository        string
	ScriptPath        string
	Hostname          string
	Workspace         string
	Username          *string
	SSHPrivateKeyHash *string
	SSHPrivateKey     []byte
}

type PipelineStore interface {
	CreatePipeline(
		context.Context,
		int64,
		string,
		string,
		string,
		string,
	) (*Pipeline, error)
	ReadPipelineByID(context.Context, int64) (*Pipeline, error)
	ReadPipelineRunData(context.Context, int64) (*PipelineRunData, error)
	UpdatePipeline(context.Context, int64, int64, string, string, string, string) error
	UpdatePipelineSchedule(context.Context, int64, *string, *string, *string) error
	UpdatePipelineScheduleJobID(context.Context, int64, *string) error
	DeletePipeline(context.Context, int64) error
	ListPipelines(context.Context) ([]*Pipeline, error)
	ListScheduledPipelines(context.Context) ([]*Pipeline, error)
}
