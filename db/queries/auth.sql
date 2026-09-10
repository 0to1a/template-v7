-- name: GetActiveUserByEmail :one
SELECT public_uuid, email
FROM users
WHERE lower(email) = lower(sqlc.arg(email)) AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (email)
VALUES (sqlc.arg(email))
RETURNING public_uuid, email;
