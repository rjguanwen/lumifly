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

// avatarImageTypes 头像允许的图片类型。
var avatarImageTypes = map[string]bool{"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true}

const maxAvatarSize = 5 * 1024 * 1024 // 5MB

// UploadAvatar 上传当前用户头像（multipart: file）。头像存 uploads/avatars/，替换时删除旧文件。
func (h *Handler) UploadAvatar(c *gin.Context) {
	cu := currentUser(c)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		badRequest(c, "请选择要上传的图片")
		return
	}
	mimeType := fileHeader.Header.Get("Content-Type")
	if !avatarImageTypes[mimeType] {
		badRequest(c, "头像仅支持 jpg/png/gif/webp 图片")
		return
	}
	if fileHeader.Size > maxAvatarSize {
		badRequest(c, "头像图片不能超过 5MB")
		return
	}

	dir := filepath.Join(h.uploadDir, "avatars")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		serverError(c, "创建目录失败")
		return
	}
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext == "" {
		ext = extForMime(mimeType)
	}
	fileName := uuid.NewString() + ext
	if err := c.SaveUploadedFile(fileHeader, filepath.Join(dir, fileName)); err != nil {
		serverError(c, "保存头像失败")
		return
	}
	newURL := "/api/uploads/avatars/" + fileName

	// 删除旧头像（仅当旧头像确实位于本应用的 avatars 目录）
	var u model.User
	if err := h.db.First(&u, cu.ID).Error; err != nil {
		os.Remove(filepath.Join(dir, fileName))
		notFound(c, "用户不存在")
		return
	}
	if u.AvatarURL != nil && strings.HasPrefix(*u.AvatarURL, "/api/uploads/avatars/") {
		old := filepath.Join(h.uploadDir, strings.TrimPrefix(*u.AvatarURL, "/api/uploads/"))
		_ = os.Remove(old)
	}
	if err := h.db.Model(&model.User{}).Where("id = ?", cu.ID).
		Updates(map[string]interface{}{"avatar_url": newURL, "updated_at": model.NowISO()}).Error; err != nil {
		os.Remove(filepath.Join(dir, fileName))
		serverError(c, "保存失败")
		return
	}
	h.db.First(&u, cu.ID)
	c.JSON(200, userJSON(&u))
}

// DeleteMedia 删除媒体记录及其物理文件（仅本人上传的媒体）。
func (h *Handler) DeleteMedia(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var m model.Media
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&m).Error; err != nil {
		notFound(c, "附件不存在")
		return
	}
	// 删除物理文件
	full := filepath.Join(h.uploadDir, filepath.FromSlash(m.FilePath))
	if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
		serverError(c, "删除文件失败")
		return
	}
	h.db.Delete(&m)
	c.JSON(200, gin.H{"success": true})
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
