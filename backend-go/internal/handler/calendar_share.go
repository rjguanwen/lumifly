package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// 私密评价单条字数上限。
const calendarCommentMax = 500

// isCalendarShared 判断 ownerID 是否已将自己的日历分享给 viewerID。
func (h *Handler) isCalendarShared(ownerID, viewerID uint) bool {
	var n int64
	h.db.Model(&model.CalendarShare{}).
		Where("user_id = ? AND friend_id = ?", ownerID, viewerID).Count(&n)
	return n > 0
}

// canViewCalendar 能否查看某人生日/日历（本人或已被分享）。
func (h *Handler) canViewCalendar(cuID, ownerID uint) bool {
	return cuID == ownerID || h.isCalendarShared(ownerID, cuID)
}

// MyCalendarShares 我分享给了哪些好友。
func (h *Handler) MyCalendarShares(c *gin.Context) {
	cu := currentUser(c)
	var rows []model.CalendarShare
	h.db.Where("user_id = ?", cu.ID).Order("id DESC").Find(&rows)
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		var u model.User
		if err := h.db.First(&u, r.FriendID).Error; err != nil {
			continue
		}
		item := h.userProfileJSON(u)
		item["sharedAt"] = r.CreatedAt
		items = append(items, item)
	}
	c.JSON(200, gin.H{"items": items})
}

