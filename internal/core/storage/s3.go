package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type s3Storage struct {
	client   *s3.Client
	uploader *manager.Uploader
	bucket   string
	region   string
	endpoint string
	useSSL   bool
}

// NewS3Storage creates an S3-compatible storage backend.
func NewS3Storage(cfg *S3Config) (Storage, error) {
	if cfg == nil {
		return nil, fmt.Errorf("s3 config is required")
	}
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("s3 bucket is required")
	}
	if cfg.Region == "" {
		return nil, fmt.Errorf("s3 region is required")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	})

	return &s3Storage{
		client:   client,
		uploader: manager.NewUploader(client),
		bucket:   cfg.Bucket,
		region:   cfg.Region,
		endpoint: strings.TrimRight(cfg.Endpoint, "/"),
		useSSL:   cfg.UseSSL,
	}, nil
}

func (s *s3Storage) Put(objectPath string, reader io.Reader) (string, error) {
	clean, err := sanitizeRelativePath(objectPath)
	if err != nil {
		return "", err
	}

	input := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(clean),
		Body:   reader,
	}
	if ct := mime.TypeByExtension(path.Ext(clean)); ct != "" {
		input.ContentType = aws.String(ct)
	}

	if _, err := s.uploader.Upload(context.Background(), input); err != nil {
		return "", fmt.Errorf("upload to s3: %w", err)
	}

	return s.URL(clean), nil
}

func (s *s3Storage) Delete(objectPath string) error {
	clean, err := sanitizeRelativePath(objectPath)
	if err != nil {
		return err
	}

	_, err = s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(clean),
	})
	return err
}

func (s *s3Storage) URL(objectPath string) string {
	clean := strings.TrimPrefix(filepathToSlash(objectPath), "/")
	if s.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucket, clean)
	}

	scheme := "https"
	if !s.useSSL {
		scheme = "http"
	}
	host := fmt.Sprintf("%s.s3.%s.amazonaws.com", s.bucket, s.region)
	u := url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   "/" + clean,
	}
	return u.String()
}

func filepathToSlash(p string) string {
	return strings.ReplaceAll(p, "\\", "/")
}
