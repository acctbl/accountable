package database

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func Migrations() fs.FS {
	files, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		panic("compiled migrations are unavailable")
	}
	return files
}
