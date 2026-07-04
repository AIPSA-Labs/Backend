-- name: CreateAPIKey :one
INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions, expires_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListAPIKeysByUser :many
SELECT id, name, key_prefix, permissions, expires_at, last_used_at, is_active, created_at
FROM api_keys
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetAPIKeyByHash :one
SELECT * FROM api_keys WHERE key_hash = $1 AND is_active = true;

-- name: UpdateAPIKeyLastUsed :exec
UPDATE api_keys SET last_used_at = NOW() WHERE id = $1;

-- name: DeleteAPIKey :exec
DELETE FROM api_keys WHERE id = $1 AND user_id = $2;
