package handler

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"lumiflybackend/internal/model"
)

// 上传文件类型与大小限制（与旧版一致）。
var allowedImageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true}
var allowedVideoTypes = map[string]bool{"video/mp4": true, "video/webm": true}

const (
	maxImageSize = 10 * 1024 * 1024  // 10MB
	maxVideoSize = 100 * 1024 * 1024 // 100MB
)

// UploadFile 上传文件（multipart: file + 可选 entityType/entityId）。
func (h *Handler) UploadFile(c *gin.Context) {
	cu := currentUser(c)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		badRequest(c, "请选择要上传的文件")
		return
	}

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		mimeType = extMime(ext)
	}

	isImage := allowedImageTypes[mimeType]
	isVideo := allowedVideoTypes[mimeType]
	if !isImage && !isVideo {
		badRequest(c, "不支持的文件类型，仅支持图片(jpg/png/gif/webp)和视频(mp4/webm)")
		return
	}
	if isImage && fileHeader.Size > maxImageSize {
		badRequest(c, "图片大小超过 10MB 限制")
		return
	}
	if isVideo && fileHeader.Size > maxVideoSize {
		badRequest(c, "视频大小超过 100MB 限制")
		return
	}

	subDir := "images"
	if isVideo {
		subDir = "videos"
	}
	dir := filepath.Join(h.uploadDir, subDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		serverError(c, "创建上传目录失败")
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" {
		ext = extForMime(mimeType)
	}
	fileName := uuid.NewString() + ext
	filePath := filepath.Join(dir, fileName)
	if err := c.SaveUploadedFile(fileHeader, filePath); err != nil {
		serverError(c, "保存文件失败")
		return
	}

	relativePath := subDir + "/" + fileName
	entityType := c.PostForm("entityType")
	var entityID *uint
	if idStr := c.PostForm("entityId"); idStr != "" {
		if idVal, err := strconv.ParseUint(idStr, 10, 32); err == nil {
			v := uint(idVal)
			entityID = &v
		}
	}

	m := model.Media{
		UserID:     cu.ID,
		FilePath:   relativePath,
		FileName:   fileHeader.Filename,
		MimeType:   mimeType,
		FileSize:   fileHeader.Size,
		EntityType: strPtr(entityType),
		EntityID:   entityID,
		CreatedAt:  model.NowISO(),
	}
	if err := h.db.Create(&m).Error; err != nil {
		os.Remove(filePath)
		serverError(c, "保存记录失败")
		return
	}

	c.JSON(200, gin.H{
		"id":       m.ID,
		"url":      "/api/uploads/" + relativePath,
		"fileName": m.FileName,
		"mimeType": m.MimeType,
		"fileSize": m.FileSize,
	})
}

// ServeFile 提供上传文件的静态访问（/api/uploads/images/xxx.png）。
func (h *Handler) ServeFile(c *gin.Context) {
	rel := strings.TrimPrefix(c.Param("filepath"), "/")
	if rel == "" {
		badRequest(c, "文件路径不能为空")
		return
	}
	// 防路径穿越
	if strings.Contains(rel, "..") {
		fail(c, 403, "禁止访问")
		return
	}
	full := filepath.Join(h.uploadDir, filepath.FromSlash(rel))
	if !strings.HasPrefix(full, h.uploadDir) {
		fail(c, 403, "禁止访问")
		return
	}
	if _, err := os.Stat(full); err != nil {
		notFound(c, "文件不存在")
		return
	}
	c.File(full)
}

func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func extMime(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".mp4":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	}
	return "application/octet-stream"
}

func extForMime(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/webm":
		return ".webm"
	}
	return ".bin"
}
