-- name: UsersByNameForceIndex :many
SELECT id, name, email FROM users@{FORCE_INDEX=users_name_idx} WHERE name = sqlc.arg(name);

-- name: UsersByPrimary :one
SELECT * FROM users@users_pkey WHERE id = $1;

-- name: NoFullScan :many
SELECT id, name FROM users@{NO_FULL_SCAN} WHERE name = $1;

-- name: FollowerRead :many
SELECT id, name, email FROM users AS OF SYSTEM TIME follower_read_timestamp() WHERE name = sqlc.arg(name);

-- name: FollowerReadInterval :many
SELECT * FROM users AS OF SYSTEM TIME '-10s';
