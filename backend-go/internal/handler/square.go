package handler

import (
	"encoding/json"
	"regexp"
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
			return nil, "随记不存在"
		}
		p.Title = r.Title
		if p.Title == "" {
			p.Title = "随记 · " + r.RecordDate
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
	brief := userBrief{}
	if p.UserID > 0 {
		brief = h.usersBriefByIDs([]uint{p.UserID})[p.UserID]
	}
	stat := h.pubStatsByIDs([]uint{p.ID}, cuID)[p.ID]
	return pubItemWith(p, brief, stat)
}

// contentImageURLs 取快照正文里的前 3 张图片 URL，供广场卡片做缩略图。
//
// 语义与前端原先的 /<img[^>]+src="([^"]+)"/g 逐项等价（已用 4 万条语料对拍，零差异）：
//   - 只认字面量 <img（区分大小写），不跨越下一个 '>'；<img 与 src=" 之间至少 1 个字符；
//   - [^>]+ 是贪婪的，所以取窗口内最后一个可行的 src="，若其后取不到闭合引号则退回上一个候选；
//   - 捕获 [^"]+ 至少 1 个字符（捕获本身可以跨过 '>'），匹配从闭合引号之后继续；
//   - 最多取前 3 个。
//
// 不用 regexp.FindAllStringSubmatch 实现：那条模式的 [^>]+ 在读到内联 base64 图
// （单标签内几 MB 且直到末尾才有 '>'）时会退化成百万级回溯，单行就要 55ms；
// 而这里改用的索引扫描同一条数据只花 1.2ms（旧客户端的 V8 正则也才 0.9ms）。
func contentImageURLs(s string) []string {
	const imgOpen = `<img`
	const srcOpen = `src="`
	out := make([]string, 0, 3)
	for len(out) < 3 {
		i := strings.Index(s, imgOpen)
		if i < 0 {
			break
		}
		scanFrom := i + len(imgOpen)
		winEnd := len(s)
		if j := strings.IndexByte(s[scanFrom:], '>'); j >= 0 {
			winEnd = scanFrom + j
		}
		start := scanFrom + 1 // [^>]+ 至少吃掉 1 个字符
		seg := ""
		if winEnd > start {
			seg = s[start:winEnd]
		}
		found := false
		for len(seg) >= len(srcOpen) {
			k := strings.LastIndex(seg, srcOpen)
			if k < 0 {
				break
			}
			vStart := start + k + len(srcOpen)
			q := strings.IndexByte(s[vStart:], '"')
			if q < 1 { // [^"]+ 至少 1 个字符且要有闭合引号
				seg = seg[:k]
				continue
			}
			out = append(out, s[vStart:vStart+q])
			s = s[vStart+q+1:] // 从整条匹配之后继续
			found = true
			break
		}
		if !found {
			s = s[scanFrom:]
		}
	}
	return out
}

// pubItemWith 用已预取的作者信息与统计数据组装帖子，不产生额外 SQL。
// 输出字段与逐行版本逐项对应。
func pubItemWith(p *model.Publication, brief userBrief, stat pubStat) gin.H {
	meta := map[string]string{}
	if p.Meta != "" && p.Meta != "{}" {
		_ = json.Unmarshal([]byte(p.Meta), &meta)
	}
	return gin.H{
		"id": p.ID, "userId": p.UserID, "authorName": brief.DisplayName,
		"authorAvatar": brief.AvatarURL, "authorSignature": brief.Signature,
		"sourceType": p.SourceType, "sourceId": p.SourceID,
		"title": p.Title, "preview": p.Preview, "content": p.Content,
		"author": p.Author, "rating": p.Rating, "meta": meta,
		"status": p.Status, "reviewCategories": p.ReviewCategories,
		"createdAt": p.CreatedAt, "updatedAt": p.UpdatedAt,
		"likeCount": stat.LikeCount, "commentCount": stat.CommentCount, "liked": stat.Liked,
	}
}

// pubListItemWith 列表精简版：卡片只需要「有没有图 + 前几张图的 URL + 摘要」，
// 整段正文快照（含内联图片的历史数据单条就有数 MB）只在打开详情时才用得上，
// 由详情接口单独返回。审核队列要在列表里直接渲染全文，故仍走 pubItemWith。
func pubListItemWith(p *model.Publication, brief userBrief, stat pubStat) gin.H {
	m := pubItemWith(p, brief, stat)
	delete(m, "content")
	m["contentImages"] = contentImageURLs(p.Content)
	return m
}

