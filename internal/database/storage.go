package database

import (
	"context"
	"fmt"
	"path"
	"strings"
	"time"
)

type File struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	OwnerUserID *string   `json:"owner_user_id,omitempty"`
	Bucket      string    `json:"bucket"`
	ObjectKey   string    `json:"object_key"`
	FileName    string    `json:"file_name"`
	MimeType    string    `json:"mime_type"`
	ByteSize    int64     `json:"byte_size"`
	Access      string    `json:"access"`
	CreatedAt   time.Time `json:"created_at"`
}

type CreateFileIntentParams struct {
	FileName string `json:"file_name"`
	MimeType string `json:"mime_type"`
	ByteSize int64  `json:"byte_size"`
	Access   string `json:"access"`
}

type FileIntent struct {
	File     File             `json:"file"`
	Upload   StorageOperation `json:"upload"`
	Download StorageOperation `json:"download"`
}

type StorageOperation struct {
	Method string            `json:"method"`
	URL    string            `json:"url"`
	Header map[string]string `json:"headers,omitempty"`
}

func (db *DB) ListFiles(ctx context.Context, projectID string) ([]File, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, project_id, owner_user_id, bucket, object_key, object_key, mime_type, byte_size, access, created_at
		FROM files
		WHERE project_id = $1
		ORDER BY created_at DESC
	`, projectID)
	if err != nil {
		return nil, fmt.Errorf("query files: %w", err)
	}
	defer rows.Close()

	files := []File{}
	for rows.Next() {
		var file File
		if err := rows.Scan(&file.ID, &file.ProjectID, &file.OwnerUserID, &file.Bucket, &file.ObjectKey, &file.FileName, &file.MimeType, &file.ByteSize, &file.Access, &file.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan file: %w", err)
		}
		file.FileName = path.Base(file.ObjectKey)
		files = append(files, file)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate files: %w", err)
	}

	return files, nil
}

func (db *DB) CreateFileIntent(ctx context.Context, projectID, ownerUserID, bucket, publicURL string, input CreateFileIntentParams) (FileIntent, error) {
	input.FileName = path.Base(strings.TrimSpace(input.FileName))
	input.MimeType = strings.TrimSpace(input.MimeType)
	input.Access = normalizeFileAccess(input.Access)
	if projectID == "" || bucket == "" || input.FileName == "" {
		return FileIntent{}, fmt.Errorf("%w: project id, bucket, and file name are required", ErrInvalidInput)
	}
	if input.MimeType == "" {
		input.MimeType = "application/octet-stream"
	}
	if input.ByteSize < 0 {
		return FileIntent{}, fmt.Errorf("%w: byte_size cannot be negative", ErrInvalidInput)
	}

	objectKey := fmt.Sprintf("projects/%s/%d-%s", projectID, time.Now().UTC().UnixNano(), input.FileName)
	var owner *string
	if strings.TrimSpace(ownerUserID) != "" {
		owner = &ownerUserID
	}

	var file File
	if err := db.pool.QueryRow(ctx, `
		INSERT INTO files (project_id, owner_user_id, bucket, object_key, mime_type, byte_size, access)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, project_id, owner_user_id, bucket, object_key, mime_type, byte_size, access, created_at
	`, projectID, owner, bucket, objectKey, input.MimeType, input.ByteSize, input.Access).Scan(
		&file.ID,
		&file.ProjectID,
		&file.OwnerUserID,
		&file.Bucket,
		&file.ObjectKey,
		&file.MimeType,
		&file.ByteSize,
		&file.Access,
		&file.CreatedAt,
	); err != nil {
		return FileIntent{}, fmt.Errorf("insert file: %w", err)
	}
	file.FileName = input.FileName

	baseURL := strings.TrimRight(publicURL, "/")
	if baseURL == "" {
		baseURL = "/"
	}
	uploadURL := fmt.Sprintf("%s/v1/projects/%s/files/%s/object", baseURL, projectID, file.ID)
	downloadURL := uploadURL

	return FileIntent{
		File: file,
		Upload: StorageOperation{
			Method: "PUT",
			URL:    uploadURL,
			Header: map[string]string{"Content-Type": input.MimeType},
		},
		Download: StorageOperation{
			Method: "GET",
			URL:    downloadURL,
		},
	}, nil
}

func (db *DB) GetFile(ctx context.Context, projectID, fileID string) (File, error) {
	var file File
	if err := db.pool.QueryRow(ctx, `
		SELECT id, project_id, owner_user_id, bucket, object_key, mime_type, byte_size, access, created_at
		FROM files
		WHERE project_id = $1
		  AND id = $2
	`, projectID, fileID).Scan(
		&file.ID,
		&file.ProjectID,
		&file.OwnerUserID,
		&file.Bucket,
		&file.ObjectKey,
		&file.MimeType,
		&file.ByteSize,
		&file.Access,
		&file.CreatedAt,
	); err != nil {
		return File{}, fmt.Errorf("query file: %w", err)
	}
	file.FileName = path.Base(file.ObjectKey)
	return file, nil
}

func normalizeFileAccess(access string) string {
	switch strings.TrimSpace(access) {
	case "api_key", "public":
		return strings.TrimSpace(access)
	default:
		return "project_members"
	}
}
