package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash *string    `json:"password_hash,omitempty"`
	FullName     string     `json:"full_name"`
	AvatarUrl    *string    `json:"avatar_url,omitempty"`
	Provider     string     `json:"provider"`
	ProviderId   *string    `json:"provider_id,omitempty"`
	EmailVerified bool      `json:"email_verified"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type Organization struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	OwnerID   uuid.UUID `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Project struct {
	ID                   uuid.UUID `json:"id"`
	OrganizationID       uuid.UUID `json:"organization_id"`
	Name                 string    `json:"name"`
	Slug                 string    `json:"slug"`
	Description          *string   `json:"description,omitempty"`
	Status               string    `json:"status"`
	DbName               string    `json:"db_name"`
	DbHost               string    `json:"db_host"`
	DbPort               int32     `json:"db_port"`
	DbUser               string    `json:"db_user"`
	DbPasswordEncrypted  string    `json:"db_password_encrypted"`
	DbSslMode            *string   `json:"db_ssl_mode,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type ApiKey struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uuid.UUID       `json:"user_id"`
	Name        string          `json:"name"`
	KeyHash     string          `json:"key_hash"`
	KeyPrefix   string          `json:"key_prefix"`
	Permissions json.RawMessage `json:"permissions"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"`
	LastUsedAt  *time.Time      `json:"last_used_at,omitempty"`
	IsActive    bool            `json:"is_active"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Session struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	RefreshTokenHash string    `json:"refresh_token_hash"`
	UserAgent        *string   `json:"user_agent,omitempty"`
	IpAddress        *string   `json:"ip_address,omitempty"`
	ExpiresAt        time.Time `json:"expires_at"`
	CreatedAt        time.Time `json:"created_at"`
}

type SqlHistory struct {
	ID           uuid.UUID  `json:"id"`
	ProjectID    uuid.UUID  `json:"project_id"`
	UserID       uuid.UUID  `json:"user_id"`
	Query        string     `json:"query"`
	DurationMs   *int32     `json:"duration_ms,omitempty"`
	RowsAffected *int64     `json:"rows_affected,omitempty"`
	ErrorMessage *string    `json:"error_message,omitempty"`
	IsReadOnly   bool       `json:"is_read_only"`
	CreatedAt    time.Time  `json:"created_at"`
}

type Queries struct {
	conn *pgxpool.Pool
}

func New(conn *pgxpool.Pool) *Queries {
	return &Queries{conn: conn}
}

func (q *Queries) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	var user User
	err := q.conn.QueryRow(ctx, `SELECT id, email, password_hash, full_name, avatar_url, provider, provider_id, email_verified, is_active, created_at, updated_at FROM users WHERE id = $1`, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.AvatarUrl, &user.Provider, &user.ProviderId, &user.EmailVerified, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	return user, err
}

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var user User
	err := q.conn.QueryRow(ctx, `SELECT id, email, password_hash, full_name, avatar_url, provider, provider_id, email_verified, is_active, created_at, updated_at FROM users WHERE email = $1`, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.AvatarUrl, &user.Provider, &user.ProviderId, &user.EmailVerified, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	return user, err
}

type CreateUserParams struct {
	Email        string
	PasswordHash *string
	FullName     string
	Provider     string
	ProviderId   *string
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	var user User
	err := q.conn.QueryRow(ctx, `INSERT INTO users (email, password_hash, full_name, provider, provider_id) VALUES ($1, $2, $3, $4, $5) RETURNING id, email, password_hash, full_name, avatar_url, provider, provider_id, email_verified, is_active, created_at, updated_at`,
		arg.Email, arg.PasswordHash, arg.FullName, arg.Provider, arg.ProviderId,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.AvatarUrl, &user.Provider, &user.ProviderId, &user.EmailVerified, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	return user, err
}

type UpdateUserParams struct {
	ID        uuid.UUID
	FullName  *string
	AvatarUrl *string
}

func (q *Queries) UpdateUser(ctx context.Context, arg UpdateUserParams) (User, error) {
	var user User
	err := q.conn.QueryRow(ctx, `UPDATE users SET full_name = COALESCE($2, full_name), avatar_url = COALESCE($3, avatar_url), updated_at = NOW() WHERE id = $1 RETURNING id, email, password_hash, full_name, avatar_url, provider, provider_id, email_verified, is_active, created_at, updated_at`,
		arg.ID, arg.FullName, arg.AvatarUrl,
	).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.AvatarUrl, &user.Provider, &user.ProviderId, &user.EmailVerified, &user.IsActive, &user.CreatedAt, &user.UpdatedAt,
	)
	return user, err
}

type CreateOrganizationParams struct {
	Name    string
	Slug    string
	OwnerID uuid.UUID
}

func (q *Queries) CreateOrganization(ctx context.Context, arg CreateOrganizationParams) (Organization, error) {
	var org Organization
	err := q.conn.QueryRow(ctx, `INSERT INTO organizations (name, slug, owner_id) VALUES ($1, $2, $3) RETURNING id, name, slug, owner_id, created_at, updated_at`,
		arg.Name, arg.Slug, arg.OwnerID,
	).Scan(&org.ID, &org.Name, &org.Slug, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt)
	return org, err
}

type AddOrganizationMemberParams struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Role           string
}

func (q *Queries) AddOrganizationMember(ctx context.Context, arg AddOrganizationMemberParams) error {
	_, err := q.conn.Exec(ctx, `INSERT INTO organization_members (organization_id, user_id, role) VALUES ($1, $2, $3)`,
		arg.OrganizationID, arg.UserID, arg.Role)
	return err
}

func (q *Queries) GetOrganizationByUser(ctx context.Context, userID uuid.UUID) (Organization, error) {
	var org Organization
	err := q.conn.QueryRow(ctx, `SELECT o.id, o.name, o.slug, o.owner_id, o.created_at, o.updated_at FROM organizations o JOIN organization_members om ON o.id = om.organization_id WHERE om.user_id = $1 LIMIT 1`, userID).Scan(
		&org.ID, &org.Name, &org.Slug, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt,
	)
	return org, err
}

func (q *Queries) ListProjectsByUser(ctx context.Context, userID uuid.UUID) ([]Project, error) {
	rows, err := q.conn.Query(ctx, `SELECT p.id, p.organization_id, p.name, p.slug, p.description, p.status, p.db_name, p.db_host, p.db_port, p.db_user, p.db_password_encrypted, p.db_ssl_mode, p.created_at, p.updated_at FROM projects p JOIN organizations o ON p.organization_id = o.id JOIN organization_members om ON o.id = om.organization_id WHERE om.user_id = $1 ORDER BY p.created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		err := rows.Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.DbName, &p.DbHost, &p.DbPort, &p.DbUser, &p.DbPasswordEncrypted, &p.DbSslMode, &p.CreatedAt, &p.UpdatedAt)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (q *Queries) GetProject(ctx context.Context, id uuid.UUID) (Project, error) {
	var p Project
	err := q.conn.QueryRow(ctx, `SELECT id, organization_id, name, slug, description, status, db_name, db_host, db_port, db_user, db_password_encrypted, db_ssl_mode, created_at, updated_at FROM projects WHERE id = $1`, id).Scan(
		&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.DbName, &p.DbHost, &p.DbPort, &p.DbUser, &p.DbPasswordEncrypted, &p.DbSslMode, &p.CreatedAt, &p.UpdatedAt,
	)
	return p, err
}

