package assets

import "embed"

//go:embed migrations/*.sql
var MigrationsFS embed.FS

//go:embed public/**
var PublicFS embed.FS
