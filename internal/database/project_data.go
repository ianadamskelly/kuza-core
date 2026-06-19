package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type ProjectTable struct {
	ID          string          `json:"id"`
	ProjectID   string          `json:"project_id"`
	Name        string          `json:"name"`
	Schema      json.RawMessage `json:"schema"`
	ReadAccess  string          `json:"read_access"`
	WriteAccess string          `json:"write_access"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type CreateProjectTableParams struct {
	Name        string          `json:"name"`
	Schema      json.RawMessage `json:"schema"`
	ReadAccess  string          `json:"read_access"`
	WriteAccess string          `json:"write_access"`
}

type ProjectTableAccess struct {
	ReadAccess  string
	WriteAccess string
}

type ProjectRecord struct {
	ID              string          `json:"id"`
	ProjectID       string          `json:"project_id"`
	TableID         string          `json:"table_id"`
	Data            json.RawMessage `json:"data"`
	CreatedByUserID *string         `json:"created_by_user_id,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type CreateProjectRecordParams struct {
	Data json.RawMessage `json:"data"`
}

type UpdateProjectRecordParams struct {
	Data json.RawMessage `json:"data"`
}

func (db *DB) ListProjectTables(ctx context.Context, projectID string) ([]ProjectTable, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, project_id, name, schema, read_access, write_access, created_at, updated_at
		FROM project_tables
		WHERE project_id = $1
		ORDER BY name ASC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("query project tables: %w", err)
	}
	defer rows.Close()

	tables := []ProjectTable{}
	for rows.Next() {
		var table ProjectTable
		if err := rows.Scan(&table.ID, &table.ProjectID, &table.Name, &table.Schema, &table.ReadAccess, &table.WriteAccess, &table.CreatedAt, &table.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan project table: %w", err)
		}
		tables = append(tables, table)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project tables: %w", err)
	}

	return tables, nil
}

func (db *DB) CreateProjectTable(ctx context.Context, projectID string, input CreateProjectTableParams) (ProjectTable, error) {
	input.Name = strings.TrimSpace(input.Name)
	if projectID == "" || input.Name == "" {
		return ProjectTable{}, fmt.Errorf("%w: project id and table name are required", ErrInvalidInput)
	}
	if len(input.Schema) == 0 {
		input.Schema = json.RawMessage(`{}`)
	}
	input.ReadAccess = normalizeTableAccess(input.ReadAccess)
	input.WriteAccess = normalizeTableAccess(input.WriteAccess)
	if !json.Valid(input.Schema) {
		return ProjectTable{}, fmt.Errorf("%w: schema must be valid json", ErrInvalidInput)
	}

	var table ProjectTable
	if err := db.pool.QueryRow(ctx, `
		INSERT INTO project_tables (project_id, name, schema, read_access, write_access)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, project_id, name, schema, read_access, write_access, created_at, updated_at
	`, projectID, input.Name, input.Schema, input.ReadAccess, input.WriteAccess).Scan(
		&table.ID,
		&table.ProjectID,
		&table.Name,
		&table.Schema,
		&table.ReadAccess,
		&table.WriteAccess,
		&table.CreatedAt,
		&table.UpdatedAt,
	); err != nil {
		return ProjectTable{}, fmt.Errorf("insert project table: %w", err)
	}

	return table, nil
}

func (db *DB) GetProjectTableAccess(ctx context.Context, projectID, tableName string) (ProjectTableAccess, error) {
	var access ProjectTableAccess
	if err := db.pool.QueryRow(ctx, `
		SELECT read_access, write_access
		FROM project_tables
		WHERE project_id = $1
		  AND name = $2
	`, projectID, tableName).Scan(&access.ReadAccess, &access.WriteAccess); err != nil {
		return ProjectTableAccess{}, fmt.Errorf("query project table access: %w", err)
	}
	return access, nil
}

