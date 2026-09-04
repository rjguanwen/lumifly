package handler

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

var tagRe = regexp.MustCompile(`<[^>]*>`)

func stripHTML(s string) string {
	s = tagRe.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return strings.Join(strings.Fields(s), " ")
}

func clip(s string, n int) string {
	rs := []rune(s)
	if len(rs) > n {
		return string(rs[:n]) + "…"
	}
	return s
}

// ---------- 发布快照 ----------

func (h *Handler) snapshot(userID uint, sourceType string, sourceID uint) (*model.Publication, string) {
	now := model.NowISO()
	p := &model.Publication{
		UserID: userID, SourceType: sourceType, SourceID: sourceID,
		Rating: 0, Meta: "{}", CreatedAt: now, UpdatedAt: now,
	}
	switch sourceType {
	case model.PubRecord:
		var r model.Record
		if err := h.db.Where("id = ? AND user_id = ?", sourceID, userID).First(&r).Error; err != nil {
			return nil, "日记不存在"
		}
		p.Title = r.Title
		if p.Title == "" {
			p.Title = "日记 · " + r.RecordDate
		}
		p.Content = r.Content
	case model.PubMilestone:
		var m model.Milestone
		if err := h.db.Where("id = ? AND user_id = ?", sourceID, userID).First(&m).Error; err != nil {
			return nil, "大事记不存在"
		}
		p.Title = m.Title
		p.Content = m.Description
		meta, _ := json.Marshal(map[string]string{
			"eventDate": m.EventDate, "category": m.Category, "importance": strconv.Itoa(m.Importance),
		})
		p.Meta = string(meta)
	case model.PubIdea:
		var i model.Idea
		if err := h.db.Where("id = ? AND user_id = ?", sourceID, userID).First(&i).Error; err != nil {
			return nil, "灵感不存在"
		}
		p.Title = i.Title
		p.Content = i.Content
	case model.PubBook:
		var b model.Book
		if err := h.db.Where("id = ? AND user_id = ?", sourceID, userID).First(&b).Error; err != nil {
			return nil, "书籍不存在"
		}
		p.Title = b.Name
		if strings.TrimSpace(b.Review) != "" {
			p.Content = b.Review
		} else if strings.TrimSpace(b.Recommend) != "" {
			p.Content = escapeHTML(b.Recommend)
		}
		p.Author = b.Author
		p.Rating = b.Rating
		meta, _ := json.Marshal(map[string]string{
			"domain": b.Domain, "readYear": b.ReadYear, "recommend": b.Recommend, "status": b.Status,
		})
		p.Meta = string(meta)
	default:
		return nil, "不支持的内容类型"
	}
	p.Preview = clip(stripHTML(p.Content), 300)
	return p, ""
}

// ---------- 序列化 ----------

// pubItemJSON 组装单个帖子（含作者名/统计/是否已赞）。
func (h *Handler) pubItemJSON(p *model.Publication, cuID uint) gin.H {
	var authorName, authorAvatar, authorSignature string
	if p.UserID > 0 {
		var u model.User
		if err := h.db.Select("display_name, avatar_url, signature").First(&u, p.UserID).Error; err == nil {
			authorName = u.DisplayName
			authorSignature = u.Signature
			if u.AvatarURL != nil {
				authorAvatar = *u.AvatarURL
			}
		}
	}
	var likeCount, commentCount int64
	h.db.Model(&model.PublicationLike{}).Where("publication_id = ?", p.ID).Count(&likeCount)
	h.db.Model(&model.PublicationComment{}).Where("publication_id = ?", p.ID).Count(&commentCount)

	meta := map[string]string{}
	if p.Meta != "" && p.Meta != "{}" {
		_ = json.Unmarshal([]byte(p.Meta), &meta)
	}
	liked := false
	if cuID > 0 {
		var cnt int64
		h.db.Model(&model.PublicationLike{}).Where("publication_id = ? AND user_id = ?", p.ID, cuID).Count(&cnt)
		liked = cnt > 0
	}
	return gin.H{
		"id": p.ID, "userId": p.UserID, "authorName": authorName,
		"authorAvatar": authorAvatar, "authorSignature": authorSignature,
		"sourceType": p.SourceType, "sourceId": p.SourceID,
		"title": p.Title, "preview": p.Preview, "content": p.Content,
		"author": p.Author, "rating": p.Rating, "meta": meta,
		"createdAt": p.CreatedAt, "updatedAt": p.UpdatedAt,
		"likeCount": likeCount, "commentCount": commentCount, "liked": liked,
	}
}

