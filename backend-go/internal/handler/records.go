package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// mediaJSON 输出媒体列表（含访问 URL）。
func mediaJSON(list []model.Media) []gin.H {
	res := make([]gin.H, 0, len(list))
	for _, m := range list {
		res = append(res, gin.H{
			"id":         m.ID,
			"userId":     m.UserID,
			"filePath":   m.FilePath,
			"fileName":   m.FileName,
			"mimeType":   m.MimeType,
			"fileSize":   m.FileSize,
			"entityType": m.EntityType,
			"entityId":   m.EntityID,
			"createdAt":  m.CreatedAt,
			"url":        "/api/uploads/" + m.FilePath,
		})
	}
	return res
}

// tagsByIDs 按 id 列表查标签。
func (h *Handler) tagsByIDs(ids []uint) []gin.H {
	res := make([]gin.H, 0, len(ids))
	if len(ids) == 0 {
		return res
	}
	var tags []model.Tag
	h.db.Where("id IN ?", ids).Find(&tags)
	for _, t := range tags {
		res = append(res, gin.H{
			"id": t.ID, "userId": t.UserID, "name": t.Name,
			"color": t.Color, "createdAt": t.CreatedAt,
		})
	}
	return res
}

// mediaByEntity 查某实体关联的媒体。
func (h *Handler) mediaByEntity(entityType string, entityID uint) []gin.H {
	var list []model.Media
	h.db.Where("entity_type = ? AND entity_id = ?", entityType, entityID).Find(&list)
	return mediaJSON(list)
}

// recordDetail 记录详情（含标签与媒体）。
func (h *Handler) recordDetail(recordID uint) gin.H {
	var rec model.Record
	if err := h.db.First(&rec, recordID).Error; err != nil {
		return nil
	}
	var links []model.RecordTag
	h.db.Where("record_id = ?", recordID).Find(&links)
	ids := make([]uint, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.TagID)
	}
	return gin.H{
		"id":         rec.ID,
		"userId":     rec.UserID,
		"title":      rec.Title,
		"content":    rec.Content,
		"recordDate": rec.RecordDate,
		"mood":       rec.Mood,
		"weather":    rec.Weather,
		"createdAt":  rec.CreatedAt,
		"updatedAt":  rec.UpdatedAt,
		"tags":       h.tagsByIDs(ids),
		"media":      h.mediaByEntity("record", rec.ID),
	}
}

