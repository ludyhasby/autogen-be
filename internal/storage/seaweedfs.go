package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type SeaweedFS struct {
	Client                        *s3.Client
	PresignClient                 *s3.PresignClient
	S3AccessKey                   string
	S3SecretKey                   string
	Bucket                        string
	DefaultExpiresTimeOfSignedURL time.Duration
}

func NewSeaweedFS(endpoint string, region string, s3AccessKey string, s3SecretKey string, bucket string) *SeaweedFS {
	client := s3.New(s3.Options{
		Region:       region,
		BaseEndpoint: aws.String(endpoint),
		Credentials: credentials.NewStaticCredentialsProvider(
			s3AccessKey, s3SecretKey, ""),
		UsePathStyle: true,
	})
	return &SeaweedFS{
		Client:                        client,
		PresignClient:                 s3.NewPresignClient(client),
		S3AccessKey:                   s3AccessKey,
		S3SecretKey:                   s3SecretKey,
		Bucket:                        bucket,
		DefaultExpiresTimeOfSignedURL: time.Second * 60 * 30,
	}
}

func (s *SeaweedFS) PutObject(ctx context.Context, objectKey string, file io.Reader, contentType string, size int64) (err error) {
	if objectKey == "" {
		return fmt.Errorf("object key is required")
	}
	if file == nil {
		return fmt.Errorf("file reader is required")
	}
	if size < 0 {
		return fmt.Errorf("file size can not be negative")
	}
	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.Bucket),
		Key:           aws.String(objectKey),
		Body:          file,
		ContentLength: aws.Int64(size),
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}

	_, err = s.Client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("upload object %q to SeaweedFS: %w", objectKey, err)
	}

	return
}

func (s *SeaweedFS) SignURL(ctx context.Context, method string, objectKey string, expiredIn time.Duration) (url string, err error) {
	if expiredIn < 0 {
		expiredIn = s.DefaultExpiresTimeOfSignedURL
	}
	if objectKey == "" {
		return "", fmt.Errorf("object key is required")
	}

	var (
		result *v4.PresignedHTTPRequest
	)

	switch strings.ToUpper(method) {
	case http.MethodGet:
		result, err = s.PresignClient.PresignGetObject(
			ctx,
			&s3.GetObjectInput{
				Bucket: aws.String(s.Bucket),
				Key:    aws.String(objectKey),
			},
			func(opts *s3.PresignOptions) {
				opts.Expires = expiredIn
			},
		)

	case http.MethodPut:
		result, err = s.PresignClient.PresignPutObject(
			ctx,
			&s3.PutObjectInput{
				Bucket: aws.String(s.Bucket),
				Key:    aws.String(objectKey),
			},
			func(opts *s3.PresignOptions) {
				opts.Expires = expiredIn
			},
		)

	default:
		return "", fmt.Errorf("unsupported presigned URL method %q", method)
	}

	if err != nil {
		return "", fmt.Errorf("generate presigned URL: %w", err)
	}

	return result.URL, nil
}

func (s *SeaweedFS) Delete(ctx context.Context, objectKey string) error {
	if objectKey == "" {
		return fmt.Errorf("object key is required")
	}

	_, err := s.Client.DeleteObject(ctx,
		&s3.DeleteObjectInput{
			Bucket: aws.String(s.Bucket),
			Key:    aws.String(objectKey),
		},
	)
	if err != nil {
		return fmt.Errorf("delete object %q: %w", objectKey, err)
	}

	return nil
}
