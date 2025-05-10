--  Copyright (c) 2024 Michael D Henderson. All rights reserved.

-- GetSession returns the session with the given id.
-- Fails if the session does not exist or user is not active.
--
-- name: GetSession :one
SELECT user_id,
       expires_at
FROM sessions
WHERE sess_id = :session_id;

-- CreateUserSession creates a new session for the given user id.
--
-- name: CreateUserSession :exec
INSERT INTO sessions (sess_id, user_id, expires_at)
VALUES (:sess_id, :user_id, :expires_at);

-- DeleteExpiredSessions deletes all expired sessions.
--
-- name: DeleteExpiredSessions :exec
DELETE
FROM sessions
WHERE expires_at >= :dttm;

-- DeleteUserSessions deletes all sessions for the given user id.
--
-- name: DeleteUserSessions :exec
DELETE
FROM sessions
WHERE user_id = :user_id;