package service

import (
	"context"

	"github.com/haatos/simple-ci/internal/store"
)

type APIKeyServicer interface {
	CreateAPIKey(context.Context, string) (*store.APIKey, error)
	GetAPIKeyByID(context.Context, int64) (*store.APIKey, error)
	GetAPIKeyByValue(context.Context, string) (*store.APIKey, error)
	DeleteAPIKey(context.Context, int64) error
}

func NewAPIKeyService(s store.APIKeyStore) *APIKeyService {
	return &APIKeyService{s}
}

type APIKeyService struct {
	store store.APIKeyStore
}

func (s *APIKeyService) CreateAPIKey(ctx context.Context, value string) (*store.APIKey, error) {
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
