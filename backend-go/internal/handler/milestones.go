package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// milestoneDetail 大事记详情（含标签与媒体）。
func (h *Handler) milestoneDetail(milestoneID uint) gin.H {
	var ms model.Milestone
	if err := h.db.First(&ms, milestoneID).Error; err != nil {
		return nil
	}
	var links []model.MilestoneTag
	h.db.Where("milestone_id = ?", milestoneID).Find(&links)
	ids := make([]uint, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.TagID)
	}
	return gin.H{
		"id":          ms.ID,
		"userId":      ms.UserID,
		"title":       ms.Title,
		"description": ms.Description,
		"eventDate":   ms.EventDate,
		"endDate":     ms.EndDate,
		"category":    ms.Category,
		"importance":  ms.Importance,
		"createdAt":   ms.CreatedAt,
		"updatedAt":   ms.UpdatedAt,
		"tags":        h.tagsByIDs(ids),
		"media":       h.mediaByEntity("milestone", ms.ID),
	}
}

// ListMilestones 大事记列表（支持 category/from/to 过滤）。
func (h *Handler) ListMilestones(c *gin.Context) {
	cu := currentUser(c)
	category := c.Query("category")
	from := c.Query("from")
	to := c.Query("to")

	var all []model.Milestone
	q := h.db.Where("user_id = ?", cu.ID).Order("event_date DESC")
	if category != "" {
		q = q.Where("category = ?", category)
	}
	if from != "" {
		q = q.Where("event_date >= ?", from)
	}
	if to != "" {
		q = q.Where("event_date <= ?", to)
	}
	if err := q.Find(&all).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	items := make([]gin.H, 0, len(all))
	for _, m := range all {
		items = append(items, h.milestoneDetail(m.ID))
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

// GetMilestone 大事记详情。
func (h *Handler) GetMilestone(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var ms model.Milestone
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&ms).Error; err != nil {
		notFound(c, "大事记不存在")
		return
	}
	c.JSON(200, h.milestoneDetail(ms.ID))
}

// CreateMilestone 新建大事记。
func (h *Handler) CreateMilestone(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		EventDate   string `json:"eventDate"`
		EndDate     string `json:"endDate"`
		Category    string `json:"category"`
		Importance  int    `json:"importance"`
		TagIDs      []uint `json:"tagIds"`
		MediaIDs    []uint `json:"mediaIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	if req.Title == "" || req.EventDate == "" {
		badRequest(c, "请填写标题和日期")
		return
	}
	if req.Category == "" {
		req.Category = "other"
	}
	if req.Importance < 1 || req.Importance > 5 {
		req.Importance = 3
	}
	now := model.NowISO()
	ms := model.Milestone{
		UserID:      cu.ID,
		Title:       req.Title,
		Description: req.Description,
		EventDate:   req.EventDate,
		Category:    req.Category,
		Importance:  req.Importance,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if req.EndDate != "" {
		ms.EndDate = &req.EndDate
	}
	if err := h.db.Create(&ms).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	h.attachTagsAndMedia(cu.ID, "milestone", ms.ID, req.TagIDs, req.MediaIDs)
	c.JSON(200, h.milestoneDetail(ms.ID))
}

// UpdateMilestone 更新大事记。
func (h *Handler) UpdateMilestone(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var ms model.Milestone
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&ms).Error; err != nil {
		notFound(c, "大事记不存在")
		return
	}
	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		EventDate   string  `json:"eventDate"`
		EndDate     *string `json:"endDate"`
		Category    *string `json:"category"`
		Importance  *int    `json:"importance"`
		TagIDs      *[]uint `json:"tagIds"`
		MediaIDs    *[]uint `json:"mediaIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	updates := map[string]interface{}{"updated_at": model.NowISO()}
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.EventDate != "" {
		updates["event_date"] = req.EventDate
	}
	if req.EndDate != nil {
		updates["end_date"] = *req.EndDate
	}
	if req.Category != nil {
		updates["category"] = *req.Category
	}
	if req.Importance != nil {
		updates["importance"] = *req.Importance
	}
	if err := h.db.Model(&model.Milestone{}).Where("id = ? AND user_id = ?", id, cu.ID).Updates(updates).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	if req.TagIDs != nil {
		h.db.Where("milestone_id = ?", id).Delete(&model.MilestoneTag{})
		for _, tid := range *req.TagIDs {
			h.db.Create(&model.MilestoneTag{MilestoneID: uint(id), TagID: tid})
		}
	}
	if req.MediaIDs != nil {
		h.db.Model(&model.Media{}).Where("user_id = ? AND entity_type = 'milestone' AND entity_id = ?", cu.ID, id).
			Updates(map[string]interface{}{"entity_type": nil, "entity_id": nil})
		for _, mid := range *req.MediaIDs {
			h.db.Model(&model.Media{}).Where("id = ? AND user_id = ?", mid, cu.ID).
				Updates(map[string]interface{}{"entity_type": "milestone", "entity_id": id})
		}
	}
	c.JSON(200, h.milestoneDetail(uint(id)))
}

// DeleteMilestone 删除大事记。
func (h *Handler) DeleteMilestone(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var ms model.Milestone
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&ms).Error; err != nil {
		notFound(c, "大事记不存在")
		return
	}
	h.db.Where("milestone_id = ?", id).Delete(&model.MilestoneTag{})
	// 引用该大事记的规划解除关联
	h.db.Model(&model.Plan{}).Where("linked_milestone_id = ?", id).Update("linked_milestone_id", nil)
	h.db.Model(&model.Idea{}).Where("linked_milestone_id = ?", id).Update("linked_milestone_id", nil)
	h.db.Model(&model.Media{}).Where("user_id = ? AND entity_type = 'milestone' AND entity_id = ?", cu.ID, id).
		Updates(map[string]interface{}{"entity_type": nil, "entity_id": nil})
	h.db.Delete(&ms)
	c.JSON(200, gin.H{"success": true})
}
