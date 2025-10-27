package store

import (
	"database/sql"
	"log"

	"github.com/haatos/simple-ci/internal/assets"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5"
)

func RunMigrations(db *sql.DB, dir string) {
	goose.SetBaseFS(assets.MigrationsFS)
	if err := goose.SetDialect("sqlite"); err != nil {
		log.Fatal(err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		log.Fatal(err)
	}
}
