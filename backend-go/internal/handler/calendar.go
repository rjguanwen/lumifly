package handler

import (
	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// CalendarSummary 日历汇总：出生信息 + 各实体数据（供人生日历渲染）。
func (h *Handler) CalendarSummary(c *gin.Context) {
	cu := currentUser(c)
	var u model.User
	if err := h.db.First(&u, cu.ID).Error; err != nil {
		notFound(c, "用户不存在")
		return
	}
	if u.BirthDate == nil || *u.BirthDate == "" {
		fail(c, 400, "请先设置出生日期")
		return
	}

	// 记录按日期分组计数（用 struct 接收，避免 Scan 到 map 切片在部分驱动下产生 null 元素）
	type recordCountRow struct {
		RecordDate string `json:"record_date"`
		Count      int64  `json:"count"`
	}
	var records []recordCountRow
	h.db.Model(&model.Record{}).Select("record_date, COUNT(*) AS count").
		Where("user_id = ?", cu.ID).Group("record_date").Scan(&records)

	// 大事记（含媒体，用于日历时段详情）
	var milestones []model.Milestone
	h.db.Where("user_id = ?", cu.ID).Find(&milestones)
	milestoneItems := make([]gin.H, 0, len(milestones))
	for _, m := range milestones {
		msJSON := h.milestoneDetail(m.ID)
		milestoneItems = append(milestoneItems, gin.H{
			"id":          m.ID,
			"title":       m.Title,
			"description": m.Description,
			"category":    m.Category,
			"eventDate":   m.EventDate,
			"importance":  m.Importance,
			"media":       msJSON["media"],
		})
	}

	// 规划
	var plans []model.Plan
	h.db.Where("user_id = ?", cu.ID).Find(&plans)
	planItems := make([]gin.H, 0, len(plans))
	for i := range plans {
		planItems = append(planItems, planJSON(&plans[i]))
	}

	// 灵感（仅 id/created_at 用于密度统计）
	type ideaRow struct {
		ID        uint   `json:"id"`
		CreatedAt string `json:"created_at"`
	}
	var ideas []ideaRow
	h.db.Model(&model.Idea{}).Select("id, created_at").
		Where("user_id = ?", cu.ID).Scan(&ideas)

	c.JSON(200, gin.H{
		"birthDate":        u.BirthDate,
		"expectedLifespan": u.ExpectedLifespan,
		"records":          records,
		"milestones":       milestoneItems,
		"plans":            planItems,
		"ideas":            ideas,
	})
}
