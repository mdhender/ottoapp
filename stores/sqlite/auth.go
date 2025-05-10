// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package sqlite

import (
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/mdhender/ottoweb/domains"
	"github.com/mdhender/ottoweb/stores/sqlite/sqlc"
	"golang.org/x/crypto/bcrypt"
	"log"
	_ "modernc.org/sqlite"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// This file should implement the store for the authentication domain.
// Maybe someday I will understand how to do this.

// important: any function that returns the user should return GetUser because it sets paths!

// UpdateAdministrator updates the administrator's password.
// Like all functions, it assumes that the administrator has user_id of 1.
func (db *DB) UpdateAdministrator(plainTextSecret string, isActive bool) error {
	var err error
	var row sqlc.GetUserHashedPasswordRow
	if plainTextSecret == "" {
		row, err = db.q.GetUserHashedPassword(db.ctx, 1)
	} else {
		row.HashedPassword, err = HashPassword(plainTextSecret)
		if err != nil {
			return err
		}
	}
	log.Printf("db: auth: updateAdministrator: password %q: hashed %q\n", plainTextSecret, row.HashedPassword)
	parms := sqlc.UpdateUserPasswordParams{
		UserID:         1,
		HashedPassword: row.HashedPassword,
	}
	if isActive {
		parms.IsActive = 1
	}
	return db.q.UpdateUserPassword(db.ctx, parms)
}

func (db *DB) CreateUser(email, plainTextSecret, clan string, timezone *time.Location) (*domains.User_t, error) {
	if strings.TrimSpace(email) != email {
		return nil, domains.ErrInvalidEmail
	}
	email = strings.ToLower(email)
	if clanNo, err := strconv.Atoi(clan); err != nil || clanNo < 1 || clanNo > 999 {
		return nil, domains.ErrInvalidClan
	}

	// hash the password. can fail if the password is too long.
	hashedPassword, err := HashPassword(plainTextSecret)
	if err != nil {
		return nil, err
	}

	magicLink := uuid.NewString()

	// lookup the timezone. not sure that can fail, but if it does, default to UTC.
	var tz string
	if timezone != nil {
		tz = timezone.String()
	}
	if tz == "" {
		tz = "UTC"
	}

	//tx, err := db.db.BeginTx(db.ctx, nil)
	//if err != nil {
	//	return 0, err
	//}
	//defer func() {
	//	_ = tx.Rollback()
	//}()
	//qtx := db.q.WithTx(tx)

	// note: we let LastLogin be the zero-value for time.Time, which means never logged in.
	id, err := db.q.CreateUser(db.ctx, sqlc.CreateUserParams{
		Email:          email,
		HashedPassword: hashedPassword,
		MagicLink:      magicLink,
		IsActive:       1,
		Clan:           clan,
		Timezone:       tz,
	})
	if err != nil {
		return nil, err
	}

	//err = tx.Commit()
	//if err != nil {
	//	return 0, err
	//}

	return db.GetUser(domains.ID(id))
}

func (db *DB) DeleteUserByClan(clan string) error {
	if clanNo, err := strconv.Atoi(clan); err != nil || clanNo < 1 || clanNo > 999 {
		return domains.ErrInvalidClan
	}

	log.Printf("db: auth: deleteUserByClan: clan %q\n", clan)

	return db.q.DeleteUserByClan(db.ctx, clan)
}

func (db *DB) AuthenticateUser(email, plainTextPassword string) (*domains.User_t, error) {
	row, err := db.q.GetUserByEmail(db.ctx, email)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		return nil, domains.ErrUnauthorized
	} else if !CheckPassword(plainTextPassword, row.HashedPassword) {
		return nil, domains.ErrUnauthorized
	}

	// update the last login time, ignoring any errors
	_ = db.q.UpdateUserLastLogin(db.ctx, sqlc.UpdateUserLastLoginParams{
		UserID:    row.UserID,
		LastLogin: time.Now().UTC().Unix(),
	})

	return db.GetUser(domains.ID(row.UserID))
}

// GetUser returns the user with the given ID.
// If the user does not exist, it returns an error.
//
// important change: any function that returns *domains.User_t must set the Data path!
func (db *DB) GetUser(userID domains.ID) (*domains.User_t, error) {
	row, err := db.q.GetUser(db.ctx, int64(userID))
	if err != nil {
		return nil, err
	}
	// convert row.Timezone to a time.Location
	loc, err := time.LoadLocation(row.Timezone)
	if err != nil {
		return nil, err
	}

	paths, err := db.q.GetServerPaths(db.ctx)
	if err != nil {
		return nil, err
	} else if paths.UserdataPath == "" {
		return nil, domains.ErrMissingUserdataPath
	}
	userDataPath := filepath.Join(paths.UserdataPath, row.Clan, "data")

	user := &domains.User_t{
		ID:    userID,
		Email: row.Email,
		Clan:  row.Clan,
		Roles: struct {
			IsActive        bool
			IsAdministrator bool
			IsAuthenticated bool
			IsOperator      bool
			IsUser          bool
		}{
			IsActive:        row.IsActive == 1,
			IsAdministrator: row.IsAdministrator == 1,
			IsOperator:      row.IsOperator == 1,
			IsUser:          row.IsUser == 1,
		},
		Data:      userDataPath,
		Created:   row.CreatedAt,
		Updated:   row.UpdatedAt,
		LastLogin: time.Unix(row.LastLogin, 0),
	}
	user.LanguageAndDates.DateFormat = "2006-01-02"
	user.LanguageAndDates.Timezone.Location = loc

	return user, nil
}

func (db *DB) GetUserByClan(clan string) (domains.ID, error) {
	id, err := db.q.GetUserByClan(db.ctx, clan)
	return domains.ID(id), err
}

func (db *DB) UpdateUserPassword(userID domains.ID, plainTextSecret string, forceActive bool) error {
	// hash the password. can fail if the password is too long.
	hashedPassword, err := HashPassword(plainTextSecret)
	if err != nil {
		return err
	}

	var isActive int64
	if forceActive {
		isActive = 1
	}

	return db.q.UpdateUserPassword(db.ctx, sqlc.UpdateUserPasswordParams{
		UserID:         int64(userID),
		HashedPassword: hashedPassword,
		IsActive:       isActive,
	})
}

func (db *DB) UpdateUserTimezone(userID domains.ID, timezone *time.Location) error {
	// lookup the timezone. not sure that can fail, but if it does, return an error.
	if timezone == nil {
		return domains.ErrInvalidTimezone
	}
	tz := timezone.String()
	if tz == "" {
		return domains.ErrInvalidTimezone
	}

	return db.q.UpdateUserTimezone(db.ctx, sqlc.UpdateUserTimezoneParams{
		UserID:   int64(userID),
		Timezone: tz,
	})
}

// simple password functions inspired by https://www.gregorygaines.com/blog/how-to-properly-hash-and-salt-passwords-in-golang-bcrypt/

// CheckPassword returns true if the plain text password matches the hashed password.
func CheckPassword(plainTextPassword, hashedPassword string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainTextPassword)) == nil
}

// HashPassword uses the cheapest bcrypt cost to hash the password because we are not going to use
// it for anything other than authentication in non-production environments.
func HashPassword(plainTextPassword string) (string, error) {
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(plainTextPassword), bcrypt.MinCost)
	if err != nil {
		return "", err
	}
	return string(hashedPasswordBytes), err
}
