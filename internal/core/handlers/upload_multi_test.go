package handlers_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	corehandlers "go-admin-scaffold/internal/core/handlers"
	"go-admin-scaffold/internal/core/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestMultiUpload_successAndRejects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store, err := storage.NewLocalStorage(t.TempDir())
	require.NoError(t, err)
	h := corehandlers.NewUploadHandler(store)

	r := gin.New()
	r.POST("/multi", h.MultiUpload)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("type", "file"))
	part, err := w.CreateFormFile("files", "a.pdf")
	require.NoError(t, err)
	_, err = part.Write([]byte("%PDF-ok"))
	require.NoError(t, err)
	part2, err := w.CreateFormFile("files", "bad.exe")
	require.NoError(t, err)
	_, err = part2.Write([]byte("x"))
	require.NoError(t, err)
	require.NoError(t, w.Close())

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/multi", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(rec, req)

	body := rec.Body.String()
	require.True(t, strings.Contains(body, `"success":1`) || strings.Contains(body, `"success": 1`))
	require.True(t, strings.Contains(body, `"failed":1`) || strings.Contains(body, `"failed": 1`))
}

func TestMultiUpload_noFiles(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store, err := storage.NewLocalStorage(t.TempDir())
	require.NoError(t, err)
	h := corehandlers.NewUploadHandler(store)
	r := gin.New()
	r.POST("/multi", h.MultiUpload)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("type", "file"))
	require.NoError(t, w.Close())

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/multi", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(rec, req)
	require.Contains(t, rec.Body.String(), "no files")
}

func TestUpload_missingFile(t *testing.T) {
	gin.SetMode(gin.TestMode)
	store, err := storage.NewLocalStorage(t.TempDir())
	require.NoError(t, err)
	h := corehandlers.NewUploadHandler(store)
	r := gin.New()
	r.POST("/upload", h.Upload)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	require.NoError(t, w.WriteField("type", "file"))
	require.NoError(t, w.Close())

	rec := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/upload", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	r.ServeHTTP(rec, req)
	require.Contains(t, rec.Body.String(), "file is required")
}
