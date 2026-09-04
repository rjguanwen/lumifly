package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// ListBookDomains 领域列表。
func (h *Handler) ListBookDomains(c *gin.Context) {
	cu := currentUser(c)
	var all []model.BookDomain
	if err := h.db.Where("user_id = ?", cu.ID).Order("created_at ASC, id ASC").Find(&all).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	items := make([]gin.H, 0, len(all))
	for i := range all {
		items = append(items, gin.H{
			"id": all[i].ID, "userId": all[i].UserID,
			"name": all[i].Name, "createdAt": all[i].CreatedAt,
		})
	}
	c.JSON(200, items)
}

// CreateBookDomain 新建领域（同名幂等，返回已有项）。
func (h *Handler) CreateBookDomain(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		badRequest(c, "领域名称不能为空")
		return
	}
	var existing model.BookDomain
	if err := h.db.Where("user_id = ? AND name = ?", cu.ID, name).First(&existing).Error; err == nil {
		c.JSON(200, gin.H{"id": existing.ID, "name": existing.Name})
		return
	}
	d := model.BookDomain{UserID: cu.ID, Name: name, CreatedAt: model.NowISO()}
	if err := h.db.Create(&d).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	c.JSON(200, gin.H{"id": d.ID, "name": d.Name})
}

// RenameBookDomain 重命名领域，并同步更新已使用该领域的书籍。
func (h *Handler) RenameBookDomain(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var d model.BookDomain
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&d).Error; err != nil {
		notFound(c, "领域不存在")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	newName := strings.TrimSpace(req.Name)
	if newName == "" {
		badRequest(c, "领域名称不能为空")
		return
	}
	if newName != d.Name {
		var dup model.BookDomain
		if err := h.db.Where("user_id = ? AND name = ? AND id <> ?", cu.ID, newName, d.ID).First(&dup).Error; err == nil {
			fail(c, 400, "该领域名称已存在")
			return
		}
		old := d.Name
		if err := h.db.Model(&model.BookDomain{}).Where("id = ?", d.ID).Update("name", newName).Error; err != nil {
			serverError(c, "保存失败")
			return
		}
		// 同步书籍：已使用旧领域名的书改为新名
		h.db.Model(&model.Book{}).Where("user_id = ? AND domain = ?", cu.ID, old).Update("domain", newName)
		d.Name = newName
	}
	c.JSON(200, gin.H{"id": d.ID, "name": d.Name})
}

// DeleteBookDomain 删除领域（书籍中原使用该领域的记录清空领域）。
func (h *Handler) DeleteBookDomain(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var d model.BookDomain
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&d).Error; err != nil {
		notFound(c, "领域不存在")
		return
	}
	// 同步清空书籍领域（必须用 map 更新以写入空字符串，单字段 Update 会忽略零值）
	h.db.Model(&model.Book{}).Where("user_id = ? AND domain = ?", cu.ID, d.Name).
		Updates(map[string]interface{}{"domain": ""})
	h.db.Delete(&d)
	c.JSON(200, gin.H{"success": true})
}
