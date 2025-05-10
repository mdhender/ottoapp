// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package main

import (
	"context"
	"fmt"
	"github.com/mdhender/ottoapp/stores/sqlite"
	"github.com/mdhender/phrases/v2"
	"github.com/spf13/cobra"
	"log"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	argsDb struct {
		force bool // if true, overwrite existing database
		paths struct {
			assets     string
			components string
			data       string
			database   string // path to the database file
		}
		secrets struct {
			useRandomSecret bool   // if true, generate a random secret for signing tokens
			admin           string // plain text password for admin user
			salt            string // salt for nothing (unused)
			signing         string // secret for signing tokens
		}
		data struct {
			user struct {
				clan      string // clan number
				email     string // email address for user
				secret    string // secret to use for user
				timezone  string // timezone for user
				usePhrase bool   // if true, use a phrase instead of a secret
				isActive  bool   // if true, force user to be active when resetting password
			}
		}
	}

	cmdDb = &cobra.Command{
		Use:   "db",
		Short: "Database management commands",
	}

	cmdDbInit = &cobra.Command{
		Use:   "init",
		Short: "Initialize the database",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if argsDb.paths.assets == "" {
				return fmt.Errorf("assets: path is required")
			} else if path, err := abspath(argsDb.paths.assets); err != nil {
				return fmt.Errorf("assets: %v\n", err)
			} else if ok, err := isdir(path); err != nil {
				return fmt.Errorf("assets: %v\n", err)
			} else if !ok {
				return fmt.Errorf("assets: %s: not a directory\n", path)
			} else {
				argsDb.paths.assets = path
			}

			if argsDb.paths.data == "" {
				return fmt.Errorf("data: path is required")
			} else if path, err := abspath(argsDb.paths.data); err != nil {
				return fmt.Errorf("data: %v\n", err)
			} else if ok, err := isdir(path); err != nil {
				return fmt.Errorf("data: %v\n", err)
			} else if !ok {
				return fmt.Errorf("data: %s: not a directory\n", path)
			} else {
				argsDb.paths.data = path
			}

			if argsDb.paths.database == "" {
				return fmt.Errorf("database: path is required\n")
			} else if path, err := filepath.Abs(argsDb.paths.database); err != nil {
				return fmt.Errorf("database: %v\n", err)
			} else {
				argsDb.paths.database = path
			}

			if argsDb.paths.components == "" {
				return fmt.Errorf("components: path is required")
			} else if path, err := abspath(argsDb.paths.components); err != nil {
				return fmt.Errorf("components: %v\n", err)
			} else if ok, err := isdir(path); err != nil {
				return fmt.Errorf("components: %v\n", err)
			} else if !ok {
				return fmt.Errorf("components: %s: not a directory\n", path)
			} else {
				argsDb.paths.components = path
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			log.Printf("db: init: database  %s\n", argsDb.paths.database)
			log.Printf("db: init: assets    %s\n", argsDb.paths.assets)
			log.Printf("db: init: data      %s\n", argsDb.paths.data)
			log.Printf("db: init: components %s\n", argsDb.paths.components)
			if argsDb.secrets.admin != "" {
				log.Printf("db: init: admin password %q\n", argsDb.secrets.admin)
			}

			// create the database
			log.Printf("db: init: creating database in %s\n", argsDb.paths.database)
			err := sqlite.Create(argsDb.paths.database, argsDb.force, argsDb.paths.assets, argsDb.paths.components, argsDb.paths.data, argsDb.secrets.admin, argsDb.secrets.salt, context.Background())
			if err != nil {
				log.Fatalf("db: init: %v\n", err)
			}

			log.Printf("db: created %q\n", argsDb.paths.database)
		},
	}

	cmdDbCreate = &cobra.Command{
		Use:   "create",
		Short: "Create data-base objects",
	}

	cmdDbCreateUser = &cobra.Command{
		Use:   "user",
		Short: "Create a new user",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if argsDb.paths.database == "" {
				return fmt.Errorf("database: path is required\n")
			} else if path, err := filepath.Abs(argsDb.paths.database); err != nil {
				return fmt.Errorf("database: %v\n", err)
			} else if ok, err := isfile(path); err != nil {
				log.Fatalf("database: %v\n", err)
			} else if !ok {
				log.Fatalf("database: %s: not a file\n", path)
			} else {
				argsDb.paths.database = path
			}

			if len(argsDb.data.user.clan) != 4 {
				log.Fatalf("db: create user: clan must be 4 digits between 1 and 999\n")
			} else if n, err := strconv.Atoi(argsDb.data.user.clan); err != nil {
				log.Fatalf("db: create user: clan must be 4 digits between 1 and 999\n")
			} else if n < 1 || n > 999 {
				log.Fatalf("db: create user: clan must be 4 digits between 1 and 999\n")
			}

			if argsDb.data.user.email != strings.TrimSpace(argsDb.data.user.email) {
				log.Fatalf("db: create user: email must not contain leading or trailing spaces\n")
			}

			if argsDb.data.user.usePhrase {
				if len(argsDb.data.user.secret) != 0 {
					log.Fatalf("db: create user: secret must be empty when using a phrase\n")
				}
				argsDb.data.user.secret = phrases.Generate(6)
			}

			if len(argsDb.data.user.secret) < 4 {
				log.Fatalf("db: create user: secret must be at least 4 characters\n")
			}

			if argsDb.data.user.timezone == "" {
				argsDb.data.user.timezone = "UTC"
			} else if argsDb.data.user.timezone != strings.TrimSpace(argsDb.data.user.timezone) {
				log.Fatalf("db: create user: timezone must not contain leading or trailing spaces\n")
			}

			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			// open the database
			log.Printf("db: create user: opening database %s\n", argsDb.paths.database)
			store, err := sqlite.Open(argsDb.paths.database, context.Background())
			if err != nil {
				log.Fatalf("db: create user: %v\n", err)
			}
			defer func() {
				_ = store.Close()
			}()

			log.Printf("db: create user: clan   %q\n", argsDb.data.user.clan)
			log.Printf("db: create user: email  %q\n", argsDb.data.user.email)
			log.Printf("db: create user: secret %q\n", argsDb.data.user.secret)

			// validate the timezone
			log.Printf("db: create user: tz     %q\n", argsDb.data.user.timezone)
			loc, err := time.LoadLocation(argsDb.data.user.timezone)
			if err != nil {
				log.Fatalf("db: create user: timezone: %v\n", err)
			}

			user, err := store.CreateUser(argsDb.data.user.email, argsDb.data.user.secret, argsDb.data.user.clan, loc)
			if err != nil {
				log.Fatalf("db: create user: %v\n", err)
			}
			log.Printf("db: create user: magic  %q\n", user.MagicLink)

			log.Printf("db: create user: user %d created\n", int(user.ID))
		},
	}

	cmdDbDelete = &cobra.Command{
		Use:   "delete",
		Short: "Delete data-base objects",
	}

	cmdDbDeleteUser = &cobra.Command{
		Use:   "user",
		Short: "Delete a user",
		PreRun: func(cmd *cobra.Command, args []string) {
			if argsDb.paths.database == "" {
				log.Fatal("database: path is required\n")
			} else if path, err := filepath.Abs(argsDb.paths.database); err != nil {
				log.Fatalf("database: %v\n", err)
			} else if ok, err := isfile(path); err != nil {
				log.Fatalf("database: %v\n", err)
			} else if !ok {
				log.Fatalf("database: %s: not a file\n", path)
			} else {
				argsDb.paths.database = path
			}

			if len(argsDb.data.user.clan) != 4 {
				log.Fatalf("clan: must be 4 digits between 1 and 999\n")
			} else if n, err := strconv.Atoi(argsDb.data.user.clan); err != nil {
				log.Fatalf("clan: must be 4 digits between 1 and 999\n")
			} else if n < 1 || n > 999 {
				log.Fatalf("clan: must be 4 digits between 1 and 999\n")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			// open the database
			log.Printf("db: delete user: opening database %s\n", argsDb.paths.database)
			store, err := sqlite.Open(argsDb.paths.database, context.Background())
			if err != nil {
				log.Fatalf("db: delete user: %v\n", err)
			}
			defer func() {
				_ = store.Close()
			}()

			log.Printf("db: delete user: clan   %q\n", argsDb.data.user.clan)

			if err := store.DeleteUserByClan(argsDb.data.user.clan); err != nil {
				log.Fatalf("db: delete user: %v\n", err)
			}

			log.Printf("db: delete user: user deleted\n")
		},
	}

	cmdDbUpdate = &cobra.Command{
		Use:   "update",
		Short: "Update database configuration",
	}

	cmdDbUpdateUser = &cobra.Command{
		Use:   "user",
		Short: "Update user in database",
	}

	cmdDbUpdateUserPassword = &cobra.Command{
		Use:   "password",
		Short: "Update user password in database",
		PreRun: func(cmd *cobra.Command, args []string) {
			if argsDb.paths.database == "" {
				log.Fatal("database: path is required\n")
			} else if path, err := filepath.Abs(argsDb.paths.database); err != nil {
				log.Fatalf("database: %v\n", err)
			} else if ok, err := isfile(path); err != nil {
				log.Fatalf("database: %v\n", err)
			} else if !ok {
				log.Fatalf("database: %s: not a file\n", path)
			} else {
				argsDb.paths.database = path
			}

			if len(argsDb.data.user.clan) != 4 {
				log.Fatalf("clan: must be 4 digits between 1 and 999\n")
			} else if n, err := strconv.Atoi(argsDb.data.user.clan); err != nil {
				log.Fatalf("clan: must be 4 digits between 1 and 999\n")
			} else if n < 1 || n > 999 {
				log.Fatalf("clan: must be 4 digits between 1 and 999\n")
			}

			argsDb.data.user.secret = phrases.Generate(6)
		},
		Run: func(cmd *cobra.Command, args []string) {
			// open the database
			log.Printf("db: update user: opening database %s\n", argsDb.paths.database)
			store, err := sqlite.Open(argsDb.paths.database, context.Background())
			if err != nil {
				log.Fatalf("db: update user: %v\n", err)
			}
			defer func() {
				_ = store.Close()
			}()

			id, err := store.GetUserByClan(argsDb.data.user.clan)
			if err != nil {
				log.Fatalf("db: update user: invalid clan: %v\n", err)
			}
			if err := store.UpdateUserPassword(id, argsDb.data.user.secret, argsDb.data.user.isActive); err != nil {
				log.Fatalf("db: update user: %v\n", err)
			}

			log.Printf("db: update user: clan %q: secret %q\n", argsDb.data.user.clan, argsDb.data.user.secret)
		},
	}

	cmdDbUpdateUserTimezone = &cobra.Command{
		Use:   "timezone",
		Short: "Update user timezone in database",
		PreRun: func(cmd *cobra.Command, args []string) {
			if argsDb.paths.database == "" {
				log.Fatal("database: path is required\n")
			} else if path, err := filepath.Abs(argsDb.paths.database); err != nil {
				log.Fatalf("database: %v\n", err)
			} else if ok, err := isfile(path); err != nil {
				log.Fatalf("database: %v\n", err)
			} else if !ok {
				log.Fatalf("database: %s: not a file\n", path)
			} else {
				argsDb.paths.database = path
			}

			if len(argsDb.data.user.clan) != 4 {
				log.Fatalf("clan: must be 4 digits between 1 and 999\n")
			} else if n, err := strconv.Atoi(argsDb.data.user.clan); err != nil {
				log.Fatalf("clan: must be 4 digits between 1 and 999\n")
			} else if n < 1 || n > 999 {
				log.Fatalf("clan: must be 4 digits between 1 and 999\n")
			}

			if argsDb.data.user.timezone = strings.TrimSpace(argsDb.data.user.timezone); argsDb.data.user.timezone == "" {
				log.Fatalf("timezone: required\n")
			}
		},
		Run: func(cmd *cobra.Command, args []string) {
			// validate the timezone
			log.Printf("db: update user: tz     %q\n", argsDb.data.user.timezone)
			loc, err := time.LoadLocation(argsDb.data.user.timezone)
			if err != nil {
				log.Fatalf("db: update user: timezone: %v\n", err)
			}

			// open the database
			log.Printf("db: update user: opening database %s\n", argsDb.paths.database)
			store, err := sqlite.Open(argsDb.paths.database, context.Background())
			if err != nil {
				log.Fatalf("db: update user: %v\n", err)
			}
			defer func() {
				_ = store.Close()
			}()

			id, err := store.GetUserByClan(argsDb.data.user.clan)
			if err != nil {
				log.Fatalf("db: update user: invalid clan: %v\n", err)
			}
			if err := store.UpdateUserTimezone(id, loc); err != nil {
				log.Fatalf("db: update user: %v\n", err)
			}

			log.Printf("db: update user: clan %q updated\n", argsDb.data.user.clan)
		},
	}
)
