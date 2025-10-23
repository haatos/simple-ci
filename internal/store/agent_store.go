package store

import "context"

type Agent struct {
	AgentID           int64
	AgentCredentialID int64
	Name              string
	Hostname          string
	Workspace         string
	Description       string
	OSType            string
}

type AgentStore interface {
	CreateAgent(context.Context, int64, string, string, string, string, string) (*Agent, error)
	ReadAgentByID(context.Context, int64) (*Agent, error)
	UpdateAgent(context.Context, int64, int64, string, string, string, string, string) error
	DeleteAgent(context.Context, int64) error
	ListAgents(context.Context) ([]*Agent, error)
}