// AddCalendarShare 将人生日历分享给某个好友。
func (h *Handler) AddCalendarShare(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		FriendID uint `json:"friendId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FriendID == 0 {
		badRequest(c, "请选择好友")
		return
	}
	if req.FriendID == cu.ID {
		badRequest(c, "不能分享给自己")
		return
	}
	// 必须是好友（已接受）
	var rel int64
	h.db.Model(&model.Friendship{}).
		Where("status = ? AND ((user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?))",
			model.FriendAccepted, cu.ID, req.FriendID, req.FriendID, cu.ID).Count(&rel)
	if rel == 0 {
		fail(c, 400, "只能分享给好友")
		return
	}
	var n int64
	h.db.Model(&model.CalendarShare{}).Where("user_id = ? AND friend_id = ?", cu.ID, req.FriendID).Count(&n)
	if n > 0 {
		c.JSON(200, gin.H{"success": true, "already": true})
		return
	}
	cs := model.CalendarShare{UserID: cu.ID, FriendID: req.FriendID, CreatedAt: model.NowISO()}
	if err := h.db.Create(&cs).Error; err != nil {
		serverError(c, "操作失败")
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// RemoveCalendarShare 停止向某好友分享日历。
func (h *Handler) RemoveCalendarShare(c *gin.Context) {
	cu := currentUser(c)
	friendID, err := strconv.Atoi(c.Param("friendId"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	h.db.Where("user_id = ? AND friend_id = ?", cu.ID, friendID).Delete(&model.CalendarShare{})
	c.JSON(200, gin.H{"success": true})
}

// ReceivedCalendarShares 我收到哪些好友的人生日历分享。
func (h *Handler) ReceivedCalendarShares(c *gin.Context) {
	cu := currentUser(c)
	var rows []model.CalendarShare
	h.db.Where("friend_id = ?", cu.ID).Order("id DESC").Find(&rows)
	items := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		var u model.User
		if err := h.db.First(&u, r.UserID).Error; err != nil {
			continue
		}
		item := h.userProfileJSON(u)
		item["sharedAt"] = r.CreatedAt
		item["birthDate"] = u.BirthDate
		item["expectedLifespan"] = u.ExpectedLifespan
		items = append(items, item)
	}
	c.JSON(200, gin.H{"items": items})
}

// ownerCalendarSummary 汇总某用户的日历格子信息。
func (h *Handler) ownerCalendarSummary(ownerID uint) gin.H {
	var rows []struct {
		RecordDate string
		Cnt        int
	}
	h.db.Model(&model.Record{}).Select("record_date, count(*) as cnt").
		Where("user_id = ?", ownerID).Group("record_date").Scan(&rows)
	records := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		records = append(records, gin.H{"record_date": r.RecordDate, "count": r.Cnt})
	}
	var ms []model.Milestone
	h.db.Select("id", "title", "event_date", "category", "importance").
		Where("user_id = ?", ownerID).Order("event_date ASC").Find(&ms)
	milestones := make([]gin.H, 0, len(ms))
	for _, m := range ms {
		milestones = append(milestones, gin.H{
			"id": m.ID, "title": m.Title, "event_date": m.EventDate, "category": m.Category, "importance": m.Importance,
		})
	}
	var owner model.User
	var birth *string
	var lifespan = 80
	var name string
	var avatar interface{}
	if err := h.db.First(&owner, ownerID).Error; err == nil {
		birth = owner.BirthDate
		lifespan = owner.ExpectedLifespan
		name = owner.DisplayName
		avatar = owner.AvatarURL
	}
	return gin.H{
		"ownerId": ownerID, "ownerName": name, "ownerAvatar": avatar,
		"birthDate": birth, "expectedLifespan": lifespan,
		"records": records, "milestones": milestones,
	}
}

// SharedCalendarSummary 查看（本人或已被分享的好友）对方日历摘要。
func (h *Handler) SharedCalendarSummary(c *gin.Context) {
	cu := currentUser(c)
	ownerID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		badRequest(c, "无效的用户 id")
		return
	}
	if !h.canViewCalendar(cu.ID, uint(ownerID)) {
		fail(c, 403, "你还没有被分享该用户的人生日历")
		return
	}
	c.JSON(200, h.ownerCalendarSummary(uint(ownerID)))
}

// SharedCalendarPeriod 取某周内的大事记与日常记录详情（分享可见）。
func (h *Handler) SharedCalendarPeriod(c *gin.Context) {
	cu := currentUser(c)
	ownerID, err := strconv.Atoi(c.Param("userId"))
	if err != nil {
		badRequest(c, "无效的用户 id")
		return
	}
	if !h.canViewCalendar(cu.ID, uint(ownerID)) {
		fail(c, 403, "你还没有被分享该用户的人生日历")
		return
	}
	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		badRequest(c, "缺少日期范围")
		return
	}
	var recs []model.Record
	h.db.Where("user_id = ? AND record_date >= ? AND record_date <= ?", ownerID, from, to).
		Order("record_date DESC").Find(&recs)
	recordItems := make([]gin.H, 0, len(recs))
	for _, r := range recs {
		recordItems = append(recordItems, h.recordDetail(r.ID))
	}
	var ms []model.Milestone
	h.db.Where("user_id = ? AND event_date >= ? AND event_date <= ?", ownerID, from, to).
		Order("event_date ASC").Find(&ms)
	milestoneItems := make([]gin.H, 0, len(ms))
	for _, m := range ms {
		milestoneItems = append(milestoneItems, h.milestoneDetail(m.ID))
	}
	c.JSON(200, gin.H{"records": recordItems, "milestones": milestoneItems})
}

// commentOwnerCan 能否在 ownerID 的内容上发表/查看私密评价。
func (h *Handler) commentOwnerCan(cuID, ownerID uint) bool {
	return cuID == ownerID || h.isCalendarShared(ownerID, cuID)
}

// ListCalendarComments 某条随记/大事的私密评价（仅双方可见）。
func (h *Handler) ListCalendarComments(c *gin.Context) {
	cu := currentUser(c)
	ownerID, err := strconv.Atoi(c.Query("ownerId"))
	targetID, err2 := strconv.Atoi(c.Query("targetId"))
	targetType := c.Query("targetType")
	if err != nil || err2 != nil || (targetType != "record" && targetType != "milestone") {
		badRequest(c, "请求参数有误")
		return
	}
	if !h.commentOwnerCan(cu.ID, uint(ownerID)) {
		fail(c, 403, "无权查看该内容的评价")
		return
	}
	var cms []model.CalendarComment
	h.db.Where("owner_user_id = ? AND target_type = ? AND target_id = ?", ownerID, targetType, targetID).
		Order("id ASC").Find(&cms)
	// 原先每条评论一次用户查询，收成一次批量；输出顺序仍按 cms 行进
	authorIDs := make([]uint, 0, len(cms))
	for _, cm := range cms {
		authorIDs = append(authorIDs, cm.AuthorUserID)
	}
	authorMap := h.userProfilesByIDs(authorIDs)
	items := make([]gin.H, 0, len(cms))
	for _, cm := range cms {
		authorName := ""
		var avatar interface{}
		if u, ok := authorMap[cm.AuthorUserID]; ok {
			authorName = u.DisplayName
			avatar = u.AvatarURL
		}
		items = append(items, gin.H{
			"id": cm.ID, "authorUserId": cm.AuthorUserID, "authorName": authorName, "authorAvatar": avatar,
			"content": cm.Content, "createdAt": cm.CreatedAt,
		})
	}
	c.JSON(200, gin.H{"items": items})
}

// AddCalendarComment 对某条随记/大事添加私密评价。
func (h *Handler) AddCalendarComment(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		OwnerID    uint   `json:"ownerId"`
		TargetType string `json:"targetType"`
		TargetID   uint   `json:"targetId"`
		Content    string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	if req.TargetID == 0 || (req.TargetType != "record" && req.TargetType != "milestone") {
		badRequest(c, "请求参数有误")
		return
	}
	if !h.commentOwnerCan(cu.ID, req.OwnerID) {
		fail(c, 403, "无权评价该内容")
		return
	}
	// 目标必须属于主人
	var n int64
	if req.TargetType == "record" {
		h.db.Model(&model.Record{}).Where("id = ? AND user_id = ?", req.TargetID, req.OwnerID).Count(&n)
	} else {
		h.db.Model(&model.Milestone{}).Where("id = ? AND user_id = ?", req.TargetID, req.OwnerID).Count(&n)
	}
	if n == 0 {
		notFound(c, "内容不存在")
		return
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		badRequest(c, "评价内容不能为空")
		return
	}
	if len([]rune(req.Content)) > calendarCommentMax {
		badRequest(c, "评价不能超过 500 字")
		return
	}
	cm := model.CalendarComment{
		OwnerUserID:  req.OwnerID,
		AuthorUserID: cu.ID,
		TargetType:   req.TargetType,
		TargetID:     req.TargetID,
		Content:      req.Content,
		CreatedAt:    model.NowISO(),
	}
	if err := h.db.Create(&cm).Error; err != nil {
		serverError(c, "评价失败")
		return
	}
	var u model.User
	h.db.First(&u, cu.ID)
	item := gin.H{
		"id": cm.ID, "authorUserId": cu.ID, "authorName": u.DisplayName, "authorAvatar": u.AvatarURL,
		"content": cm.Content, "createdAt": cm.CreatedAt,
	}
	c.JSON(200, item)
}

// DeleteCalendarComment 删除自己的私密评价。
func (h *Handler) DeleteCalendarComment(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	res := h.db.Where("id = ? AND author_user_id = ?", id, cu.ID).Delete(&model.CalendarComment{})
	if res.Error != nil {
		serverError(c, "操作失败")
		return
	}
	if res.RowsAffected == 0 {
		fail(c, 400, "只能删除自己的评价")
		return
	}
	c.JSON(200, gin.H{"success": true})
}
