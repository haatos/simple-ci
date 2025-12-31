package store

import (
	"time"
)

type APIKey struct {
	ID        int64
	Value     string
	CreatedOn time.Time
}
