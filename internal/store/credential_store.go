package store

import "context"

type Credential struct {
	CredentialID      int64
	Username          string
	Description       string
	SSHPrivateKeyHash string

	SSHPrivateKey []byte
}

type CredentialStore interface {
	CreateCredential(context.Context, string, string, string) (*Credential, error)
	ReadCredentialByID(context.Context, int64) (*Credential, error)
	UpdateCredential(context.Context, int64, string, string) error
	DeleteCredential(context.Context, int64) error
	ListCredentials(context.Context) ([]*Credential, error)
}
