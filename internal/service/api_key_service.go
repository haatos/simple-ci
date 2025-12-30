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

type APIKeyService struct {
	store         store.APIKeyStore
	uuidGenerator UUIDGenerator
}

func NewAPIKeyService(store store.APIKeyStore, uuidGenerator UUIDGenerator) *APIKeyService {
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