func (db *DB) ListProjectRecords(ctx context.Context, projectID, tableName string) ([]ProjectRecord, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT project_records.id,
		       project_records.project_id,
		       project_records.table_id,
		       project_records.data,
		       project_records.created_by_user_id,
		       project_records.created_at,
		       project_records.updated_at
		FROM project_records
		JOIN project_tables ON project_tables.id = project_records.table_id
		WHERE project_records.project_id = $1
		  AND project_tables.name = $2
		ORDER BY project_records.created_at DESC
	`, projectID, tableName)
	if err != nil {
		return nil, fmt.Errorf("query project records: %w", err)
	}
	defer rows.Close()

	records := []ProjectRecord{}
	for rows.Next() {
		var record ProjectRecord
		if err := rows.Scan(
			&record.ID,
			&record.ProjectID,
			&record.TableID,
			&record.Data,
			&record.CreatedByUserID,
			&record.CreatedAt,
			&record.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan project record: %w", err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate project records: %w", err)
	}

	return records, nil
}

func normalizeTableAccess(access string) string {
	switch strings.TrimSpace(access) {
	case "api_key", "public":
		return strings.TrimSpace(access)
	default:
		return "project_members"
	}
}

func (db *DB) CreateProjectRecord(ctx context.Context, projectID, tableName, createdByUserID string, input CreateProjectRecordParams) (ProjectRecord, error) {
	tableName = strings.TrimSpace(tableName)
	if projectID == "" || tableName == "" {
		return ProjectRecord{}, fmt.Errorf("%w: project id and table name are required", ErrInvalidInput)
	}
	if len(input.Data) == 0 {
		input.Data = json.RawMessage(`{}`)
	}
	if !json.Valid(input.Data) {
		return ProjectRecord{}, fmt.Errorf("%w: data must be valid json", ErrInvalidInput)
	}
	if err := db.validateRecordData(ctx, projectID, tableName, input.Data); err != nil {
		return ProjectRecord{}, err
	}
	var createdBy *string
	if strings.TrimSpace(createdByUserID) != "" {
		createdBy = &createdByUserID
	}

	var record ProjectRecord
	if err := db.pool.QueryRow(ctx, `
		WITH table_ref AS (
			SELECT id
			FROM project_tables
			WHERE project_id = $1
			  AND name = $2
		)
		INSERT INTO project_records (project_id, table_id, data, created_by_user_id)
		SELECT $1, table_ref.id, $3, $4
		FROM table_ref
		RETURNING id, project_id, table_id, data, created_by_user_id, created_at, updated_at
	`, projectID, tableName, input.Data, createdBy).Scan(
		&record.ID,
		&record.ProjectID,
		&record.TableID,
		&record.Data,
		&record.CreatedByUserID,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return ProjectRecord{}, fmt.Errorf("insert project record: %w", err)
	}

	return record, nil
}

func (db *DB) UpdateProjectRecord(ctx context.Context, projectID, tableName, recordID string, input UpdateProjectRecordParams) (ProjectRecord, error) {
	tableName = strings.TrimSpace(tableName)
	recordID = strings.TrimSpace(recordID)
	if projectID == "" || tableName == "" || recordID == "" {
		return ProjectRecord{}, fmt.Errorf("%w: project id, table name, and record id are required", ErrInvalidInput)
	}
	if len(input.Data) == 0 {
		return ProjectRecord{}, fmt.Errorf("%w: data is required", ErrInvalidInput)
	}
	if !json.Valid(input.Data) {
		return ProjectRecord{}, fmt.Errorf("%w: data must be valid json", ErrInvalidInput)
	}
	if err := db.validateRecordData(ctx, projectID, tableName, input.Data); err != nil {
		return ProjectRecord{}, err
	}

	var record ProjectRecord
	if err := db.pool.QueryRow(ctx, `
		UPDATE project_records
		SET data = $4,
		    updated_at = now()
		FROM project_tables
		WHERE project_records.table_id = project_tables.id
		  AND project_records.project_id = $1
		  AND project_tables.name = $2
		  AND project_records.id = $3
		RETURNING project_records.id,
		          project_records.project_id,
		          project_records.table_id,
		          project_records.data,
		          project_records.created_by_user_id,
		          project_records.created_at,
		          project_records.updated_at
	`, projectID, tableName, recordID, input.Data).Scan(
		&record.ID,
		&record.ProjectID,
		&record.TableID,
		&record.Data,
		&record.CreatedByUserID,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		return ProjectRecord{}, fmt.Errorf("update project record: %w", err)
	}

	return record, nil
}

func (db *DB) DeleteProjectRecord(ctx context.Context, projectID, tableName, recordID string) error {
	tableName = strings.TrimSpace(tableName)
	recordID = strings.TrimSpace(recordID)
	if projectID == "" || tableName == "" || recordID == "" {
		return fmt.Errorf("%w: project id, table name, and record id are required", ErrInvalidInput)
	}

	tag, err := db.pool.Exec(ctx, `
		DELETE FROM project_records
		USING project_tables
		WHERE project_records.table_id = project_tables.id
		  AND project_records.project_id = $1
		  AND project_tables.name = $2
		  AND project_records.id = $3
	`, projectID, tableName, recordID)
	if err != nil {
		return fmt.Errorf("delete project record: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("%w: record not found", ErrInvalidInput)
	}

	return nil
}

type tableSchema struct {
	Fields   map[string]string `json:"fields"`
	Required []string          `json:"required"`
}

func (db *DB) validateRecordData(ctx context.Context, projectID, tableName string, data json.RawMessage) error {
	var rawSchema json.RawMessage
	if err := db.pool.QueryRow(ctx, `
		SELECT schema
		FROM project_tables
		WHERE project_id = $1
		  AND name = $2
	`, projectID, tableName).Scan(&rawSchema); err != nil {
		return fmt.Errorf("query project table schema: %w", err)
	}
	return validateRecordData(rawSchema, data)
}

func validateRecordData(rawSchema, data json.RawMessage) error {
	var record map[string]any
	if err := json.Unmarshal(data, &record); err != nil {
		return fmt.Errorf("%w: data must be a json object", ErrInvalidInput)
	}
	if record == nil {
		return fmt.Errorf("%w: data must be a json object", ErrInvalidInput)
	}

	var schema tableSchema
	if len(rawSchema) > 0 {
		if err := json.Unmarshal(rawSchema, &schema); err != nil {
			return fmt.Errorf("%w: table schema is invalid", ErrInvalidInput)
		}
	}

	for _, field := range schema.Required {
		if _, ok := record[field]; !ok {
			return fmt.Errorf("%w: missing required field %q", ErrInvalidInput, field)
		}
	}

	for field, expectedType := range schema.Fields {
		value, ok := record[field]
		if !ok || value == nil {
			continue
		}
		if !matchesSchemaType(value, expectedType) {
			return fmt.Errorf("%w: field %q must be %s", ErrInvalidInput, field, expectedType)
		}
	}

	return nil
}

func matchesSchemaType(value any, expectedType string) bool {
	switch expectedType {
	case "text", "string":
		_, ok := value.(string)
		return ok
	case "number":
		_, ok := value.(float64)
		return ok
	case "boolean", "bool":
		_, ok := value.(bool)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	default:
		return true
	}
}
