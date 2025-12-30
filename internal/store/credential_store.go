package store

type Credential struct {
	CredentialID      int64
	Username          string
	Description       string
	SSHPrivateKeyHash string

	SSHPrivateKey []byte
}
