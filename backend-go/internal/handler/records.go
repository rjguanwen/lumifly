package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

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
		res = append(res, tagJSONRow(&t))
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
	return h.recordDetailOf(&rec)
}

// recordDetailOf 用已取到的记录行组装详情，避免再查一次主行。
func (h *Handler) recordDetailOf(rec *model.Record) gin.H {
	var links []model.RecordTag
	h.db.Where("record_id = ?", rec.ID).Find(&links)
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

	// 标签与媒体改为批量加载：原先每行 4~5 条 SQL（limit=100 时单请求近 500 条），
	// 现在固定 3 条，输出与逐行版本完全一致。
	ids := make([]uint, 0, len(paged))
	for i := range paged {
		ids = append(ids, paged[i].ID)
	}
	tagMap := h.tagsByEntities("record_tags", "record_id", ids)
	mediaMap := h.mediaByEntities("record", ids)

	items := make([]gin.H, 0, len(paged))
	for i := range paged {
		r := &paged[i]
		items = append(items, gin.H{
			"id":         r.ID,
			"userId":     r.UserID,
			"title":      r.Title,
			"content":    r.Content,
			"recordDate": r.RecordDate,
			"mood":       r.Mood,
			"weather":    r.Weather,
			"createdAt":  r.CreatedAt,
			"updatedAt":  r.UpdatedAt,
			"tags":       tagMap[r.ID],
			"media":      mediaMap[r.ID],
		})
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
	// 直接用已取到的行组装，避开 recordDetail 内重复的主行查询
	c.JSON(200, h.recordDetailOf(&rec))
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
	c.JSON(200, h.recordDetailOf(&rec))
}

// attachTagsAndMedia 写入标签关联与媒体关联（事务内），供各资源复用。
// 写入方式改为单事务 + 批量语句：原先逐条 Create/Update 每个都是一次独立写事务（各一次 fsync）。
// 只改写法，落库结果与逐条版本一致（错误同样不中断后续语句）。
func (h *Handler) attachTagsAndMedia(userID uint, entityType string, entityID uint, tagIDs, mediaIDs []uint) {
	table, column, ok := tagLinkTable(entityType)
	if !ok && len(mediaIDs) == 0 {
		return
	}
	h.db.Transaction(func(tx *gorm.DB) error {
		if ok {
			insertTagLinks(tx, table, column, entityID, tagIDs)
		}
		if len(mediaIDs) > 0 {
			tx.Model(&model.Media{}).Where("id IN ? AND user_id = ?", mediaIDs, userID).
				Updates(map[string]interface{}{"entity_type": entityType, "entity_id": entityID})
		}
		return nil
	})
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
		h.replaceTagLinks("record", uint(id), *req.TagIDs)
	}
	if req.MediaIDs != nil {
		// 先解除旧媒体关联，再挂载新关联
		h.remountMediaOnUpdate(cu.ID, "record", uint(id), *req.MediaIDs)
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