type CreateProjectParams struct {
	OrganizationID uuid.UUID
	Name           string
	Slug           string
	Description    *string
	Status         string
}

func (q *Queries) CreateProject(ctx context.Context, arg CreateProjectParams) (Project, error) {
	var p Project
	err := q.conn.QueryRow(ctx, `INSERT INTO projects (organization_id, name, slug, description, status) VALUES ($1, $2, $3, $4, $5) RETURNING id, organization_id, name, slug, description, status, db_name, db_host, db_port, db_user, db_password_encrypted, db_ssl_mode, created_at, updated_at`,
		arg.OrganizationID, arg.Name, arg.Slug, arg.Description, arg.Status,
	).Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.DbName, &p.DbHost, &p.DbPort, &p.DbUser, &p.DbPasswordEncrypted, &p.DbSslMode, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

type UpdateProjectParams struct {
	ID          uuid.UUID
	Name        *string
	Description *string
}

func (q *Queries) UpdateProject(ctx context.Context, arg UpdateProjectParams) (Project, error) {
	var p Project
	err := q.conn.QueryRow(ctx, `UPDATE projects SET name = COALESCE($2, name), description = COALESCE($3, description), updated_at = NOW() WHERE id = $1 RETURNING id, organization_id, name, slug, description, status, db_name, db_host, db_port, db_user, db_password_encrypted, db_ssl_mode, created_at, updated_at`,
		arg.ID, arg.Name, arg.Description,
	).Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.DbName, &p.DbHost, &p.DbPort, &p.DbUser, &p.DbPasswordEncrypted, &p.DbSslMode, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

type UpdateProjectDatabaseParams struct {
	ID                  uuid.UUID
	DbName              string
	DbHost              string
	DbPort              int32
	DbUser              string
	DbPasswordEncrypted string
}

func (q *Queries) UpdateProjectDatabase(ctx context.Context, arg UpdateProjectDatabaseParams) (Project, error) {
	var p Project
	err := q.conn.QueryRow(ctx, `UPDATE projects SET db_name = $2, db_host = $3, db_port = $4, db_user = $5, db_password_encrypted = $6, updated_at = NOW() WHERE id = $1 RETURNING id, organization_id, name, slug, description, status, db_name, db_host, db_port, db_user, db_password_encrypted, db_ssl_mode, created_at, updated_at`,
		arg.ID, arg.DbName, arg.DbHost, arg.DbPort, arg.DbUser, arg.DbPasswordEncrypted,
	).Scan(&p.ID, &p.OrganizationID, &p.Name, &p.Slug, &p.Description, &p.Status, &p.DbName, &p.DbHost, &p.DbPort, &p.DbUser, &p.DbPasswordEncrypted, &p.DbSslMode, &p.CreatedAt, &p.UpdatedAt)
	return p, err
}

func (q *Queries) DeleteProject(ctx context.Context, id uuid.UUID) error {
	_, err := q.conn.Exec(ctx, `DELETE FROM projects WHERE id = $1`, id)
	return err
}

type CreateSessionParams struct {
	UserID           uuid.UUID
	RefreshTokenHash string
	UserAgent        *string
	IpAddress        *string
	ExpiresAt        time.Time
}

func (q *Queries) CreateSession(ctx context.Context, arg CreateSessionParams) (Session, error) {
	var s Session
	err := q.conn.QueryRow(ctx, `INSERT INTO sessions (user_id, refresh_token_hash, user_agent, ip_address, expires_at) VALUES ($1, $2, $3, $4, $5) RETURNING id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at`,
		arg.UserID, arg.RefreshTokenHash, arg.UserAgent, arg.IpAddress, arg.ExpiresAt,
	).Scan(&s.ID, &s.UserID, &s.RefreshTokenHash, &s.UserAgent, &s.IpAddress, &s.ExpiresAt, &s.CreatedAt)
	return s, err
}

func (q *Queries) GetSessionByToken(ctx context.Context, tokenHash string) (Session, error) {
	var s Session
	err := q.conn.QueryRow(ctx, `SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at FROM sessions WHERE refresh_token_hash = $1`, tokenHash).Scan(
		&s.ID, &s.UserID, &s.RefreshTokenHash, &s.UserAgent, &s.IpAddress, &s.ExpiresAt, &s.CreatedAt,
	)
	return s, err
}

func (q *Queries) DeleteSession(ctx context.Context, id uuid.UUID) error {
	_, err := q.conn.Exec(ctx, `DELETE FROM sessions WHERE id = $1`, id)
	return err
}

type CreateAPIKeyParams struct {
	UserID      uuid.UUID
	Name        string
	KeyHash     string
	KeyPrefix   string
	Permissions []string
	ExpiresAt   *time.Time
}

func (q *Queries) CreateAPIKey(ctx context.Context, arg CreateAPIKeyParams) (ApiKey, error) {
	var k ApiKey
	err := q.conn.QueryRow(ctx, `INSERT INTO api_keys (user_id, name, key_hash, key_prefix, permissions, expires_at) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, user_id, name, key_hash, key_prefix, permissions, expires_at, last_used_at, is_active, created_at`,
		arg.UserID, arg.Name, arg.KeyHash, arg.KeyPrefix, arg.Permissions, arg.ExpiresAt,
	).Scan(&k.ID, &k.UserID, &k.Name, &k.KeyHash, &k.KeyPrefix, &k.Permissions, &k.ExpiresAt, &k.LastUsedAt, &k.IsActive, &k.CreatedAt)
	return k, err
}

func (q *Queries) ListAPIKeysByUser(ctx context.Context, userID uuid.UUID) ([]ApiKey, error) {
	rows, err := q.conn.Query(ctx, `SELECT id, name, key_prefix, permissions, expires_at, last_used_at, is_active, created_at FROM api_keys WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []ApiKey
	for rows.Next() {
		var k ApiKey
		err := rows.Scan(&k.ID, &k.Name, &k.KeyPrefix, &k.Permissions, &k.ExpiresAt, &k.LastUsedAt, &k.IsActive, &k.CreatedAt)
		if err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

type DeleteAPIKeyParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func (q *Queries) DeleteAPIKey(ctx context.Context, arg DeleteAPIKeyParams) error {
	_, err := q.conn.Exec(ctx, `DELETE FROM api_keys WHERE id = $1 AND user_id = $2`, arg.ID, arg.UserID)
	return err
}

type CreateSQLHistoryParams struct {
	ProjectID    uuid.UUID
	UserID       uuid.UUID
	Query        string
	DurationMs   *int32
	RowsAffected *int64
	ErrorMessage *string
	IsReadOnly   bool
}

func (q *Queries) CreateSQLHistory(ctx context.Context, arg CreateSQLHistoryParams) (SqlHistory, error) {
	var h SqlHistory
	err := q.conn.QueryRow(ctx, `INSERT INTO sql_history (project_id, user_id, query, duration_ms, rows_affected, error_message, is_read_only) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, project_id, user_id, query, duration_ms, rows_affected, error_message, is_read_only, created_at`,
		arg.ProjectID, arg.UserID, arg.Query, arg.DurationMs, arg.RowsAffected, arg.ErrorMessage, arg.IsReadOnly,
	).Scan(&h.ID, &h.ProjectID, &h.UserID, &h.Query, &h.DurationMs, &h.RowsAffected, &h.ErrorMessage, &h.IsReadOnly, &h.CreatedAt)
	return h, err
}

type ListSQLHistoryByProjectParams struct {
	ProjectID uuid.UUID
	Limit     int32
	Offset    int32
}

func (q *Queries) ListSQLHistoryByProject(ctx context.Context, arg ListSQLHistoryByProjectParams) ([]SqlHistory, error) {
	rows, err := q.conn.Query(ctx, `SELECT id, project_id, user_id, query, duration_ms, rows_affected, error_message, is_read_only, created_at FROM sql_history WHERE project_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		arg.ProjectID, arg.Limit, arg.Offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []SqlHistory
	for rows.Next() {
		var h SqlHistory
		err := rows.Scan(&h.ID, &h.ProjectID, &h.UserID, &h.Query, &h.DurationMs, &h.RowsAffected, &h.ErrorMessage, &h.IsReadOnly, &h.CreatedAt)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}
