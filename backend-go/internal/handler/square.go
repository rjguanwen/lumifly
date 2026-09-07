package handler

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/middleware"
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

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// textToHTML 将纯文本转义为 HTML（保留换行），用于广场快照。
func textToHTML(s string) string {
	s = escapeHTML(s)
	return strings.ReplaceAll(s, "\n", "<br>")
}

// contentDateISO 将内容自带的日期（YYYY-MM-DD）转为 ISO 时间，作为快照展示时间；
// 传入空值或非日期时返回空字符串（调用方回退为原逻辑）。
func contentDateISO(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 10 && s[4] == '-' && s[7] == '-' {
		return s + "T00:00:00.000Z"
	}
	return ""
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
		if d := contentDateISO(r.RecordDate); d != "" {
			p.CreatedAt = d
		}
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
		if d := contentDateISO(m.EventDate); d != "" {
			p.CreatedAt = d
		}
	case model.PubIdea:
		var i model.Idea
		if err := h.db.Where("id = ? AND user_id = ?", sourceID, userID).First(&i).Error; err != nil {
			return nil, "灵感不存在"
		}
		p.Title = i.Title
		p.Content = i.Content
		if strings.TrimSpace(i.CreatedAt) != "" {
			p.CreatedAt = i.CreatedAt
		}
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
		if strings.TrimSpace(b.CreatedAt) != "" {
			p.CreatedAt = b.CreatedAt
		}
	case model.PubQuick:
		var q model.QuickNote
		if err := h.db.Where("id = ? AND user_id = ?", sourceID, userID).First(&q).Error; err != nil {
			return nil, "速记语录不存在"
		}
		// 语录正文已作为预览全文展示，标题改用日期避免内容重复
		p.Title = "语录 · " + now[:10]
		p.Content = textToHTML(q.Content)
	default:
		return nil, "不支持的内容类型"
	}
	// 附件媒体（图片/视频）追加进快照正文，使广场上也能看到相关图片
	if sourceType == model.PubRecord || sourceType == model.PubMilestone || sourceType == model.PubIdea {
		if media := h.mediaByEntity(sourceType, sourceID); len(media) > 0 {
			p.Content = contentWithMedia(p.Content, media)
		}
	}
	p.Preview = clip(stripHTML(p.Content), 300)
	return p, ""
}

// contentWithMedia 在正文末尾追加附件媒体 HTML（图片/视频）。
func contentWithMedia(html string, media []gin.H) string {
	var b strings.Builder
	b.WriteString(html)
	for _, m := range media {
		mt, _ := m["mimeType"].(string)
		u, _ := m["url"].(string)
		if u == "" {
			continue
		}
		if strings.HasPrefix(mt, "image/") {
			b.WriteString(`<p class="pub-media"><img src="` + u + `" alt="" style="max-width:100%;border-radius:12px;margin:8px 0" /></p>`)
		} else if strings.HasPrefix(mt, "video/") {
			b.WriteString(`<p class="pub-media"><video controls preload="metadata" src="` + u + `" style="max-width:100%;border-radius:12px;margin:8px 0"></video></p>`)
		}
	}
	return b.String()
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
		"status": p.Status, "reviewCategories": p.ReviewCategories,
		"createdAt": p.CreatedAt, "updatedAt": p.UpdatedAt,
		"likeCount": likeCount, "commentCount": commentCount, "liked": liked,
	}
}

// ---------- 接口 ----------

// Publish 将个人内容发布到广场（自动审查，命中敏感内容进入待审）。
// 若该内容已存在且为 published 则直接返回；pending/rejected 状态会按最新源内容重建快照并重新审查。
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
	had := h.db.Where("user_id = ? AND source_type = ? AND source_id = ?", cu.ID, req.SourceType, req.SourceID).
		First(&existed).Error == nil
	if had && existed.Status == model.PubStatusPublished {
		c.JSON(200, h.pubItemJSON(&existed, cu.ID))
		return
	}

	pub, errMsg := h.snapshot(cu.ID, req.SourceType, req.SourceID)
	if errMsg != "" {
		badRequest(c, errMsg)
		return
	}
	// 自动审查：标题 + 纯文本正文
	blocked, cats := h.moderateText(pub.Title + "\n" + stripHTML(pub.Content))
	status := model.PubStatusPublished
	if blocked {
		status = model.PubStatusPending
	}
	pub.Status = status
	pub.ReviewCategories = strings.Join(cats, ",")

	if had {
		// 撤回/驳回后修改源内容重新提交：重建快照并重新审查
		if err := h.db.Model(&model.Publication{}).Where("id = ?", existed.ID).Updates(map[string]interface{}{
			"title": pub.Title, "preview": pub.Preview, "content": pub.Content,
			"author": pub.Author, "rating": pub.Rating, "meta": pub.Meta, "created_at": pub.CreatedAt,
			"status": status, "review_categories": pub.ReviewCategories, "updated_at": model.NowISO(),
		}).Error; err != nil {
			serverError(c, "发布失败")
			return
		}
		var after model.Publication
		h.db.First(&after, existed.ID)
		c.JSON(200, h.pubItemJSON(&after, cu.ID))
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
	q := h.db.Model(&model.Publication{}).Where("status = ?", model.PubStatusPublished)
	if typ != "" {
		q = q.Where("source_type = ?", typ)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("title LIKE ? OR preview LIKE ?", like, like)
	}
	if s := c.Query("userId"); s != "" {
		if uid, err := strconv.ParseUint(s, 10, 64); err == nil && uid > 0 {
			q = q.Where("user_id = ?", uid)
		}
	}
	if err := q.Find(&pubs).Error; err != nil {
		serverError(c, "查询失败")
		return
	}

	// 组装（含统计与点赞态），并标注相对用户已读水位的“新”内容
	wm := h.squareReadWatermark(cu.ID)
	items := make([]gin.H, 0, len(pubs))
	for i := range pubs {
		item := h.pubItemJSON(&pubs[i], cu.ID)
		item["isNew"] = pubs[i].Status == model.PubStatusPublished &&
			pubs[i].UserID != cu.ID && pubs[i].UpdatedAt > wm
		items = append(items, item)
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

// canAccessPub 判断某用户能否查看帖子（公开帖所有人可见；待审/驳回帖仅作者与审核员可见）。
func canAccessPub(p *model.Publication, cu *middleware.UserContext) bool {
	if p.Status == model.PubStatusPublished {
		return true
	}
	if cu.ID == p.UserID {
		return true
	}
	return cu.Role == model.RoleAdmin || cu.Role == model.RoleModerator
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
	if !canAccessPub(&pub, cu) {
		notFound(c, "帖子不存在或未通过审核")
		return
	}
	// 打开公开帖即视为已读（推进水位）
	if pub.Status == model.PubStatusPublished && pub.UserID != cu.ID {
		h.advanceSquareRead(cu.ID, pub.UpdatedAt)
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
	if !canAccessPub(&pub, cu) {
		notFound(c, "帖子不存在或未通过审核")
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
	if !canAccessPub(&pub, cu) {
		notFound(c, "帖子不存在或未通过审核")
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
	if !canAccessPub(&pub, cu) {
		notFound(c, "帖子不存在或未通过审核")
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
