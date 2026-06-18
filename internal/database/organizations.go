package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidInput = errors.New("invalid input")

type Organization struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateOrganizationParams struct {
	Name string `json:"name"`
	Slug string `json:"slug"`
	Kind string `json:"kind"`
}

func (db *DB) ListOrganizations(ctx context.Context) ([]Organization, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, name, slug, kind, created_at, updated_at
		FROM organizations
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query organizations: %w", err)
	}
	defer rows.Close()

	organizations := []Organization{}
	for rows.Next() {
		var organization Organization
		if err := rows.Scan(
			&organization.ID,
			&organization.Name,
			&organization.Slug,
			&organization.Kind,
			&organization.CreatedAt,
			&organization.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan organization: %w", err)
		}
		organizations = append(organizations, organization)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organizations: %w", err)
	}

	return organizations, nil
}

func (db *DB) CreateOrganization(ctx context.Context, input CreateOrganizationParams) (Organization, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Kind = strings.TrimSpace(input.Kind)
	if input.Kind == "" {
		input.Kind = "school"
	}
	if input.Name == "" || input.Slug == "" {
		return Organization{}, fmt.Errorf("%w: name and slug are required", ErrInvalidInput)
	}

	var organization Organization
	if err := db.pool.QueryRow(ctx, `
		INSERT INTO organizations (name, slug, kind)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, kind, created_at, updated_at
	`, input.Name, input.Slug, input.Kind).Scan(
		&organization.ID,
		&organization.Name,
		&organization.Slug,
		&organization.Kind,
		&organization.CreatedAt,
		&organization.UpdatedAt,
	); err != nil {
		return Organization{}, fmt.Errorf("insert organization: %w", err)
	}

	return organization, nil
}
