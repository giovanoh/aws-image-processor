package aws

import (
	"context"
	"fmt"
	"io"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	svc *s3.Client
}

func NewS3Client(cfg awsv2.Config) *S3Client {
	return &S3Client{
		svc: s3.NewFromConfig(cfg),
	}
}

func (c *S3Client) DownloadObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	result, err := c.svc.GetObject(ctx, &s3.GetObjectInput{
		Bucket: awsv2.String(bucket),
		Key:    awsv2.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("erro ao baixar objeto: %w", err)
	}

	return result.Body, nil
}

func (c *S3Client) UploadObject(ctx context.Context, bucket, key string, body io.Reader, contentType string) error {
	_, err := c.svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      awsv2.String(bucket),
		Key:         awsv2.String(key),
		Body:        body,
		ContentType: awsv2.String(contentType),
	})

	if err != nil {
		return fmt.Errorf("erro ao fazer upload: %w", err)
	}

	return nil
}
