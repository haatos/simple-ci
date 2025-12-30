package store

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