// ListRecords 记录列表（支持 from/to/mood 过滤 + 分页，供前端滚动加载）。
func (h *Handler) ListRecords(c *gin.Context) {
	cu := currentUser(c)
	from := c.Query("from")
	to := c.Query("to")
	mood := c.Query("mood")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	q := h.db.Model(&model.Record{}).Where("user_id = ?", cu.ID)
	if from != "" {
		q = q.Where("record_date >= ?", from)
	}
	if to != "" {
		q = q.Where("record_date <= ?", to)
	}
	if mood != "" {
		q = q.Where("mood = ?", mood)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		serverError(c, "查询失败")
		return
	}

	var paged []model.Record
	if err := q.Order("record_date DESC, id DESC").Offset((page - 1) * limit).Limit(limit).Find(&paged).Error; err != nil {
		serverError(c, "查询失败")
		return
	}

	items := make([]gin.H, 0, len(paged))
	for _, r := range paged {
		items = append(items, h.recordDetail(r.ID))
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

// GetRecord 记录详情。
func (h *Handler) GetRecord(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var rec model.Record
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&rec).Error; err != nil {
		notFound(c, "记录不存在")
		return
	}
	c.JSON(200, h.recordDetail(rec.ID))
}

// CreateRecord 新建记录。
func (h *Handler) CreateRecord(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		Title      string `json:"title"`
		Content    string `json:"content"`
		RecordDate string `json:"recordDate"`
		Mood       string `json:"mood"`
		Weather    string `json:"weather"`
		TagIDs     []uint `json:"tagIds"`
		MediaIDs   []uint `json:"mediaIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	if req.RecordDate == "" {
		badRequest(c, "请选择记录日期")
		return
	}
	now := model.NowISO()
	rec := model.Record{
		UserID:     cu.ID,
		Title:      req.Title,
		Content:    req.Content,
		RecordDate: req.RecordDate,
		Mood:       req.Mood,
		Weather:    req.Weather,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := h.db.Create(&rec).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	h.attachTagsAndMedia(cu.ID, "record", rec.ID, req.TagIDs, req.MediaIDs)
	c.JSON(200, h.recordDetail(rec.ID))
}

// attachTagsAndMedia 写入标签关联与媒体关联（事务内），供各资源复用。
func (h *Handler) attachTagsAndMedia(userID uint, entityType string, entityID uint, tagIDs, mediaIDs []uint) {
	for _, tid := range tagIDs {
		switch entityType {
		case "record":
			h.db.Create(&model.RecordTag{RecordID: entityID, TagID: tid})
		case "milestone":
			h.db.Create(&model.MilestoneTag{MilestoneID: entityID, TagID: tid})
		case "idea":
			h.db.Create(&model.IdeaTag{IdeaID: entityID, TagID: tid})
		}
	}
	for _, mid := range mediaIDs {
		h.db.Model(&model.Media{}).Where("id = ? AND user_id = ?", mid, userID).
			Updates(map[string]interface{}{"entity_type": entityType, "entity_id": entityID})
	}
}

// UpdateRecord 更新记录。
func (h *Handler) UpdateRecord(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var rec model.Record
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&rec).Error; err != nil {
		notFound(c, "记录不存在")
		return
	}
	var req struct {
		Title      *string `json:"title"`
		Content    *string `json:"content"`
		RecordDate string  `json:"recordDate"`
		Mood       *string `json:"mood"`
		Weather    *string `json:"weather"`
		TagIDs     *[]uint `json:"tagIds"`
		MediaIDs   *[]uint `json:"mediaIds"`
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
	if req.RecordDate != "" {
		updates["record_date"] = req.RecordDate
	}
	if req.Mood != nil {
		updates["mood"] = *req.Mood
	}
	if req.Weather != nil {
		updates["weather"] = *req.Weather
	}
	if err := h.db.Model(&model.Record{}).Where("id = ? AND user_id = ?", id, cu.ID).Updates(updates).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	if req.TagIDs != nil {
		h.db.Where("record_id = ?", id).Delete(&model.RecordTag{})
		for _, tid := range *req.TagIDs {
			h.db.Create(&model.RecordTag{RecordID: uint(id), TagID: tid})
		}
	}
	if req.MediaIDs != nil {
		// 先解除旧媒体关联，再挂载新关联
		h.db.Model(&model.Media{}).Where("user_id = ? AND entity_type = 'record' AND entity_id = ?", cu.ID, id).
			Updates(map[string]interface{}{"entity_type": nil, "entity_id": nil})
		for _, mid := range *req.MediaIDs {
			h.db.Model(&model.Media{}).Where("id = ? AND user_id = ?", mid, cu.ID).
				Updates(map[string]interface{}{"entity_type": "record", "entity_id": id})
		}
	}
	c.JSON(200, h.recordDetail(uint(id)))
}

// DeleteRecord 删除记录（级联清理标签关联；媒体记录保留，解除实体关联）。
func (h *Handler) DeleteRecord(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var rec model.Record
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&rec).Error; err != nil {
		notFound(c, "记录不存在")
		return
	}
	h.db.Where("record_id = ?", id).Delete(&model.RecordTag{})
	h.db.Model(&model.Media{}).Where("user_id = ? AND entity_type = 'record' AND entity_id = ?", cu.ID, id).
		Updates(map[string]interface{}{"entity_type": nil, "entity_id": nil})
	h.db.Delete(&rec)
	c.JSON(200, gin.H{"success": true})
}
