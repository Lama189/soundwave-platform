package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/Lama189/soundwave-platform/catalog-service/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinioStorage struct {
	client        *minio.Client
	presignClient *minio.Client
	rawBucket     string
	audioBucket   string
	coversBucket  string
	uploadExpiry  time.Duration
}

func NewMinioStorage(ctx context.Context, cfg config.StorageConfig) (*MinioStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("init minio client: %w", err)
	}

	presignEndpoint := cfg.Endpoint
	if cfg.PublicEndpoint != "" {
		presignEndpoint = cfg.PublicEndpoint
	}

	presignClient, err := minio.New(presignEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: "us-east-1",
	})
	if err != nil {
		return nil, fmt.Errorf("init minio presign client: %w", err)
	}

	storage := &MinioStorage{
		client:        client,
		presignClient: presignClient,
		rawBucket:     cfg.RawBucket,
		audioBucket:   cfg.AudioBucket,
		coversBucket:  cfg.CoversBucket,
		uploadExpiry:  cfg.UploadExpiry,
	}

	if err := storage.ensureBuckets(ctx); err != nil {
		return nil, fmt.Errorf("ensure buckets exist: %w", err)
	}

	return storage, nil
}

func (s *MinioStorage) GenerateUploadURL(ctx context.Context, key string) (string, error) {
	presignedURL, err := s.presignClient.PresignedPutObject(ctx, s.rawBucket, key, s.uploadExpiry)
	if err != nil {
		return "", fmt.Errorf("generate presigned put url for %s in %s: %w", key, s.rawBucket, err)
	}

	return presignedURL.String(), nil
}

func (s *MinioStorage) GenerateCoverUploadURL(ctx context.Context, key string) (string, error) {
	presignedURL, err := s.presignClient.PresignedPutObject(ctx, s.coversBucket, key, s.uploadExpiry)
	if err != nil {
		return "", fmt.Errorf("generate presigned put url for %s in %s: %w", key, s.coversBucket, err)
	}

	return presignedURL.String(), nil
}

func (s *MinioStorage) GenerateAudioDownloadURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if expiry <= 0 {
		expiry = s.uploadExpiry
	}
	presignedURL, err := s.presignClient.PresignedGetObject(ctx, s.audioBucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("generate presigned get url for %s in %s: %w", key, s.audioBucket, err)
	}

	return presignedURL.String(), nil
}

func (s *MinioStorage) ObjectExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.StatObject(ctx, s.rawBucket, key, minio.GetObjectOptions{})
	if err != nil {
		errResp := minio.ToErrorResponse(err)
		
		if errResp.Code == "NoSuchKey" {
			return false, nil
		}

		return false, fmt.Errorf("stat object %s in %s: %w", key, s.rawBucket, err)
	}

	return true, nil
}

func (s *MinioStorage) ensureBuckets(ctx context.Context) error {
	buckets := []string{s.rawBucket, s.audioBucket, s.coversBucket}
	for _, b := range buckets {
		exists, err := s.client.BucketExists(ctx, b)
		if err != nil {
			return fmt.Errorf("check bucket %s: %w", b, err)
		}

		if !exists {
			err := s.client.MakeBucket(ctx, b, minio.MakeBucketOptions{})
			if err != nil {
				return fmt.Errorf("create bucket %s: %w", b, err)
			}
		}
	}

	policy := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}
		]
	}`, s.coversBucket)

	_ = s.client.SetBucketPolicy(ctx, s.coversBucket, policy)

	return nil
}
