package handlers

import (
	"go-admin-scaffold/internal/core/storage"
	"go-admin-scaffold/pkg/response"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// UploadHandler 处理文件上传
type UploadHandler struct {
	storage storage.Storage
}

// NewUploadHandler 创建上传处理器
func NewUploadHandler(storage storage.Storage) *UploadHandler {
	return &UploadHandler{
		storage: storage,
	}
}

// UploadRequest 上传请求
type UploadRequest struct {
	Type string `form:"type" binding:"required,oneof=avatar image file"` // 文件类型：avatar-头像，image-图片，file-其他文件
}

// UploadResponse 上传响应
type UploadResponse struct {
	URL  string `json:"url"`  // 文件访问URL
	Path string `json:"path"` // 文件存储路径
	Name string `json:"name"` // 文件名
	Size int64  `json:"size"` // 文件大小
	Type string `json:"type"` // 文件类型
}

// MultiUploadResponse 多文件上传响应
type MultiUploadResponse struct {
	Total   int              `json:"total"`   // 总文件数
	Success int              `json:"success"` // 成功上传数
	Failed  int              `json:"failed"`  // 失败数
	Files   []UploadResponse `json:"files"`   // 文件列表
}

// Upload 处理单文件上传
func (h *UploadHandler) Upload(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBind(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		response.ParamError(c, "file is required")
		return
	}

	// 验证文件类型
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !isAllowedFileType(req.Type, ext) {
		response.ParamError(c, fmt.Sprintf("invalid file type. allowed types: %s", getAllowedExtensions(req.Type)))
		return
	}

	// 验证文件大小
	if !isAllowedFileSize(req.Type, file.Size) {
		response.ParamError(c, fmt.Sprintf("file too large. maximum size: %dMB", getMaxFileSize(req.Type)/1024/1024))
		return
	}

	// 构建存储路径
	path := buildStoragePath(req.Type, file.Filename)

	// 上传文件
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
		Name: file.Filename,
		Size: file.Size,
		Type: req.Type,
	})
}

// MultiUpload 处理多文件上传
func (h *UploadHandler) MultiUpload(c *gin.Context) {
	var req UploadRequest
	if err := c.ShouldBind(&req); err != nil {
		response.ValidationError(c, err.Error())
		return
	}

	// 获取上传的文件
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

	// 验证文件数量
	maxFiles := 10 // 最大文件数
	if len(files) > maxFiles {
		response.ParamError(c, fmt.Sprintf("too many files. maximum allowed: %d", maxFiles))
		return
	}

	result := MultiUploadResponse{
		Total:   len(files),
		Success: 0,
		Failed:  0,
		Files:   make([]UploadResponse, 0, len(files)),
	}

	// 处理每个文件
	for _, file := range files {
		// 验证文件类型
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if !isAllowedFileType(req.Type, ext) {
			result.Failed++
			continue
		}

		// 验证文件大小
		if !isAllowedFileSize(req.Type, file.Size) {
			result.Failed++
			continue
		}

		// 构建存储路径
		path := buildStoragePath(req.Type, file.Filename)

		// 上传文件
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
			Name: file.Filename,
			Size: file.Size,
			Type: req.Type,
		})
	}

	response.Success(c, result)
}

// 检查文件类型是否允许
func isAllowedFileType(fileType, ext string) bool {
	switch fileType {
	case "avatar":
		return isImageExt(ext)
	case "image":
		return isImageExt(ext)
	case "file":
		return true
	default:
		return false
	}
}

// 检查是否是图片扩展名
func isImageExt(ext string) bool {
	allowedExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	for _, allowed := range allowedExts {
		if ext == allowed {
			return true
		}
	}
	return false
}

// 获取允许的文件扩展名
func getAllowedExtensions(fileType string) string {
	switch fileType {
	case "avatar", "image":
		return ".jpg, .jpeg, .png, .gif, .webp"
	case "file":
		return "all"
	default:
		return ""
	}
}

// 检查文件大小是否允许
func isAllowedFileSize(fileType string, size int64) bool {
	return size <= getMaxFileSize(fileType)
}

// 获取最大文件大小
func getMaxFileSize(fileType string) int64 {
	switch fileType {
	case "avatar":
		return 2 * 1024 * 1024 // 2MB
	case "image":
		return 5 * 1024 * 1024 // 5MB
	case "file":
		return 10 * 1024 * 1024 // 10MB
	default:
		return 0
	}
}

// 构建存储路径
func buildStoragePath(fileType, filename string) string {
	now := time.Now()
	return fmt.Sprintf("%s/%d/%02d/%02d/%s", fileType, now.Year(), now.Month(), now.Day(), filename)
}