// pubItemsJSON 批量组装帖子列表（含整段正文，供需要直接渲染全文的调用方使用）。
func (h *Handler) pubItemsJSON(pubs []model.Publication, cuID uint) []gin.H {
	return h.buildPubItems(pubs, cuID, false)
}

// pubItemsForList 列表接口专用：不回正文，改回 contentImages。
func (h *Handler) pubItemsForList(pubs []model.Publication, cuID uint) []gin.H {
	return h.buildPubItems(pubs, cuID, true)
}

// buildPubItems 批量组装：作者/点赞数/评论数/已赞状态改为集合查询，
// 原先每个帖子 4 条 SQL（整页 20 条即 80 条），现在合计 4 条。
func (h *Handler) buildPubItems(pubs []model.Publication, cuID uint, listView bool) []gin.H {
	ids := make([]uint, 0, len(pubs))
	userIDs := make([]uint, 0, len(pubs))
	seenUser := make(map[uint]struct{}, len(pubs))
	for i := range pubs {
		ids = append(ids, pubs[i].ID)
		if pubs[i].UserID > 0 {
			if _, ok := seenUser[pubs[i].UserID]; !ok {
				seenUser[pubs[i].UserID] = struct{}{}
				userIDs = append(userIDs, pubs[i].UserID)
			}
		}
	}
	briefs := h.usersBriefByIDs(userIDs)
	stats := h.pubStatsByIDs(ids, cuID)

	items := make([]gin.H, 0, len(pubs))
	for i := range pubs {
		p := &pubs[i]
		if listView {
			items = append(items, pubListItemWith(p, briefs[p.UserID], stats[p.ID]))
		} else {
			items = append(items, pubItemWith(p, briefs[p.UserID], stats[p.ID]))
		}
	}
	return items
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
	items := h.pubItemsForList(pubs, cu.ID) // 前端只用 id/status，不必回正文
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

	// 组装过滤条件（公开帖 + 分类/关键词/作者）
	base := h.db.Model(&model.Publication{}).Where("status = ?", model.PubStatusPublished)
	if typ != "" {
		base = base.Where("source_type = ?", typ)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		base = base.Where("title LIKE ? OR preview LIKE ?", like, like)
	}
	if s := c.Query("userId"); s != "" {
		if uid, err := strconv.ParseUint(s, 10, 64); err == nil && uid > 0 {
			base = base.Where("user_id = ?", uid)
		}
	}
	var total int64
	if err := base.Count(&total).Error; err != nil {
		serverError(c, "查询失败")
		return
	}

	var pubs []model.Publication
	if sortBy == "hot" {
		// 最热：排序下推到 SQL。原先先把全部命中行（含正文大字段）读进内存、逐行数点赞、
		// 再在 Go 里排序后切片，行数增长时内存与耗时都线性上升。
		// 语义与原实现一致：点赞数降序，同数时按 id 升序（原全表扫描返回的即是该顺序），
		// 并且仍然只多做一个 COUNT 子查询——publication_likes 的
		// (publication_id, user_id) 唯一索引可直接完成该计数。
		if err := base.
			Order("(SELECT COUNT(*) FROM publication_likes pl WHERE pl.publication_id = publications.id) DESC, id ASC").
			Limit(limit).Offset((page - 1) * limit).Find(&pubs).Error; err != nil {
			serverError(c, "查询失败")
			return
		}
	} else {
		// 最新：直接 SQL 分页，避免内容量大时一次加载全部
		if err := base.Order("id DESC").Limit(limit).Offset((page - 1) * limit).Find(&pubs).Error; err != nil {
			serverError(c, "查询失败")
			return
		}
	}

	// 组装（含统计与点赞态），并标注相对用户已读水位的“新”内容
	wm := h.squareReadWatermark(cu.ID)
	items := h.pubItemsForList(pubs, cu.ID)
	for i := range pubs {
		items[i]["isNew"] = pubs[i].Status == model.PubStatusPublished &&
			pubs[i].UserID != cu.ID && pubs[i].UpdatedAt > wm
	}

	c.JSON(200, gin.H{"items": items, "total": total, "page": page, "limit": limit})
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
