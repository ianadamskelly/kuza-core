package database

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type ProjectAPIKey struct {
	ID              string     `json:"id"`
	ProjectID       string     `json:"project_id"`
	Name            string     `json:"name"`
	TokenPrefix     string     `json:"token_prefix"`
	CreatedByUserID *string    `json:"created_by_user_id,omitempty"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type CreatedProjectAPIKey struct {
	ProjectAPIKey
	Token string `json:"token"`
}

type CreateProjectAPIKeyParams struct {
	Name string `json:"name"`
}

func (db *DB) ListProjectAPIKeys(ctx context.Context, projectID string) ([]ProjectAPIKey, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, project_id, name, token_prefix, created_by_user_id, expires_at, created_at
		FROM project_api_keys
		WHERE project_id = $1
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("query project api keys: %w", err)
	}
	defer rows.Close()

	keys := []ProjectAPIKey{}
	for rows.Next() {
		var key ProjectAPIKey
		if err := rows.Scan(&key.ID, &key.ProjectID, &key.Name, &key.TokenPrefix, &key.CreatedByUserID, &key.ExpiresAt, &key.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan project api key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project api keys: %w", err)
	}

	return keys, nil
}

func (db *DB) CreateProjectAPIKey(ctx context.Context, projectID, createdByUserID string, input CreateProjectAPIKeyParams) (CreatedProjectAPIKey, error) {
	input.Name = strings.TrimSpace(input.Name)
	if projectID == "" || createdByUserID == "" || input.Name == "" {
		return CreatedProjectAPIKey{}, fmt.Errorf("%w: project id, creator user id, and name are required", ErrInvalidInput)
	}

	token, err := randomToken()
	if err != nil {
		return CreatedProjectAPIKey{}, fmt.Errorf("generate api key: %w", err)
	}
	prefix := token
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}

	var key ProjectAPIKey
	if err := db.pool.QueryRow(ctx, `
		INSERT INTO project_api_keys (project_id, name, token_hash, token_prefix, created_by_user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, project_id, name, token_prefix, created_by_user_id, expires_at, created_at
	`, projectID, input.Name, tokenHash(token), prefix, createdByUserID).Scan(
		&key.ID,
		&key.ProjectID,
		&key.Name,
		&key.TokenPrefix,
		&key.CreatedByUserID,
		&key.ExpiresAt,
		&key.CreatedAt,
	); err != nil {
		return CreatedProjectAPIKey{}, fmt.Errorf("insert project api key: %w", err)
	}

	return CreatedProjectAPIKey{ProjectAPIKey: key, Token: token}, nil
}

func (db *DB) AuthenticateProjectAPIKey(ctx context.Context, token string) (ProjectAPIKey, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return ProjectAPIKey{}, ErrUnauthorized
	}

	var key ProjectAPIKey
	if err := db.pool.QueryRow(ctx, `
		SELECT id, project_id, name, token_prefix, created_by_user_id, expires_at, created_at
		FROM project_api_keys
		WHERE token_hash = $1
		  AND (expires_at IS NULL OR expires_at > now())
	`, tokenHash(token)).Scan(
		&key.ID,
		&key.ProjectID,
		&key.Name,
		&key.TokenPrefix,
		&key.CreatedByUserID,
		&key.ExpiresAt,
		&key.CreatedAt,
	); err != nil {
		return ProjectAPIKey{}, ErrUnauthorized
	}

	return key, nil
}
