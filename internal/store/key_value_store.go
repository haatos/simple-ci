package store

import (
	"database/sql"
	"log"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/haatos/simple-ci/internal"
)

var KVStore *KeyValueStore

const initQuery = `
	CREATE TABLE IF NOT EXISTS kvstore (
		email TEXT PRIMARY KEY,
		expires TIMESTAMP
	)
`

type KeyValueStore struct {
	DB *sql.DB
}

func NewKeyValueStore() *KeyValueStore {
	db, err := sql.Open("sqlite", "file:kvstore?mode=memory&cache=shared")
	if err != nil {
		log.Fatal(err)
	}
	return &KeyValueStore{DB: db}
}

func (kvs *KeyValueStore) ScheduleDailyCleanUp(s gocron.Scheduler) {
	if _, err := s.NewJob(gocron.DailyJob(1, gocron.NewAtTimes(gocron.NewAtTime(0, 0, 0))), gocron.NewTask(func() {
		if err := kvs.RemoveExpired(); err != nil {
			log.Println("err deleting expired keys from kvstore:", err)
		}
	})); err != nil {
		log.Fatal(err)
	}
}

func (kvs *KeyValueStore) Add(email string, expires time.Time) error {
	query := "insert into kvstore (email, expires) values($1, $2)"
	_, err := kvs.DB.Exec(query, email, expires.Format(internal.DBTimestampLayout))
	return err
}

func (kvs *KeyValueStore) Get(email string) (time.Time, error) {
	query := "select expires from kvstore where email = $1"
	var timestamp string
	err := kvs.DB.QueryRow(query, email).Scan(&timestamp)
	if err != nil {
		return time.Now(), err
	}
	output, err := time.Parse(internal.DBTimestampLayout, timestamp)
	return output, err
}

func (kvs *KeyValueStore) RemoveExpired() error {
	query := "delete from kvstore where expires < CURRENT_TIMESTAMP"
	_, err := kvs.DB.Exec(query)
	return err
}
