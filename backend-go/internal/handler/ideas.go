package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// ideaDetail 灵感详情（含标签与媒体）。
func (h *Handler) ideaDetail(ideaID uint) gin.H {
	var id model.Idea
	if err := h.db.First(&id, ideaID).Error; err != nil {
		return nil
	}
	return h.ideaDetailOf(&id)
}

// ideaDetailOf 用已取到的灵感行组装详情，避免再查一次主行。
func (h *Handler) ideaDetailOf(id *model.Idea) gin.H {
	var links []model.IdeaTag
	h.db.Where("idea_id = ?", id.ID).Find(&links)
	ids := make([]uint, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.TagID)
	}
	return gin.H{
		"id":                id.ID,
		"userId":            id.UserID,
		"title":             id.Title,
		"content":           id.Content,
		"linkedPlanId":      id.LinkedPlanID,
		"linkedMilestoneId": id.LinkedMilestoneID,
		"createdAt":         id.CreatedAt,
		"updatedAt":         id.UpdatedAt,
		"tags":              h.tagsByIDs(ids),
		"media":             h.mediaByEntity("idea", id.ID),
	}
}

// ListIdeas 灵感列表（支持分页，供前端滚动加载）。
func (h *Handler) ListIdeas(c *gin.Context) {
	cu := currentUser(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	q := h.db.Model(&model.Idea{}).Where("user_id = ?", cu.ID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		serverError(c, "查询失败")
		return
	}

	var all []model.Idea
	if err := q.Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&all).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	// 标签/媒体批量加载：原先每条灵感 4 条 SQL，现在固定 3 条。
	ids := make([]uint, 0, len(all))
	for i := range all {
		ids = append(ids, all[i].ID)
	}
	tagMap := h.tagsByEntities("idea_tags", "idea_id", ids)
	mediaMap := h.mediaByEntities("idea", ids)

	items := make([]gin.H, 0, len(all))
	for i := range all {
		it := &all[i]
		items = append(items, gin.H{
			"id":                it.ID,
			"userId":            it.UserID,
			"title":             it.Title,
			"content":           it.Content,
			"linkedPlanId":      it.LinkedPlanID,
			"linkedMilestoneId": it.LinkedMilestoneID,
			"createdAt":         it.CreatedAt,
			"updatedAt":         it.UpdatedAt,
			"tags":              tagMap[it.ID],
			"media":             mediaMap[it.ID],
		})
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

// GetIdea 灵感详情。
func (h *Handler) GetIdea(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var idea model.Idea
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&idea).Error; err != nil {
		notFound(c, "灵感不存在")
		return
	}
	c.JSON(200, h.ideaDetailOf(&idea))
}

// CreateIdea 新建灵感。
func (h *Handler) CreateIdea(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		Title             string `json:"title"`
		Content           string `json:"content"`
		LinkedPlanID      *uint  `json:"linkedPlanId"`
		LinkedMilestoneID *uint  `json:"linkedMilestoneId"`
		MediaIDs          []uint `json:"mediaIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	if req.Title == "" && req.Content == "" {
		badRequest(c, "请填写标题或内容")
		return
	}
	now := model.NowISO()
	idea := model.Idea{
		UserID:            cu.ID,
		Title:             req.Title,
		Content:           req.Content,
		LinkedPlanID:      req.LinkedPlanID,
		LinkedMilestoneID: req.LinkedMilestoneID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := h.db.Create(&idea).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	h.attachTagsAndMedia(cu.ID, "idea", idea.ID, nil, req.MediaIDs)
	c.JSON(200, h.ideaDetailOf(&idea))
}

// UpdateIdea 更新灵感。
func (h *Handler) UpdateIdea(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var idea model.Idea
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&idea).Error; err != nil {
		notFound(c, "灵感不存在")
		return
	}
	var req struct {
		Title             *string `json:"title"`
		Content           *string `json:"content"`
		LinkedPlanID      *uint   `json:"linkedPlanId"`
		LinkedMilestoneID *uint   `json:"linkedMilestoneId"`
		MediaIDs          *[]uint `json:"mediaIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	updates := map[string]interface{}{"updated_at": model.NowISO()}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Content != nil {
		updates["content"] = *req.Content
	}
	if req.LinkedPlanID != nil {
		updates["linked_plan_id"] = *req.LinkedPlanID
	}
	if req.LinkedMilestoneID != nil {
		updates["linked_milestone_id"] = *req.LinkedMilestoneID
	}
	if err := h.db.Model(&model.Idea{}).Where("id = ? AND user_id = ?", id, cu.ID).Updates(updates).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	if req.MediaIDs != nil {
		h.remountMediaOnUpdate(cu.ID, "idea", uint(id), *req.MediaIDs)
	}
	c.JSON(200, h.ideaDetail(uint(id)))
}

// DeleteIdea 删除灵感。
func (h *Handler) DeleteIdea(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var idea model.Idea
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&idea).Error; err != nil {
		notFound(c, "灵感不存在")
		return
	}
	h.db.Where("idea_id = ?", id).Delete(&model.IdeaTag{})
	h.db.Model(&model.Media{}).Where("user_id = ? AND entity_type = 'idea' AND entity_id = ?", cu.ID, id).
		Updates(map[string]interface{}{"entity_type": nil, "entity_id": nil})
	h.db.Delete(&idea)
	c.JSON(200, gin.H{"success": true})
}
