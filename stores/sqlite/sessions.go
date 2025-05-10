// Copyright (c) 2024 Michael D Henderson. All rights reserved.

package sqlite

import (
	"database/sql"
	"errors"
	"github.com/google/uuid"
	"github.com/mdhender/ottoapp/domains"
	"github.com/mdhender/ottoapp/stores/sqlite/sqlc"
	"time"
)

func (db *DB) CreateSession(userId domains.ID, ttl time.Duration) (string, error) {
	err := db.q.DeleteUserSessions(db.ctx, int64(userId))
	if err != nil {
		return "", err
	}

	sessionId := uuid.NewString()
	err = db.q.CreateUserSession(db.ctx, sqlc.CreateUserSessionParams{
		SessID:    sessionId,
		UserID:    int64(userId),
		ExpiresAt: time.Now().Add(ttl).UTC().Unix(),
	})
	if err != nil {
		return "", err
	}

	return sessionId, nil
}

func (db *DB) DeleteUserSessions(userId domains.ID) error {
	return db.q.DeleteUserSessions(db.ctx, int64(userId))
}

func (db *DB) GetSession(id string) (*domains.User_t, error) {
	row, err := db.q.GetSession(db.ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	expiresAt := time.Unix(row.ExpiresAt, 0)
	//log.Printf("sessions: %s: expires at %v\n", id, expiresAt)
	//db.q.DeleteExpiredSessions(db.ctx, time.Now().UTC().Unix())
	if !time.Now().Before(expiresAt) {
		// session expired, should delete it
		return nil, db.q.DeleteExpiredSessions(db.ctx, time.Now().UTC().Unix())
	}

	return db.GetUser(domains.ID(row.UserID))
}
