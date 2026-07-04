-- name: CreateSQLHistory :one
INSERT INTO sql_history (project_id, user_id, query, duration_ms, rows_affected, error_message, is_read_only)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: ListSQLHistoryByProject :many
SELECT * FROM sql_history
WHERE project_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSQLHistoryByProject :one
SELECT COUNT(*) FROM sql_history WHERE project_id = $1;
