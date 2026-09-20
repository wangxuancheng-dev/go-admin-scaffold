package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestS3Storage_URL(t *testing.T) {
	awsStyle := &s3Storage{bucket: "my-bucket", region: "ap-northeast-1", useSSL: true}
	got := awsStyle.URL("dir/a.txt")
	require.Equal(t, "https://my-bucket.s3.ap-northeast-1.amazonaws.com/dir/a.txt", got)

	httpStyle := &s3Storage{bucket: "my-bucket", region: "us-east-1", useSSL: false}
	got = httpStyle.URL("/dir/b.txt")
	require.Equal(t, "http://my-bucket.s3.us-east-1.amazonaws.com/dir/b.txt", got)

	custom := &s3Storage{bucket: "b", endpoint: "http://127.0.0.1:9000", region: "us-east-1"}
	got = custom.URL(`folder\c.txt`)
	require.Equal(t, "http://127.0.0.1:9000/b/folder/c.txt", got)
}

func TestNewS3Storage_validation(t *testing.T) {
	_, err := NewS3Storage(&S3Config{Bucket: "b"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "region")

	_, err = NewS3Storage(&S3Config{Region: "us-east-1"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "bucket")
}

func TestSanitizeRelativePath_empty(t *testing.T) {
	_, err := sanitizeRelativePath("   ")
	require.Error(t, err)
}