// ---------- 接口 ----------

// Publish 将个人内容发布到广场。
func (h *Handler) Publish(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		SourceType string `json:"sourceType"`
		SourceID   uint   `json:"sourceId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.SourceID == 0 {
		badRequest(c, "请求参数有误")
		return
	}
	var existed model.Publication
	if err := h.db.Where("user_id = ? AND source_type = ? AND source_id = ?", cu.ID, req.SourceType, req.SourceID).
		First(&existed).Error; err == nil {
		c.JSON(200, h.pubItemJSON(&existed, cu.ID))
		return
	}
	pub, errMsg := h.snapshot(cu.ID, req.SourceType, req.SourceID)
	if errMsg != "" {
		badRequest(c, errMsg)
		return
	}
	if err := h.db.Create(pub).Error; err != nil {
		serverError(c, "发布失败")
		return
	}
	c.JSON(200, h.pubItemJSON(pub, cu.ID))
}

// Unpublish 撤回自己发布的帖子。
func (h *Handler) Unpublish(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var pub model.Publication
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&pub).Error; err != nil {
		notFound(c, "帖子不存在")
		return
	}
	h.db.Where("publication_id = ?", pub.ID).Delete(&model.PublicationLike{})
	h.db.Where("publication_id = ?", pub.ID).Delete(&model.PublicationComment{})
	h.db.Delete(&pub)
	c.JSON(200, gin.H{"success": true})
}

// Mine 我发布的帖子（按来源绑定状态）。
func (h *Handler) Mine(c *gin.Context) {
	cu := currentUser(c)
	var pubs []model.Publication
	if err := h.db.Where("user_id = ?", cu.ID).Order("id DESC").Find(&pubs).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	items := make([]gin.H, 0, len(pubs))
	for i := range pubs {
		items = append(items, h.pubItemJSON(&pubs[i], cu.ID))
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

// ListSquare 广场流（分类/关键词/排序/分页）。
func (h *Handler) ListSquare(c *gin.Context) {
	cu := currentUser(c)
	typ := c.Query("type")
	keyword := strings.TrimSpace(c.Query("keyword"))
	sortBy := c.DefaultQuery("sort", "latest")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 60 {
		limit = 20
	}

	var pubs []model.Publication
	q := h.db.Model(&model.Publication{})
	if typ != "" {
		q = q.Where("source_type = ?", typ)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("title LIKE ? OR preview LIKE ?", like, like)
	}
	if err := q.Find(&pubs).Error; err != nil {
		serverError(c, "查询失败")
		return
	}

	// 组装（含统计与点赞态）
	items := make([]gin.H, 0, len(pubs))
	for i := range pubs {
		items = append(items, h.pubItemJSON(&pubs[i], cu.ID))
	}
	if sortBy == "hot" {
		sort.SliceStable(items, func(i, j int) bool {
			return items[i]["likeCount"].(int64) > items[j]["likeCount"].(int64)
		})
	} else {
		// latest: 按 id 降序（已由查询 id 乱序？需显式）
		sort.SliceStable(items, func(i, j int) bool {
			return items[i]["id"].(uint) > items[j]["id"].(uint)
		})
	}

	start := (page - 1) * limit
	end := start + limit
	if start > len(items) {
		start = len(items)
	}
	if end > len(items) {
		end = len(items)
	}
	c.JSON(200, gin.H{"items": items[start:end], "total": len(items), "page": page, "limit": limit})
}

// GetSquarePost 帖子详情。
func (h *Handler) GetSquarePost(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var pub model.Publication
	if err := h.db.First(&pub, id).Error; err != nil {
		notFound(c, "帖子不存在")
		return
	}
	c.JSON(200, h.pubItemJSON(&pub, cu.ID))
}

// ToggleLike 点赞/取消点赞。
func (h *Handler) ToggleLike(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var pub model.Publication
	if err := h.db.First(&pub, id).Error; err != nil {
		notFound(c, "帖子不存在")
		return
	}
	var cnt int64
	h.db.Model(&model.PublicationLike{}).Where("publication_id = ? AND user_id = ?", id, cu.ID).Count(&cnt)
	liked := true
	if cnt > 0 {
		h.db.Where("publication_id = ? AND user_id = ?", id, cu.ID).Delete(&model.PublicationLike{})
		liked = false
	} else {
		h.db.Create(&model.PublicationLike{PublicationID: pub.ID, UserID: cu.ID, CreatedAt: model.NowISO()})
	}
	var likeCount int64
	h.db.Model(&model.PublicationLike{}).Where("publication_id = ?", pub.ID).Count(&likeCount)
	c.JSON(200, gin.H{"liked": liked, "likeCount": likeCount})
}

// AddComment 发表评论。
func (h *Handler) AddComment(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var pub model.Publication
	if err := h.db.First(&pub, id).Error; err != nil {
		notFound(c, "帖子不存在")
		return
	}
	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		badRequest(c, "评论内容不能为空")
		return
	}
	if len([]rune(content)) > 1000 {
		badRequest(c, "评论内容过长（最多 1000 字）")
		return
	}
	cm := model.PublicationComment{
		PublicationID: pub.ID, UserID: cu.ID, Content: content, CreatedAt: model.NowISO(),
	}
	if err := h.db.Create(&cm).Error; err != nil {
		serverError(c, "评论失败")
		return
	}
	c.JSON(200, h.commentJSON(&cm))
}

// commentJSON 评论（含昵称与是否可删）。
func (h *Handler) commentJSON(cm *model.PublicationComment) gin.H {
	var u model.User
	nick, avatar, signature := "", "", ""
	if err := h.db.Select("display_name, avatar_url, signature").First(&u, cm.UserID).Error; err == nil {
		nick = u.DisplayName
		signature = u.Signature
		if u.AvatarURL != nil {
			avatar = *u.AvatarURL
		}
	}
	return gin.H{
		"id": cm.ID, "publicationId": cm.PublicationID, "userId": cm.UserID,
		"authorName": nick, "authorAvatar": avatar, "authorSignature": signature,
		"content": cm.Content, "createdAt": cm.CreatedAt,
	}
}

// ListComments 帖子评论列表。
func (h *Handler) ListComments(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var pub model.Publication
	if err := h.db.First(&pub, id).Error; err != nil {
		notFound(c, "帖子不存在")
		return
	}
	var cms []model.PublicationComment
	if err := h.db.Where("publication_id = ?", pub.ID).Order("id ASC").Find(&cms).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	items := make([]gin.H, 0, len(cms))
	for i := range cms {
		item := h.commentJSON(&cms[i])
		item["canDelete"] = cms[i].UserID == cu.ID
		items = append(items, item)
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

// DeleteComment 删除评论（仅本人）。
func (h *Handler) DeleteComment(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var cm model.PublicationComment
	if err := h.db.Where("id = ? AND user_id = ?", id, cu.ID).First(&cm).Error; err != nil {
		notFound(c, "评论不存在或无权删除")
		return
	}
	h.db.Delete(&cm)
	c.JSON(200, gin.H{"success": true})
}
