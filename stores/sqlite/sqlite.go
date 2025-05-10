// Copyright (c) 2024 Michael D Henderson. All rights reserved.

// Package sqlite implements the Sqlite database store.
package sqlite

//go:generate sqlc generate

import (
	"context"
	"database/sql"
	_ "embed"
	"github.com/mdhender/ottoapp/stores/sqlite/sqlc"
)

type DB struct {
	path string
	db   *sql.DB
	ctx  context.Context
	q    *sqlc.Queries
}
