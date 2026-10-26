-- name: GetActiveUserByEmail :one
SELECT public_uuid, email
FROM users
WHERE lower(email) = lower(sqlc.arg(email)) AND deleted_at IS NULL;

-- name: CreateUser :one
INSERT INTO users (email)
VALUES (sqlc.arg(email))
RETURNING public_uuid, email;

-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
SELECT id, sqlc.arg(token_hash), sqlc.arg(expires_at)
FROM users
WHERE public_uuid = sqlc.arg(public_uuid) AND deleted_at IS NULL;

-- Atomic: of two concurrent calls with the same token, only one gets a row
-- name: RevokeRefreshToken :one
UPDATE refresh_tokens rt
SET revoked_at = now()
FROM users u
WHERE rt.token_hash = sqlc.arg(token_hash)
  AND rt.revoked_at IS NULL
  AND rt.expires_at > now()
  AND u.id = rt.user_id
  AND u.deleted_at IS NULL
RETURNING u.public_uuid;

-- Replay of an already-revoked token: assume theft, end every session of that user
-- name: RevokeRefreshTokensIfReused :exec
UPDATE refresh_tokens
SET revoked_at = now()
WHERE revoked_at IS NULL
  AND user_id = (
    SELECT reused.user_id FROM refresh_tokens reused
    WHERE reused.token_hash = sqlc.arg(token_hash) AND reused.revoked_at IS NOT NULL
  );
