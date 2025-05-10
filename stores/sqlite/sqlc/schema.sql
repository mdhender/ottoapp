--  Copyright (c) 2024 Michael D Henderson. All rights reserved.

-- foreign keys must be disabled to drop tables with foreign keys
PRAGMA foreign_keys = OFF;

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS server;
DROP TABLE IF EXISTS users;

-- foreign keys must be enabled with every database connection
PRAGMA foreign_keys = ON;

CREATE TABLE users
(
    -- columns for identification
    user_id          INTEGER PRIMARY KEY AUTOINCREMENT,
    email            TEXT UNIQUE NOT NULL,
    timezone         TEXT        NOT NULL DEFAULT 'UTC',

    -- columns for authentication
    is_active        INTEGER     NOT NULL DEFAULT 0,
    hashed_password  TEXT        NOT NULL,
    magic_link       TEXT UNIQUE NOT NULL DEFAULT '',

    -- columns for authorization
    clan             TEXT UNIQUE NOT NULL,
    is_administrator INTEGER     NOT NULL DEFAULT 0,
    is_operator      INTEGER     NOT NULL DEFAULT 0,
    is_user          INTEGER     NOT NULL DEFAULT 1,

    last_login       INTEGER     NOT NULL,

    -- columns for auditing
    created_at       TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE sessions
(
    sess_id    TEXT           NOT NULL,
    user_id    INTEGER UNIQUE NOT NULL,
    expires_at INTEGER        NOT NULL,

    -- columns for auditing
    created_at TIMESTAMP      NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (sess_id),
    FOREIGN KEY (user_id) REFERENCES users (user_id) ON DELETE CASCADE
);

CREATE TABLE server
(
    assets_path     TEXT NOT NULL,
    components_path TEXT NOT NULL,
    database_path   TEXT NOT NULL,
    userdata_path   TEXT NOT NULL,
    salt            TEXT NOT NULL
);
