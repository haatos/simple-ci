package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/haatos/simple-ci/internal/store"
)

type UUIDGenerator interface {
	GenerateUUID() string
}

func NewUUIDGen() *UUIDGen {
	return &UUIDGen{}
}

type UUIDGen struct{}

func (ug *UUIDGen) GenerateUUID() string {
	return uuid.NewString()
}

type APIKeyWriter interface {
	CreateAPIKey(ctx context.Context, uuid string) (*store.APIKey, error)
	DeleteAPIKey(ctx context.Context, id int64) error
}

type APIKeyReader interface {
	ReadAPIKeyByID(ctx context.Context, id int64) (*store.APIKey, error)
	ReadAPIKeyByValue(ctx context.Context, uuid string) (*store.APIKey, error)
	ListAPIKeys(ctx context.Context) ([]*store.APIKey, error)
}

type APIKeyStore interface {
	APIKeyWriter
	APIKeyReader
}

type APIKeyService struct {
	store         APIKeyStore
	uuidGenerator UUIDGenerator
}

func NewAPIKeyService(store APIKeyStore, uuidGenerator UUIDGenerator) *APIKeyService {
	return &APIKeyService{store, uuidGenerator}
}

func (s *APIKeyService) CreateAPIKey(ctx context.Context) (*store.APIKey, error) {
	value := s.uuidGenerator.GenerateUUID()
	return s.store.CreateAPIKey(ctx, value)
}

func (s *APIKeyService) GetAPIKeyByID(ctx context.Context, id int64) (*store.APIKey, error) {
	return s.store.ReadAPIKeyByID(ctx, id)
}

func (s *APIKeyService) GetAPIKeyByValue(ctx context.Context, value string) (*store.APIKey, error) {
	return s.store.ReadAPIKeyByValue(ctx, value)
}

func (s *APIKeyService) DeleteAPIKey(ctx context.Context, id int64) error {
	return s.store.DeleteAPIKey(ctx, id)
}

func (s *APIKeyService) ListAPIKeys(ctx context.Context) ([]*store.APIKey, error) {
	return s.store.ListAPIKeys(ctx)
}
