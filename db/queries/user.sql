-- name: GetUserByPublicUUID :one
SELECT public_uuid, email, display_name
FROM users
WHERE public_uuid = sqlc.arg(public_uuid) AND deleted_at IS NULL;

-- name: UpdateDisplayName :one
UPDATE users
SET display_name = sqlc.arg(display_name), updated_at = now()
WHERE public_uuid = sqlc.arg(public_uuid) AND deleted_at IS NULL
RETURNING public_uuid, email, display_name;
