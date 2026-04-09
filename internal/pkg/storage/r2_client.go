package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/spatial-memory/spatial-memory/internal/config"
)

// Client is the interface for R2/S3-compatible object storage.
type Client interface {
	GeneratePresignedUploadURL(ctx context.Context, key, contentType string, expiresIn time.Duration) (string, error)
	GeneratePresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error)
	DeleteObject(ctx context.Context, key string) error
	HeadObject(ctx context.Context, key string) (*ObjectInfo, error)
}

type ObjectInfo struct {
	Size        int64
	ContentType string
	ETag        string
}

type r2Client struct {
	client    *s3.Client
	presign   *s3.PresignClient
	bucket    string
	publicURL string
}

// NewClient creates a new R2 storage client.
func NewClient(cfg config.R2Config) (Client, error) {
	if cfg.AccountID == "" || cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
		return nil, fmt.Errorf("R2 credentials not configured")
	}

	endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", cfg.AccountID)

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("auto"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.AccessKeySecret,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.EndpointResolver = s3.EndpointResolverFromURL(endpoint)
	})

	return &r2Client{
		client:    client,
		presign:   s3.NewPresignClient(client),
		bucket:    cfg.Bucket,
		publicURL: cfg.PublicURL,
	}, nil
}

func (c *r2Client) GeneratePresignedUploadURL(ctx context.Context, key, contentType string, expiresIn time.Duration) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	req, err := c.presign.PresignPutObject(ctx, input, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", fmt.Errorf("presign upload URL: %w", err)
	}

	return req.URL, nil
}

func (c *r2Client) GeneratePresignedDownloadURL(ctx context.Context, key string, expiresIn time.Duration) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	}

	req, err := c.presign.PresignGetObject(ctx, input, s3.WithPresignExpires(expiresIn))
	if err != nil {
		return "", fmt.Errorf("presign download URL: %w", err)
	}

	return req.URL, nil
}

func (c *r2Client) DeleteObject(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func (c *r2Client) HeadObject(ctx context.Context, key string) (*ObjectInfo, error) {
	head, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("head object: %w", err)
	}

	return &ObjectInfo{
		Size:        head.ContentLength,
		ContentType: aws.ToString(head.ContentType),
		ETag:        aws.ToString(head.ETag),
	}, nil
}
