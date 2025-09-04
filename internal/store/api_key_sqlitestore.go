package store

import (
	"context"
	"database/sql"

	"github.com/georgysavva/scany/v2/sqlscan"
)

func NewAPIKeySQLiteStore(rdb, rwdb *sql.DB) *APIKeySQLiteStore {
	return &APIKeySQLiteStore{rdb, rwdb}
}

type APIKeySQLiteStore struct {
	rdb, rwdb *sql.DB
}

func (store *APIKeySQLiteStore) CreateAPIKey(ctx context.Context, value string) (*APIKey, error) {
	key := new(APIKey)
	query := `insert into api_keys (value) values ($1) returning id`
	err := sqlscan.Get(ctx, store.rwdb, key, query, value)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (store *APIKeySQLiteStore) ReadAPIKeyByID(ctx context.Context, id int64) (*APIKey, error) {
	key := new(APIKey)
	query := `select * from api_keys where id = $1`
	err := sqlscan.Get(ctx, store.rdb, key, query, id)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (store *APIKeySQLiteStore) ReadAPIKeyByValue(
	ctx context.Context,
	value string,
) (*APIKey, error) {
	key := new(APIKey)
	query := `select * from api_keys where value = $1`
	err := sqlscan.Get(ctx, store.rdb, key, query, value)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (store *APIKeySQLiteStore) DeleteAPIKey(ctx context.Context, id int64) error {
	query := `delete from api_keys where id = $1`
	_, err := store.rwdb.ExecContext(ctx, query, id)
	return err
}
