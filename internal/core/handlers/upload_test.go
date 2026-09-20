package handlers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	corehandlers "go-admin-scaffold/internal/core/handlers"
	"go-admin-scaffold/internal/core/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUpload_rejectsTraversalAndDisallowedExt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	dir := t.TempDir()
	store, err := storage.NewLocalStorage(dir)
	require.NoError(t, err)
	h := corehandlers.NewUploadHandler(store)

	r := gin.New()
	r.POST("/upload", h.Upload)

	// path traversal via original filename must not land on disk as-is
	body, ctype := multipartFile(t, "file", "../../../etc/passwd.exe", []byte("x"), "file")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", ctype)
	r.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), "invalid file type")

	// allowed image uses UUID path under type/year/...
	body, ctype = multipartFile(t, "file", "avatar.png", []byte("img"), "avatar")
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", ctype)
	r.ServeHTTP(w, req)
	require.Contains(t, w.Body.String(), `/api/admin/v1/files/avatar/`)
	require.NotContains(t, w.Body.String(), "avatar.png")

	entries, err := filepath.Glob(filepath.Join(dir, "avatar", "*", "*", "*", "*.png"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
}

func multipartFile(t *testing.T, field, filename string, content []byte, fileType string) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("type", fileType))
	part, err := w.CreateFormFile(field, filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return &buf, w.FormDataContentType()
}

func TestUpload_documentAllowlist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store, err := storage.NewLocalStorage(t.TempDir())
	require.NoError(t, err)
	h := corehandlers.NewUploadHandler(store)
	r := gin.New()
	r.POST("/upload", h.Upload)

	body, ctype := multipartFile(t, "file", "ok.pdf", []byte("%PDF"), "file")
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/upload", body)
	req.Header.Set("Content-Type", ctype)
	r.ServeHTTP(w, req)
	require.True(t, strings.Contains(w.Body.String(), `"code":0`) || strings.Contains(w.Body.String(), `"code": 0`))
}
