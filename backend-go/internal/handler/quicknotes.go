package handler

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// quickNoteMaxRunes 速记语录内容最大字数。
const quickNoteMaxRunes = 360

// quickNoteDetail 速记语录详情（含标签）。
func (h *Handler) quickNoteDetail(noteID uint) gin.H {
	var n model.QuickNote
	if err := h.db.First(&n, noteID).Error; err != nil {
		return nil
	}
	var links []model.QuickNoteTag
	h.db.Where("quick_note_id = ?", noteID).Find(&links)
	ids := make([]uint, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.TagID)
	}
	return gin.H{
		"id":        n.ID,
		"userId":    n.UserID,
		"content":   n.Content,
		"createdAt": n.CreatedAt,
		"updatedAt": n.UpdatedAt,
		"tags":      h.tagsByIDs(ids),
	}
}

// syncQuickNoteTags 全量替换某条语录的标签关联。
func (h *Handler) syncQuickNoteTags(noteID uint, tagIDs []uint) {
	h.db.Where("quick_note_id = ?", noteID).Delete(&model.QuickNoteTag{})
	for _, tid := range tagIDs {
		h.db.Create(&model.QuickNoteTag{QuickNoteID: noteID, TagID: tid})
	}
}

func validateQuickNote(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return "内容不能为空"
	}
	if utf8.RuneCountInString(content) > quickNoteMaxRunes {
		return "内容不能超过 360 字"
	}
	return ""
}

// ListQuickNotes 速记语录列表（最新的在前，支持分页，供前端滚动加载）。
func (h *Handler) ListQuickNotes(c *gin.Context) {
	cu := currentUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	q := h.db.Model(&model.QuickNote{}).Where("user_id = ?", cu.ID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		serverError(c, "查询失败")
		return
	}

	var all []model.QuickNote
	if err := q.Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&all).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	items := make([]gin.H, 0, len(all))
	for _, n := range all {
		items = append(items, h.quickNoteDetail(n.ID))
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

// CreateQuickNote 新建速记语录。
func (h *Handler) CreateQuickNote(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		Content string `json:"content"`
		TagIDs  []uint `json:"tagIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if msg := validateQuickNote(req.Content); msg != "" {
		badRequest(c, msg)
		return
	}
	now := model.NowISO()
	n := model.QuickNote{
		UserID:    cu.ID,
		Content:   req.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.db.Create(&n).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	h.syncQuickNoteTags(n.ID, req.TagIDs)
	c.JSON(200, h.quickNoteDetail(n.ID))
}

// UpdateQuickNote 更新速记语录（可改内容与标签）。
func (h *Handler) UpdateQuickNote(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var note model.QuickNote
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&note).Error; err != nil {
		notFound(c, "速记语录不存在")
		return
	}
	var req struct {
		Content *string `json:"content"`
		TagIDs  *[]uint `json:"tagIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	updates := map[string]interface{}{"updated_at": model.NowISO()}
	if req.Content != nil {
		content := strings.TrimSpace(*req.Content)
		if msg := validateQuickNote(content); msg != "" {
			badRequest(c, msg)
			return
		}
		updates["content"] = content
	}
	if err := h.db.Model(&model.QuickNote{}).Where("id = ? AND user_id = ?", id, cu.ID).Updates(updates).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	if req.TagIDs != nil {
		h.syncQuickNoteTags(uint(id), *req.TagIDs)
	}
	c.JSON(200, h.quickNoteDetail(uint(id)))
}

// DeleteQuickNote 删除速记语录。
func (h *Handler) DeleteQuickNote(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var note model.QuickNote
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&note).Error; err != nil {
		notFound(c, "速记语录不存在")
		return
	}
	h.syncQuickNoteTags(uint(id), nil)
	h.db.Delete(&note)
	c.JSON(200, gin.H{"success": true})
}
