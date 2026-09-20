package handlers

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"go-admin-scaffold/internal/core/storage"
	"go-admin-scaffold/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadHandler handles file uploads against a Storage backend.
type UploadHandler struct {
	storage storage.Storage
}

// NewUploadHandler creates an upload handler.
func NewUploadHandler(store storage.Storage) *UploadHandler {
	return &UploadHandler{storage: store}
}

// UploadRequest is the form payload for uploads.
type UploadRequest struct {
	Type string `form:"type" binding:"required,oneof=avatar image file"`
}

// UploadResponse is returned after a successful upload.
type UploadResponse struct {
	URL  string `json:"url"`
	Path string `json:"path"`
	Name string `json:"name"`
	Size int64  `json:"size"`
	Type string `json:"type"`
}

// MultiUploadResponse aggregates multi-file upload results.
type MultiUploadResponse struct {
	Total   int              `json:"total"`
	Success int              `json:"success"`
	Failed  int              `json:"failed"`
	Files   []UploadResponse `json:"files"`
}

// Upload handles a single file upload.
func (h *UploadHandler) Upload(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBind(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.ParamError(c, "file is required")
		return
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !isAllowedFileType(req.Type, ext) {
		response.ParamError(c, fmt.Sprintf("invalid file type. allowed types: %s", getAllowedExtensions(req.Type)))
		return
	}
	if !isAllowedFileSize(req.Type, file.Size) {
		response.ParamError(c, fmt.Sprintf("file too large. maximum size: %dMB", getMaxFileSize(req.Type)/1024/1024))
		return
	}

	path := buildStoragePath(req.Type, ext)
	src, err := file.Open()
	if err != nil {
		response.ServerError(c)
		return
	}
	defer src.Close()

	url, err := h.storage.Put(path, src)
	if err != nil {
		response.ServerError(c)
		return
	}

	response.Success(c, UploadResponse{
		URL:  url,
		Path: path,
		Name: filepath.Base(path),
		Size: file.Size,
		Type: req.Type,
	})
}

// MultiUpload handles multiple file uploads.
func (h *UploadHandler) MultiUpload(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBind(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		response.ParamError(c, "failed to get form data")
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		response.ParamError(c, "no files uploaded")
		return
	}

	const maxFiles = 10
	if len(files) > maxFiles {
		response.ParamError(c, fmt.Sprintf("too many files. maximum allowed: %d", maxFiles))
		return
	}

	result := MultiUploadResponse{
		Total: len(files),
		Files: make([]UploadResponse, 0, len(files)),
	}

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !isAllowedFileType(req.Type, ext) || !isAllowedFileSize(req.Type, file.Size) {
			result.Failed++
			continue
		}

		path := buildStoragePath(req.Type, ext)
		src, err := file.Open()
		if err != nil {
			result.Failed++
			continue
		}

		url, err := h.storage.Put(path, src)
		src.Close()
		if err != nil {
			result.Failed++
			continue
		}

		result.Success++
		result.Files = append(result.Files, UploadResponse{
			URL:  url,
			Path: path,
			Name: filepath.Base(path),
			Size: file.Size,
			Type: req.Type,
		})
	}

	response.Success(c, result)
}

func isAllowedFileType(fileType, ext string) bool {
	switch fileType {
	case "avatar", "image":
		return isImageExt(ext)
	case "file":
		return isDocumentExt(ext)
	default:
		return false
	}
}

func isImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	default:
		return false
	}
}

func isDocumentExt(ext string) bool {
	switch ext {
	case ".pdf", ".txt", ".csv", ".doc", ".docx", ".xls", ".xlsx", ".zip":
		return true
	default:
		return false
	}
}

func getAllowedExtensions(fileType string) string {
	switch fileType {
	case "avatar", "image":
		return ".jpg, .jpeg, .png, .gif, .webp"
	case "file":
		return ".pdf, .txt, .csv, .doc, .docx, .xls, .xlsx, .zip"
	default:
		return ""
	}
}

func isAllowedFileSize(fileType string, size int64) bool {
	return size > 0 && size <= getMaxFileSize(fileType)
}

func getMaxFileSize(fileType string) int64 {
	switch fileType {
	case "avatar":
		return 2 * 1024 * 1024
	case "image":
		return 5 * 1024 * 1024
	case "file":
		return 10 * 1024 * 1024
	default:
		return 0
	}
}

// buildStoragePath uses a random UUID filename so client-supplied names never enter the path.
func buildStoragePath(fileType, ext string) string {
	now := time.Now()
	safeExt := sanitizeExt(ext)
	name := uuid.NewString() + safeExt
	return fmt.Sprintf("%s/%d/%02d/%02d/%s", fileType, now.Year(), int(now.Month()), now.Day(), name)
}

func sanitizeExt(ext string) string {
	ext = strings.ToLower(strings.TrimSpace(ext))
	if ext == "" || !strings.HasPrefix(ext, ".") || strings.ContainsAny(ext, "/\\") {
		return ""
	}
	return ext
}
