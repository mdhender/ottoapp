--  Copyright (c) 2024 Michael D Henderson. All rights reserved.

-- GetServerPaths returns the paths to the server's assets, components, and user data directories.
--
-- name: GetServerPaths :one
SELECT assets_path, components_path, database_path, userdata_path
FROM server;

-- SetServerAssetsPath sets the path to the assets directory for the server.
--
-- name: SetServerAssetsPath :exec
UPDATE server
SET assets_path = :path;

-- SetServerComponentsPath sets the path to the components directory for the server.
--
-- name: SetServerComponentsPath :exec
UPDATE server
SET components_path = :path;

-- SetServerSalt sets the salt for the server.
--
-- name: SetServerSalt :exec
UPDATE server
SET salt = :salt;
