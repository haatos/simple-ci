package store

import (
	"context"
	"time"
)

type APIKey struct {
	ID        int64
	Value     string
	CreatedOn time.Time
}

type APIKeyStore interface {
	CreateAPIKey(context.Context, string) (*APIKey, error)
	ReadAPIKeyByID(context.Context, int64) (*APIKey, error)
	ReadAPIKeyByValue(context.Context, string) (*APIKey, error)
	DeleteAPIKey(context.Context, int64) error
}
