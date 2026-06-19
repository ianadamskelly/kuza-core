package objectstore

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"kuza-core/internal/database"
)

// Signer uses the MinIO Go client as a generic S3-compatible signer.
// It works with Garage, MinIO, AWS S3, and other compatible object stores.
type Signer struct {
	client *minio.Client
	expiry time.Duration
}

func New(endpoint, accessKey, secretKey string) (*Signer, error) {
	if endpoint == "" || accessKey == "" || secretKey == "" {
		return nil, nil
	}

	secure := false
	host := endpoint
	if parsed, err := url.Parse(endpoint); err == nil && parsed.Host != "" {
		host = parsed.Host
		secure = parsed.Scheme == "https"
	}
	host = strings.TrimRight(host, "/")
	if host == "" {
		return nil, fmt.Errorf("storage endpoint is empty")
	}

	client, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	if err != nil {
		return nil, err
	}

	return &Signer{
		client: client,
		expiry: 15 * time.Minute,
	}, nil
}

func (signer *Signer) PresignUpload(ctx context.Context, file database.File) (database.StorageOperation, error) {
	url, err := signer.client.PresignedPutObject(ctx, file.Bucket, file.ObjectKey, signer.expiry)
	if err != nil {
		return database.StorageOperation{}, err
	}

	return database.StorageOperation{
		Method: "PUT",
		URL:    url.String(),
		Header: map[string]string{"Content-Type": file.MimeType},
	}, nil
}

func (signer *Signer) PresignDownload(ctx context.Context, file database.File) (database.StorageOperation, error) {
	url, err := signer.client.PresignedGetObject(ctx, file.Bucket, file.ObjectKey, signer.expiry, nil)
	if err != nil {
		return database.StorageOperation{}, err
	}

	return database.StorageOperation{
		Method: "GET",
		URL:    url.String(),
	}, nil
}
