package store

import (
	"context"
	"database/sql"
	"errors"

	"github.com/georgysavva/scany/v2/sqlscan"
)

func NewAPIKeySQLiteStore(rdb, rwdb *sql.DB) *APIKeySQLiteStore {
	return &APIKeySQLiteStore{rdb, rwdb}
}

type APIKeySQLiteStore struct {
	rdb, rwdb *sql.DB
}

func (store *APIKeySQLiteStore) CreateAPIKey(ctx context.Context, value string) (*APIKey, error) {
	key := &APIKey{Value: value}
	query := `insert into api_keys (
		value
	)
	values ($1)
	returning id, created_on`
	err := sqlscan.Get(ctx, store.rwdb, key, query, key.Value)
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
	tx, err := store.rwdb.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	key := &APIKey{ID: id}
	if err := sqlscan.Get(ctx, tx, key, "select * from api_keys where id = $1", key.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "delete from api_keys where id = $1", key.ID); err != nil {
		return err
	}
	return tx.Commit()
}

func (store *APIKeySQLiteStore) ListAPIKeys(ctx context.Context) ([]*APIKey, error) {
	keys := make([]*APIKey, 0)
	query := `select * from api_keys`
	err := sqlscan.Select(ctx, store.rdb, &keys, query)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return keys, nil
}
