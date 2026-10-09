package storage

import (
	"context"
	"io"
	"time"
)

type ObjectStorage interface {
	PutObject(ctx context.Context, objectKey string, file io.Reader, contentType string, size int64)
	SignURL(ctx context.Context, method string, objectKey string, expiredIn time.Duration) (url string, err error)
	Delete(ctx context.Context, objectKey string) error
}
