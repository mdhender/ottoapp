// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package sqlite

// initialization functions

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"github.com/google/uuid"
	"github.com/mdhender/ottoapp/domains"
	"log"
	"os"
)

var (
	//go:embed sqlc/schema.sql
	schemaDDL string
)

// Create creates a new store.
// Returns an error if the database already exists.
func Create(path string, force bool, assets, components, userdata string, adminSecret, salt string, ctx context.Context) error {
	// all the data paths must exist and be folders
	for _, path := range []string{assets, components, userdata} {
		sb, err := os.Stat(path)
		if err != nil {
			log.Printf("[sqldb] %q: %s\n", path, err)
			return err
		}
		if !sb.IsDir() {
			log.Printf("[sqldb] %q: not a directory\n", path)
			return domains.ErrNotDirectory
		}
	}

	if adminSecret == "" {
		log.Printf("[sqldb] admin secret is required\n")
		return errors.New("missing admin secret")
	}

	if salt == "" {
		salt = uuid.NewString()
	}

	// if the stat fails because the file doesn't exist, we're okay.
	// if it fails for any other reason, it's an error.
	sb, err := os.Stat(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("[sqldb] %q: %s\n", path, err)
		return err
	}

	// it is an error if the path exists and is not a regular file.
	if sb != nil && (sb.IsDir() || !sb.Mode().IsRegular()) {
		log.Printf("[sqldb] %q: is a folder\n", path)
		return domains.ErrInvalidPath
	}

	// it is an error if the database already exists unless force is true.
	// in that case, we remove the database so that we can create it again.
	if sb != nil { // database file exists
		if !force {
			// we're not forcing the creation of a new database so this is an error
			return domains.ErrDatabaseExists
		}
		log.Printf("[sqldb] removing %s\n", path)
		if err := os.Remove(path); err != nil {
			return err
		}
	}

	// create the database
	log.Printf("store: creating %s\n", path)
	db, err := sql.Open("sqlite", path)
	if err != nil {
		log.Printf("[sqldb] %s\n", err)
		return err
	}
	defer db.Close()

	// confirm that the database has foreign keys enabled
	checkPragma := "PRAGMA" + " foreign_keys = ON"
	if rslt, err := db.Exec(checkPragma); err != nil {
		log.Printf("[sqldb] error: foreign keys are disabled\n")
		return domains.ErrForeignKeysDisabled
	} else if rslt == nil {
		log.Printf("[sqldb] error: foreign keys pragma failed\n")
		return domains.ErrPragmaReturnedNil
	}

	// create the schema
	if _, err := db.Exec(schemaDDL); err != nil {
		log.Printf("[sqldb] failed to initialize schema\n")
		log.Printf("[sqldb] %v\n", err)
		return errors.Join(domains.ErrCreateSchema, err)
	}

	log.Printf("store: updating server metadata\n")
	if _, err := db.Exec(`INSERT INTO server (assets_path, components_path, database_path, userdata_path, salt) VALUES (?1, ?2, ?3, ?4, lower(hex(randomblob(16))))`, assets, components, path, userdata); err != nil {
		log.Printf("[sqldb] %v\n", err)
		return err
	}

	log.Printf("store: creating the admin user\n")
	if hashedAdminSecret, err := HashPassword(adminSecret); err != nil {
		log.Printf("[sqldb] %v\n", err)
		return err
	} else if _, err = db.Exec(`INSERT INTO users (user_id, email, is_active, is_administrator, is_user, hashed_password, clan, last_login) VALUES (1, 'admin@ottomap', 1, 1, 0, ?1, '0000', 0)`, hashedAdminSecret); err != nil {
		log.Printf("[sqldb] %v\n", err)
		return err
	}

	log.Printf("store: created %s\n", path)

	return nil
}
