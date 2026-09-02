package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// ListTags 标签列表。
func (h *Handler) ListTags(c *gin.Context) {
	cu := currentUser(c)
	var all []model.Tag
	if err := h.db.Where("user_id = ?", cu.ID).Order("created_at ASC").Find(&all).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	items := make([]gin.H, 0, len(all))
	for i := range all {
		items = append(items, gin.H{
			"id": all[i].ID, "userId": all[i].UserID, "name": all[i].Name,
			"color": all[i].Color, "createdAt": all[i].CreatedAt,
		})
	}
	c.JSON(200, items)
}

// CreateTag 创建标签（同名幂等，返回已有标签）。
func (h *Handler) CreateTag(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		badRequest(c, "标签名称不能为空")
		return
	}
	var existing model.Tag
	if err := h.db.Where("user_id = ? AND name = ?", cu.ID, name).First(&existing).Error; err == nil {
		c.JSON(200, gin.H{
			"id": existing.ID, "userId": existing.UserID, "name": existing.Name,
			"color": existing.Color, "createdAt": existing.CreatedAt,
		})
		return
	}
	tag := model.Tag{
		UserID:    cu.ID,
		Name:      name,
		CreatedAt: model.NowISO(),
	}
	if req.Color != "" {
		tag.Color = &req.Color
	}
	if err := h.db.Create(&tag).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	c.JSON(200, gin.H{
		"id": tag.ID, "userId": tag.UserID, "name": tag.Name,
		"color": tag.Color, "createdAt": tag.CreatedAt,
	})
}
