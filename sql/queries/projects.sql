-- name: GetProject :one
SELECT * FROM projects WHERE id = $1;

-- name: ListProjectsByUser :many
SELECT p.* FROM projects p
JOIN organizations o ON p.organization_id = o.id
JOIN organization_members om ON o.id = om.organization_id
WHERE om.user_id = $1
ORDER BY p.created_at DESC;

-- name: CreateProject :one
INSERT INTO projects (organization_id, name, slug, description, status)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateProject :one
UPDATE projects
SET name = COALESCE($2, name),
    description = COALESCE($3, description),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateProjectDatabase :one
UPDATE projects
SET db_name = $2,
    db_host = $3,
    db_port = $4,
    db_user = $5,
    db_password_encrypted = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteProject :exec
DELETE FROM projects WHERE id = $1;
