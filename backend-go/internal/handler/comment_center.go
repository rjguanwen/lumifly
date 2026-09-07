package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// isAcceptedFriend 两人是否为已接受的好友。
func (h *Handler) isAcceptedFriend(a, b uint) bool {
	var n int64
	h.db.Model(&model.Friendship{}).
		Where("status = ? AND ((user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?))",
			model.FriendAccepted, a, b, b, a).Count(&n)
	return n > 0
}

// pairComments 我与对方在双方日历内容上的全部往来评论（升序）。
func (h *Handler) pairComments(meID, otherID uint) []model.CalendarComment {
	var cms []model.CalendarComment
	h.db.Where("(owner_user_id = ? AND author_user_id = ?) OR (owner_user_id = ? AND author_user_id = ?)",
		meID, otherID, otherID, meID).Order("id ASC").Find(&cms)
	return cms
}

// pairMaxCommentID 该对往来评论的最大 id。
func (h *Handler) pairMaxCommentID(meID, otherID uint) uint {
	var mx uint
	rows := h.pairComments(meID, otherID)
	for _, cm := range rows {
		if cm.ID > mx {
			mx = cm.ID
		}
	}
	return mx
}

// myReadID 我对某对往来的已读水位。
func (h *Handler) myReadID(meID, otherID uint) uint {
	var r model.CalendarCommentRead
	if err := h.db.Where("user_id = ? AND other_id = ?", meID, otherID).First(&r).Error; err == nil {
		return r.LastCommentID
	}
	return 0
}

// markPairRead 打开评论中心后，把与对方的已读水位推进到最新。
func (h *Handler) markPairRead(meID, otherID uint) {
	maxID := h.pairMaxCommentID(meID, otherID)
	now := model.NowISO()
	var r model.CalendarCommentRead
	if err := h.db.Where("user_id = ? AND other_id = ?", meID, otherID).First(&r).Error; err == nil {
		h.db.Model(&model.CalendarCommentRead{}).Where("id = ?", r.ID).
			Updates(map[string]interface{}{"last_comment_id": maxID, "updated_at": now})
		return
	}
	h.db.Create(&model.CalendarCommentRead{UserID: meID, OtherID: otherID, LastCommentID: maxID, UpdatedAt: now})
}

// CommentCenterUnread 我与每位好友之间的未读往来评论数。
func (h *Handler) CommentCenterUnread(c *gin.Context) {
	cu := currentUser(c)
	var rows []model.Friendship
	h.db.Where("status = ? AND (user_id = ? OR friend_id = ?)", model.FriendAccepted, cu.ID, cu.ID).Find(&rows)
	seen := map[uint]bool{}
	items := make([]gin.H, 0)
	for _, f := range rows {
		var otherID uint
		if f.UserID == cu.ID {
			otherID = f.FriendID
		} else {
			otherID = f.UserID
		}
		if seen[otherID] {
			continue
		}
		seen[otherID] = true
		var unread int64
		last := h.myReadID(cu.ID, otherID)
		h.db.Model(&model.CalendarComment{}).
			Where("((owner_user_id = ? AND author_user_id = ?) OR (owner_user_id = ? AND author_user_id = ?)) AND id > ? AND author_user_id <> ?",
				cu.ID, otherID, otherID, cu.ID, last, cu.ID).Count(&unread)
		if unread == 0 {
			continue
		}
		var u model.User
		if err := h.db.First(&u, otherID).Error; err != nil {
			continue
		}
		item := h.userProfileJSON(u)
		item["unread"] = unread
		items = append(items, item)
	}
	c.JSON(200, gin.H{"items": items})
}

// CommentCenterList 与某位好友的私密评论中心（按内容分组）。
func (h *Handler) CommentCenterList(c *gin.Context) {
	cu := currentUser(c)
	friendID, err := strconv.Atoi(c.Query("friendId"))
	if err != nil {
		badRequest(c, "缺少 friendId")
		return
	}
	if !h.isAcceptedFriend(cu.ID, uint(friendID)) {
		fail(c, 400, "仅好友之间可以查看评论")
		return
	}
	cms := h.pairComments(cu.ID, uint(friendID))

	lookup := func(ownerID uint, typ string, tid uint) (title, date string) {
		if typ == "record" {
			var r model.Record
			if err := h.db.Select("title", "record_date").First(&r, tid).Error; err == nil {
				title, date = r.Title, r.RecordDate
			}
			if title == "" {
				title = "（无标题日记）"
			}
			return title, date
		}
		var m model.Milestone
		if err := h.db.Select("title", "event_date").First(&m, tid).Error; err == nil {
			title, date = m.Title, m.EventDate
		}
		if title == "" {
			title = "（无标题大事）"
		}
		return title, date
	}
	ownerName := func(uid uint) string {
		var u model.User
		if err := h.db.First(&u, uid).Error; err == nil {
			return u.DisplayName
		}
		return ""
	}

	// 先按内容归类评论
	commentsByKey := map[string][]gin.H{}
	var order []string
	for _, cm := range cms {
		var u model.User
		h.db.First(&u, cm.AuthorUserID)
		item := gin.H{
			"id": cm.ID, "authorUserId": cm.AuthorUserID, "authorName": u.DisplayName,
			"authorAvatar": u.AvatarURL, "content": cm.Content, "createdAt": cm.CreatedAt,
			"isMine": cm.AuthorUserID == cu.ID,
		}
		key := strconv.FormatUint(uint64(cm.OwnerUserID), 10) + ":" + cm.TargetType + ":" + strconv.FormatUint(uint64(cm.TargetID), 10)
		if _, ok := commentsByKey[key]; !ok {
			order = append(order, key)
		}
		commentsByKey[key] = append(commentsByKey[key], item)
	}

	groups := make([]gin.H, 0, len(order))
	for _, key := range order {
		parts := strings.SplitN(key, ":", 3)
		ownerID, _ := strconv.ParseUint(parts[0], 10, 64)
		typ := parts[1]
		tid, _ := strconv.ParseUint(parts[2], 10, 64)
		title, date := lookup(uint(ownerID), typ, uint(tid))
		groups = append(groups, gin.H{
			"ownerUserId": ownerID, "ownerName": ownerName(uint(ownerID)),
			"isMine": ownerID == uint64(cu.ID), "targetType": typ, "targetId": tid,
			"title": title, "date": date, "comments": commentsByKey[key],
		})
	}
	var friend model.User
	h.db.First(&friend, friendID)
	c.JSON(200, gin.H{
		"friend": h.userProfileJSON(friend),
		"groups": groups,
	})
}

// CommentCenterRead 标记与某位好友的评论为已读。
func (h *Handler) CommentCenterRead(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		FriendID uint `json:"friendId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FriendID == 0 {
		badRequest(c, "请求参数有误")
		return
	}
	h.markPairRead(cu.ID, req.FriendID)
	c.JSON(200, gin.H{"success": true})
}
