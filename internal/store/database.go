package store

import (
	"database/sql"
	"log"
	"runtime"

	"github.com/haatos/simple-ci/internal/settings"
)

func InitDatabase(readonly bool) *sql.DB {
	db, err := sql.Open("sqlite", settings.Settings.SQLiteDbString(readonly))
	if err != nil {
		log.Fatal("fatal error opening sqlite database:", err)
	}

	if readonly {
		db.SetMaxOpenConns(max(4, runtime.NumCPU()))
	} else {
		if _, err := db.Exec("PRAGMA temp_store=memory"); err != nil {
			log.Fatal(err)
		}
		if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
			log.Fatal(err)
		}
		db.SetMaxOpenConns(1)
	}

	return db
}
