package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

func planJSON(p *model.Plan) gin.H {
	return gin.H{
		"id":                p.ID,
		"userId":            p.UserID,
		"title":             p.Title,
		"description":       p.Description,
		"targetDate":        p.TargetDate,
		"startDate":         p.StartDate,
		"status":            p.Status,
		"priority":          p.Priority,
		"linkedMilestoneId": p.LinkedMilestoneID,
		"createdAt":         p.CreatedAt,
		"updatedAt":         p.UpdatedAt,
	}
}

// ListPlans 规划列表。
func (h *Handler) ListPlans(c *gin.Context) {
	cu := currentUser(c)
	var all []model.Plan
	if err := h.db.Where("user_id = ?", cu.ID).Order("created_at DESC").Find(&all).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	items := make([]gin.H, 0, len(all))
	for i := range all {
		items = append(items, planJSON(&all[i]))
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

// GetPlan 规划详情。
func (h *Handler) GetPlan(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var p model.Plan
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&p).Error; err != nil {
		notFound(c, "规划不存在")
		return
	}
	c.JSON(200, planJSON(&p))
}

// CreatePlan 新建规划。
func (h *Handler) CreatePlan(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		Title             string `json:"title"`
		Description       string `json:"description"`
		TargetDate        string `json:"targetDate"`
		StartDate         string `json:"startDate"`
		Status            string `json:"status"`
		Priority          string `json:"priority"`
		LinkedMilestoneID *uint  `json:"linkedMilestoneId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	if req.Title == "" {
		badRequest(c, "请填写规划标题")
		return
	}
	if req.Status == "" {
		req.Status = "not_started"
	}
	if req.Priority == "" {
		req.Priority = "medium"
	}
	now := model.NowISO()
	p := model.Plan{
		UserID:            cu.ID,
		Title:             req.Title,
		Description:       req.Description,
		Status:            req.Status,
		Priority:          req.Priority,
		LinkedMilestoneID: req.LinkedMilestoneID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if req.TargetDate != "" {
		p.TargetDate = &req.TargetDate
	}
	if req.StartDate != "" {
		p.StartDate = &req.StartDate
	}
	if err := h.db.Create(&p).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	c.JSON(200, planJSON(&p))
}

// UpdatePlan 更新规划。
func (h *Handler) UpdatePlan(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var p model.Plan
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&p).Error; err != nil {
		notFound(c, "规划不存在")
		return
	}
	var req struct {
		Title             *string `json:"title"`
		Description       *string `json:"description"`
		TargetDate        *string `json:"targetDate"`
		StartDate         *string `json:"startDate"`
		Status            *string `json:"status"`
		Priority          *string `json:"priority"`
		LinkedMilestoneID *uint   `json:"linkedMilestoneId"`
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
	if req.TargetDate != nil {
		updates["target_date"] = *req.TargetDate
	}
	if req.StartDate != nil {
		updates["start_date"] = *req.StartDate
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.LinkedMilestoneID != nil {
		updates["linked_milestone_id"] = *req.LinkedMilestoneID
	}
	if err := h.db.Model(&model.Plan{}).Where("id = ? AND user_id = ?", id, cu.ID).Updates(updates).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	h.db.First(&p, id)
	c.JSON(200, planJSON(&p))
}

// DeletePlan 删除规划。
func (h *Handler) DeletePlan(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var p model.Plan
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&p).Error; err != nil {
		notFound(c, "规划不存在")
		return
	}
	h.db.Model(&model.Idea{}).Where("linked_plan_id = ?", id).Update("linked_plan_id", nil)
	h.db.Delete(&p)
	c.JSON(200, gin.H{"success": true})
}
