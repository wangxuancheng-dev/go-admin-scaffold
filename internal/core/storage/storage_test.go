package storage_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"go-admin-scaffold/internal/core/storage"

	"github.com/stretchr/testify/require"
)

func TestLocalStorage_PutDeleteURL(t *testing.T) {
	dir := t.TempDir()
	store, err := storage.NewStorage(&storage.Config{Driver: "local", LocalPath: dir})
	require.NoError(t, err)

	url, err := store.Put("a/b.txt", bytes.NewReader([]byte("hi")))
	require.NoError(t, err)
	require.Equal(t, "/uploads/a/b.txt", url)

	_, err = os.Stat(filepath.Join(dir, "a", "b.txt"))
	require.NoError(t, err)

	require.Equal(t, "/uploads/a/b.txt", store.URL("a/b.txt"))
	require.NoError(t, store.Delete("a/b.txt"))
}

func TestLocalStorage_rejectsTraversal(t *testing.T) {
	store, err := storage.NewStorage(&storage.Config{Driver: "local", LocalPath: t.TempDir()})
	require.NoError(t, err)
	_, err = store.Put("../x.txt", bytes.NewReader([]byte("x")))
	require.Error(t, err)
}

func TestNewStorage_unsupported(t *testing.T) {
	_, err := storage.NewStorage(&storage.Config{Driver: "ftp"})
	require.Error(t, err)
}

func TestNewStorage_nilConfig(t *testing.T) {
	_, err := storage.NewStorage(nil)
	require.Error(t, err)
}

func TestNewStorage_s3RequiresConfig(t *testing.T) {
	_, err := storage.NewStorage(&storage.Config{Driver: "s3"})
	require.Error(t, err)
	require.Contains(t, err.Error(), "s3 config")
}

func TestLocalStorage_emptyPathDefault(t *testing.T) {
	// empty LocalPath defaults under cwd; use relative temp via NewLocalStorage directly
	dir := t.TempDir()
	store, err := storage.NewLocalStorage(dir)
	require.NoError(t, err)
	url, err := store.Put("x.txt", bytes.NewReader([]byte("1")))
	require.NoError(t, err)
	require.Equal(t, "/uploads/x.txt", url)
	require.NoError(t, store.Delete("missing.txt"))
}
