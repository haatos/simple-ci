package store

type Agent struct {
	AgentID           int64
	AgentCredentialID *int64
	Name              string
	Hostname          string
	Workspace         string
	Description       string
	OSType            string
}
