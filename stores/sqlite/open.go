// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package sqlite

import (
	"context"
	"database/sql"
	"github.com/mdhender/ottoapp/domains"
	"github.com/mdhender/ottoapp/stores/sqlite/sqlc"
	"log"
	"os"
)

func (db *DB) Close() error {
	var err error
	if db != nil {
		if db.db != nil {
			err = db.db.Close()
			db.db = nil
		}
	}
	return err
}

// Open opens an existing store.
// Returns an error if the path is not a directory, or if the database does not exist.
// Caller must call Close() when done.
func Open(path string, ctx context.Context) (*DB, error) {
	// it is an error if the database does not already exist and is not a file.
	sb, err := os.Stat(path)
	if err != nil {
		log.Printf("[sqldb] %q: %s\n", path, err)
		return nil, err
	} else if sb.IsDir() || !sb.Mode().IsRegular() {
		log.Printf("[sqldb] %q: %s\n", path, err)
		return nil, domains.ErrInvalidPath
	}
	log.Printf("[sqldb] opening %s\n", path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// confirm that the database has foreign keys enabled
	checkPragma := "PRAGMA" + " foreign_keys = ON"
	if rslt, err := db.Exec(checkPragma); err != nil {
		_ = db.Close()
		log.Printf("[sqldb] error: foreign keys are disabled\n")
		return nil, domains.ErrForeignKeysDisabled
	} else if rslt == nil {
		_ = db.Close()
		log.Printf("[sqldb] error: foreign keys pragma failed\n")
		return nil, domains.ErrPragmaReturnedNil
	}

	// return the store.
	return &DB{path: path, db: db, ctx: ctx, q: sqlc.New(db)}, nil
}
