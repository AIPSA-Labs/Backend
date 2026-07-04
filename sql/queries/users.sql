-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (email, password_hash, full_name, provider, provider_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET full_name = COALESCE($2, full_name),
    avatar_url = COALESCE($3, avatar_url),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
