package database

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"kuza-core/internal/config"
)

func (db *DB) BootstrapOwner(ctx context.Context, cfg config.BootstrapConfig) error {
	if cfg.OrganizationName == "" && cfg.OwnerEmail == "" && cfg.OwnerPassword == "" {
		return nil
	}
	if cfg.OrganizationName == "" || cfg.OrganizationSlug == "" || cfg.OwnerEmail == "" || cfg.OwnerPassword == "" {
		return fmt.Errorf("bootstrap requires organization name, organization slug, owner email, and owner password")
	}

	email := strings.ToLower(strings.TrimSpace(cfg.OwnerEmail))
	if email == "" {
		return fmt.Errorf("bootstrap owner email is empty after normalization")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.OwnerPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash bootstrap owner password: %w", err)
	}

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin bootstrap: %w", err)
	}
	defer tx.Rollback(ctx)

	var orgID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO organizations (name, slug)
		VALUES ($1, $2)
		ON CONFLICT (slug) DO UPDATE
		SET name = EXCLUDED.name,
		    updated_at = now()
		RETURNING id
	`, cfg.OrganizationName, cfg.OrganizationSlug).Scan(&orgID); err != nil {
		return fmt.Errorf("upsert bootstrap organization: %w", err)
	}

	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (email, display_name, password_hash)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE
		SET display_name = EXCLUDED.display_name,
		    password_hash = COALESCE(users.password_hash, EXCLUDED.password_hash),
		    updated_at = now()
		RETURNING id
	`, email, "Owner", string(hash)).Scan(&userID); err != nil {
		return fmt.Errorf("upsert bootstrap owner: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO memberships (organization_id, user_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT (organization_id, user_id, role) DO NOTHING
	`, orgID, userID); err != nil {
		return fmt.Errorf("upsert bootstrap owner membership: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO audit_events (organization_id, actor_user_id, action, target_type, target_id, metadata)
		VALUES ($1, $2, 'bootstrap.owner', 'organization', $1, jsonb_build_object('owner_email', $3))
	`, orgID, userID, email); err != nil {
		return fmt.Errorf("record bootstrap audit event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit bootstrap: %w", err)
	}

	return nil
}
