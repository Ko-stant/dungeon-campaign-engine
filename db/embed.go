// Package db embeds the goose SQL migrations so the server can apply them at
// startup without the migrations directory on disk.
package db

import "embed"

// Migrations holds migrations/*.sql.
//
//go:embed migrations/*.sql
var Migrations embed.FS
