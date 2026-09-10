package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lumiflybackend/internal/model"
)

// bookJSON 书籍详情（含书评与标签）。详情/新建/更新接口使用。
func (h *Handler) bookJSON(b *model.Book) gin.H {
	return gin.H{
		"id":        b.ID,
		"userId":    b.UserID,
		"name":      b.Name,
		"author":    b.Author,
		"domain":    b.Domain,
		"status":    b.Status,
		"rating":    b.Rating,
		"readYear":  b.ReadYear,
		"recommend": b.Recommend,
		"review":    b.Review,
		"createdAt": b.CreatedAt,
		"updatedAt": b.UpdatedAt,
		"tags":      h.tagsByIDs(bTagsIDs(h.db, b.ID)),
	}
}

// bTagsIDs 查询一本书的标签关联 id 列表。
func bTagsIDs(db *gorm.DB, bookID uint) []uint {
	var links []model.BookTag
	db.Where("book_id = ?", bookID).Find(&links)
	ids := make([]uint, 0, len(links))
	for _, l := range links {
		ids = append(ids, l.TagID)
	}
	return ids
}

// bookCardJSON 书籍列表卡片（不含书评大字段，减少列表响应体积）。
func bookCardJSON(b *model.Book, tags []gin.H) gin.H {
	return gin.H{
		"id":        b.ID,
		"userId":    b.UserID,
		"name":      b.Name,
		"author":    b.Author,
		"domain":    b.Domain,
		"status":    b.Status,
		"rating":    b.Rating,
		"readYear":  b.ReadYear,
		"recommend": b.Recommend,
		"createdAt": b.CreatedAt,
		"updatedAt": b.UpdatedAt,
		"tags":      tags,
	}
}

// bookTagsByIDs 批量加载多本书的标签（book_id -> tags JSON），
// 用 2 条 SQL 替代逐本查询，避免列表 N+1 慢查询。
func (h *Handler) bookTagsByIDs(bookIDs []uint) map[uint][]gin.H {
	out := make(map[uint][]gin.H, len(bookIDs))
	if len(bookIDs) == 0 {
		return out
	}
	var links []model.BookTag
	h.db.Where("book_id IN ?", bookIDs).Find(&links)
	if len(links) == 0 {
		return out
	}
	tagIDSet := make(map[uint]struct{}, len(links))
	order := make([]uint, 0, len(links))
	for _, l := range links {
		if _, ok := tagIDSet[l.TagID]; !ok {
			tagIDSet[l.TagID] = struct{}{}
			order = append(order, l.TagID)
		}
	}
	var tags []model.Tag
	h.db.Where("id IN ?", order).Find(&tags)
	tagJSON := make(map[uint]gin.H, len(tags))
	for i := range tags {
		t := &tags[i]
		tagJSON[t.ID] = gin.H{
			"id": t.ID, "userId": t.UserID, "name": t.Name,
			"color": t.Color, "createdAt": t.CreatedAt,
		}
	}
	for _, l := range links {
		if j, ok := tagJSON[l.TagID]; ok {
			out[l.BookID] = append(out[l.BookID], j)
		}
	}
	return out
}

