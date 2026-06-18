package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidInput = errors.New("invalid input")

type Project struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Template  string    `json:"template"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateProjectParams struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Template string `json:"template"`
}

func (db *DB) ListProjects(ctx context.Context) ([]Project, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, name, slug, template, created_at, updated_at
		FROM projects
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query projects: %w", err)
	}
	defer rows.Close()

	projects := []Project{}
	for rows.Next() {
		var project Project
		if err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Slug,
			&project.Template,
			&project.CreatedAt,
			&project.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}

	return projects, nil
}

func (db *DB) CreateProject(ctx context.Context, input CreateProjectParams) (Project, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Slug = strings.TrimSpace(input.Slug)
	input.Template = strings.TrimSpace(input.Template)
	if input.Template == "" {
		input.Template = "blank"
	}
	if input.Name == "" || input.Slug == "" {
		return Project{}, fmt.Errorf("%w: name and slug are required", ErrInvalidInput)
	}

	var project Project
	if err := db.pool.QueryRow(ctx, `
		INSERT INTO projects (name, slug, template)
		VALUES ($1, $2, $3)
		RETURNING id, name, slug, template, created_at, updated_at
	`, input.Name, input.Slug, input.Template).Scan(
		&project.ID,
		&project.Name,
		&project.Slug,
		&project.Template,
		&project.CreatedAt,
		&project.UpdatedAt,
	); err != nil {
		return Project{}, fmt.Errorf("insert project: %w", err)
	}

	return project, nil
}
