package handler

import "github.com/haatos/simple-ci/internal/types"

type CredentialParams struct {
	CredentialID  int64  `form:"credential_id"   param:"credential_id"`
	Username      string `form:"username"`
	Description   string `form:"description"`
	SSHPrivateKey string `form:"ssh_private_key"`
}

type AgentParams struct {
	AgentID           int64  `form:"agent_id"            param:"agent_id"`
	AgentCredentialID int64  `form:"agent_credential_id"`
	Name              string `form:"name"`
	Hostname          string `form:"hostname"`
	Workspace         string `form:"workspace"`
	Description       string `form:"description"`
	OSType            string `form:"os_type"`
}

type PipelineParams struct {
	PipelineID      int64   `form:"pipeline_id"       param:"pipeline_id"`
	PipelineAgentID int64   `form:"pipeline_agent_id"`
	Name            string  `form:"name"`
	Description     string  `form:"description"`
	Repository      string  `form:"repository"`
	ScriptPath      string  `form:"script_path"`
	Schedule        *string `form:"schedule"`
	ScheduleBranch  *string `form:"schedule_branch"`
	ScheduleJobID   *string `form:"schedule_job_id"`
}

type RunParams struct {
	PipelineID int64  `param:"pipeline_id"`
	RunID      int64  `param:"run_id"`
	Branch     string `param:"branch"      form:"branch"`
}

type ListRunsParams struct {
	PipelineID int64 `param:"pipeline_id"`
	Page       int64 `                    query:"page"`
}

type PatchUserParams struct {
	UserID int64      `param:"user_id"`
	RoleID types.Role `                form:"role_id"`
}

type PatchUserPasswordParams struct {
	UserID          int64  `param:"user_id" form:"user_id"`
	OldPassword     string `                form:"old_password"`
	Password        string `                form:"password"`
	PasswordConfirm string `                form:"password_confirm"`
}

type UserParams struct {
	UserID          int64      `param:"user_id"`
	UserRoleID      types.Role `                form:"user_role_id"`
	Username        string     `                form:"username"`
	Password        string     `                form:"password"`
	PasswordConfirm string     `                form:"password_confirm"`
}

type APIKeyParams struct {
	ID int64 `param:"id"`
}

type ConfigParams struct {
	SessionExpiresHours int64 `form:"session_expires_hours"`
	QueueSize           int64 `form:"queue_size"`
}