// ListBooks 书籍列表（支持 status/domain/tag_id/keyword 过滤 + 分页，供前端滚动加载）。
func (h *Handler) ListBooks(c *gin.Context) {
	cu := currentUser(c)
	status := c.Query("status")
	domain := c.Query("domain")
	tagID := c.Query("tag_id")
	keyword := strings.TrimSpace(c.Query("keyword"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	q := h.db.Model(&model.Book{}).Where("user_id = ?", cu.ID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	if domain != "" {
		q = q.Where("domain = ?", domain)
	}
	if tagID != "" {
		if id, err := strconv.Atoi(tagID); err == nil {
			q = q.Where("id IN (SELECT book_id FROM book_tags WHERE tag_id = ?)", id)
		}
	}
	if keyword != "" {
		q = q.Where("name LIKE ? OR author LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		serverError(c, "查询失败")
		return
	}

	var all []model.Book
	if err := q.Order("id DESC").Offset((page - 1) * limit).Limit(limit).Find(&all).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	// 批量加载标签，避免逐本查询
	bookIDs := make([]uint, 0, len(all))
	for i := range all {
		bookIDs = append(bookIDs, all[i].ID)
	}
	tagsMap := h.bookTagsByIDs(bookIDs)
	items := make([]gin.H, 0, len(all))
	for i := range all {
		tgs := tagsMap[all[i].ID]
		if tgs == nil {
			tgs = []gin.H{}
		}
		items = append(items, bookCardJSON(&all[i], tgs))
	}
	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "limit": limit})
}

// GetBook 书籍详情。
func (h *Handler) GetBook(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var b model.Book
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&b).Error; err != nil {
		notFound(c, "书籍不存在")
		return
	}
	c.JSON(200, h.bookJSON(&b))
}

// CreateBook 新建书籍。
func (h *Handler) CreateBook(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		Name      string `json:"name"`
		Author    string `json:"author"`
		Domain    string `json:"domain"`
		Status    string `json:"status"`
		Rating    int    `json:"rating"`
		ReadYear  string `json:"readYear"`
		Recommend string `json:"recommend"`
		Review    string `json:"review"`
		TagIDs    []uint `json:"tagIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		badRequest(c, "请填写书名")
		return
	}
	if req.Status == "" {
		req.Status = model.BookStatusNotStarted
	}
	if req.Rating < 0 || req.Rating > 5 {
		req.Rating = 0
	}
	now := model.NowISO()
	b := model.Book{
		UserID:    cu.ID,
		Name:      name,
		Author:    strings.TrimSpace(req.Author),
		Domain:    strings.TrimSpace(req.Domain),
		Status:    req.Status,
		Rating:    req.Rating,
		ReadYear:  strings.TrimSpace(req.ReadYear),
		Recommend: req.Recommend,
		Review:    req.Review,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.db.Create(&b).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	h.createTagLinks("book", b.ID, req.TagIDs)
	c.JSON(200, h.bookJSON(&b))
}

// UpdateBook 更新书籍（部分更新，tagIds 显式传才整体替换）。
func (h *Handler) UpdateBook(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var b model.Book
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&b).Error; err != nil {
		notFound(c, "书籍不存在")
		return
	}
	var req struct {
		Name      *string `json:"name"`
		Author    *string `json:"author"`
		Domain    *string `json:"domain"`
		Status    *string `json:"status"`
		Rating    *int    `json:"rating"`
		ReadYear  *string `json:"readYear"`
		Recommend *string `json:"recommend"`
		Review    *string `json:"review"`
		TagIDs    *[]uint `json:"tagIds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	updates := map[string]interface{}{"updated_at": model.NowISO()}
	if req.Name != nil {
		n := strings.TrimSpace(*req.Name)
		if n == "" {
			badRequest(c, "书名不能为空")
			return
		}
		updates["name"] = n
	}
	if req.Author != nil {
		updates["author"] = strings.TrimSpace(*req.Author)
	}
	if req.Domain != nil {
		updates["domain"] = strings.TrimSpace(*req.Domain)
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.Rating != nil {
		if *req.Rating < 0 || *req.Rating > 5 {
			badRequest(c, "星级需在 0-5 之间")
			return
		}
		updates["rating"] = *req.Rating
	}
	if req.ReadYear != nil {
		updates["read_year"] = strings.TrimSpace(*req.ReadYear)
	}
	if req.Recommend != nil {
		updates["recommend"] = *req.Recommend
	}
	if req.Review != nil {
		updates["review"] = *req.Review
	}
	if len(updates) > 1 {
		if err := h.db.Model(&model.Book{}).Where("id = ? AND user_id = ?", id, cu.ID).Updates(updates).Error; err != nil {
			serverError(c, "保存失败")
			return
		}
	}
	if req.TagIDs != nil {
		h.replaceTagLinks("book", uint(id), *req.TagIDs)
	}
	h.db.First(&b, id)
	c.JSON(200, h.bookJSON(&b))
}

// DeleteBook 删除书籍（级联清理标签关联，标签库保留）。
func (h *Handler) DeleteBook(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var b model.Book
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&b).Error; err != nil {
		notFound(c, "书籍不存在")
		return
	}
	h.db.Where("book_id = ?", id).Delete(&model.BookTag{})
	h.db.Delete(&b)
	c.JSON(200, gin.H{"success": true})
}
