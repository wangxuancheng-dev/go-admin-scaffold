package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const localURLPrefix = "/uploads"

type localStorage struct {
	baseDir string
}

// NewLocalStorage creates a local filesystem storage backend.
func NewLocalStorage(baseDir string) (Storage, error) {
	if strings.TrimSpace(baseDir) == "" {
		baseDir = "storage/uploads"
	}

	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("create local storage dir: %w", err)
	}

	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("resolve local storage dir: %w", err)
	}

	return &localStorage{baseDir: abs}, nil
}

func (s *localStorage) Put(path string, reader io.Reader) (string, error) {
	clean, err := sanitizeRelativePath(path)
	if err != nil {
		return "", err
	}

	fullPath := filepath.Join(s.baseDir, filepath.FromSlash(clean))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("create upload directory: %w", err)
	}

	f, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, reader); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return s.URL(clean), nil
}

func (s *localStorage) Delete(path string) error {
	clean, err := sanitizeRelativePath(path)
	if err != nil {
		return err
	}

	fullPath := filepath.Join(s.baseDir, filepath.FromSlash(clean))
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func (s *localStorage) URL(path string) string {
	clean := strings.TrimPrefix(filepath.ToSlash(path), "/")
	return localURLPrefix + "/" + clean
}

func sanitizeRelativePath(path string) (string, error) {
	clean := filepath.ToSlash(strings.TrimSpace(path))
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" || strings.Contains(clean, "..") {
		return "", fmt.Errorf("invalid path")
	}
	return clean, nil
}
