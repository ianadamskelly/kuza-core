package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type AuditEvent struct {
	ID          string          `json:"id"`
	ProjectID   *string         `json:"project_id,omitempty"`
	ActorUserID *string         `json:"actor_user_id,omitempty"`
	Action      string          `json:"action"`
	TargetType  string          `json:"target_type"`
	TargetID    *string         `json:"target_id,omitempty"`
	Metadata    json.RawMessage `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
}

type CreateAuditEventParams struct {
	ProjectID   string         `json:"project_id"`
	ActorUserID string         `json:"actor_user_id"`
	Action      string         `json:"action"`
	TargetType  string         `json:"target_type"`
	TargetID    string         `json:"target_id"`
	Metadata    map[string]any `json:"metadata"`
}

func (db *DB) CreateAuditEvent(ctx context.Context, input CreateAuditEventParams) error {
	if input.Action == "" || input.TargetType == "" {
		return fmt.Errorf("%w: action and target type are required", ErrInvalidInput)
	}

	metadata := input.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	rawMetadata, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	var projectID *string
	if input.ProjectID != "" {
		projectID = &input.ProjectID
	}
	var actorUserID *string
	if input.ActorUserID != "" {
		actorUserID = &input.ActorUserID
	}
	var targetID *string
	if input.TargetID != "" {
		targetID = &input.TargetID
	}

	if _, err := db.pool.Exec(ctx, `
		INSERT INTO audit_events (project_id, actor_user_id, action, target_type, target_id, metadata)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, projectID, actorUserID, input.Action, input.TargetType, targetID, rawMetadata); err != nil {
		return fmt.Errorf("insert audit event: %w", err)
	}

	return nil
}

func (db *DB) ListAuditEvents(ctx context.Context, projectID string) ([]AuditEvent, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, project_id, actor_user_id, action, target_type, target_id, metadata, created_at
		FROM audit_events
		WHERE project_id = $1
		ORDER BY created_at DESC
		LIMIT 200
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("query audit events: %w", err)
	}
	defer rows.Close()

	events := []AuditEvent{}
	for rows.Next() {
		var event AuditEvent
		if err := rows.Scan(&event.ID, &event.ProjectID, &event.ActorUserID, &event.Action, &event.TargetType, &event.TargetID, &event.Metadata, &event.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}

	return events, nil
}
