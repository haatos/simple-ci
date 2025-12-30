package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/haatos/simple-ci/internal/security"
	"github.com/haatos/simple-ci/internal/store"
)

type CredentialWriter interface {
	CreateCredential(context.Context, string, string, string) (*store.Credential, error)
	UpdateCredential(context.Context, int64, string, string) error
	DeleteCredential(context.Context, int64) error
}

type CredentialReader interface {
	ReadCredentialByID(context.Context, int64) (*store.Credential, error)
	ListCredentials(context.Context) ([]*store.Credential, error)
}

type CredentialStore interface {
	CredentialWriter
	CredentialReader
}

type CredentialService struct {
	credentialStore CredentialStore
	encrypter       security.Encrypter
}

func NewCredentialService(
	s CredentialStore,
	encrypter security.Encrypter,
) *CredentialService {
	return &CredentialService{credentialStore: s, encrypter: encrypter}
}

func (s *CredentialService) DecryptAES(hash string) ([]byte, error) {
	return s.encrypter.DecryptAES(hash)
}

func (s *CredentialService) CreateCredential(
	ctx context.Context,
	username, description, sshPrivateKey string,
) (*store.Credential, error) {
	hash := s.encrypter.EncryptAES(sshPrivateKey)
	c, err := s.credentialStore.CreateCredential(ctx, username, description, hash)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (s *CredentialService) GetCredentialByID(
	ctx context.Context,
	credentialID int64,
) (*store.Credential, error) {
	return s.credentialStore.ReadCredentialByID(ctx, credentialID)
}

func (s *CredentialService) ListCredentials(ctx context.Context) ([]*store.Credential, error) {
	credentials, err := s.credentialStore.ListCredentials(ctx)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return credentials, nil
}

func (s *CredentialService) UpdateCredential(
	ctx context.Context,
	credentialID int64,
	username, description string,
) error {
	return s.credentialStore.UpdateCredential(ctx, credentialID, username, description)
}

func (s *CredentialService) DeleteCredential(ctx context.Context, credentialID int64) error {
	return s.credentialStore.DeleteCredential(ctx, credentialID)
}
