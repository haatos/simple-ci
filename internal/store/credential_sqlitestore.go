package store

import (
	"context"
	"database/sql"

	"github.com/georgysavva/scany/v2/sqlscan"
)

type CredentialSQLiteStore struct {
	rdb, rwdb *sql.DB
}

func NewCredentialSQLiteStore(rdb, rwdb *sql.DB) *CredentialSQLiteStore {
	return &CredentialSQLiteStore{rdb, rwdb}
}

func (store *CredentialSQLiteStore) CreateCredential(
	ctx context.Context,
	username, description, sshPrivateKeyHash string,
) (*Credential, error) {
	c := &Credential{
		Username:          username,
		Description:       description,
		SSHPrivateKeyHash: sshPrivateKeyHash,
	}
	query := `insert into credentials (
		username,
		description,
		ssh_private_key_hash
	)
	values ($1, $2, $3)
	returning credential_id`
	err := sqlscan.Get(ctx, store.rwdb, c, query, c.Username, c.Description, c.SSHPrivateKeyHash)
	return c, err
}

func (store *CredentialSQLiteStore) ReadCredentialByID(
	ctx context.Context,
	credentialID int64,
) (*Credential, error) {
	c := new(Credential)
	c.CredentialID = credentialID
	query := `select * from credentials where credential_id = $1`
	err := sqlscan.Get(ctx, store.rdb, c, query, c.CredentialID)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (store *CredentialSQLiteStore) UpdateCredential(
	ctx context.Context,
	credentialID int64,
	username, description string,
) error {
	query := `update credentials
	set username = $1,
		description = $2
	where credential_id = $3`
	_, err := store.rwdb.ExecContext(ctx, query, username, description, credentialID)
	return err
}

func (store *CredentialSQLiteStore) DeleteCredential(
	ctx context.Context,
	credentialID int64,
) error {
	query := `delete from credentials where credential_id = $1`
	_, err := store.rwdb.ExecContext(ctx, query, credentialID)
	return err
}

func (store *CredentialSQLiteStore) ListCredentials(ctx context.Context) ([]*Credential, error) {
	query := `select credential_id, username, description, ssh_private_key_hash from credentials`
	credentials := make([]*Credential, 0)
	err := sqlscan.Select(ctx, store.rdb, &credentials, query)
	return credentials, err
}
