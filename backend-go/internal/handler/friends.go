package handler

import (
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// userProfileJSON 公开用户资料（不含邮箱/密码）。
func (h *Handler) userProfileJSON(u model.User) gin.H {
	return gin.H{
		"id":          u.ID,
		"displayName": u.DisplayName,
		"avatarUrl":   u.AvatarURL,
		"signature":   u.Signature,
	}
}

// relationBetween 计算我与另一用户的当前关系：friend/requested/requested_me/none。
func (h *Handler) relationBetween(meID, otherID uint) (string, *model.Friendship) {
	var f model.Friendship
	if err := h.db.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", meID, otherID, otherID, meID).
		First(&f).Error; err != nil {
		return "none", nil
	}
	if f.Status == model.FriendAccepted {
		return "friend", &f
	}
	if f.UserID == meID {
		return "requested", &f // 我发起的待处理申请
	}
	return "requested_me", &f // 对方发给我的申请
}

// SearchUsers 按昵称搜索用户（用于添加好友）。
func (h *Handler) SearchUsers(c *gin.Context) {
	cu := currentUser(c)
	keyword := strings.TrimSpace(c.Query("keyword"))
	items := make([]gin.H, 0)
	if keyword == "" {
		c.JSON(200, gin.H{"items": items})
		return
	}
	like := "%" + keyword + "%"
	var users []model.User
	if err := h.db.Select("id", "display_name", "avatar_url", "signature").
		Where("display_name LIKE ? AND id <> ?", like, cu.ID).Limit(20).Find(&users).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	for _, u := range users {
		rel, f := h.relationBetween(cu.ID, u.ID)
		item := h.userProfileJSON(u)
		item["relation"] = rel
		if f != nil {
			item["relationId"] = f.ID
		}
		items = append(items, item)
	}
	c.JSON(200, gin.H{"items": items})
}

// SendFriendRequest 发送好友申请。
func (h *Handler) SendFriendRequest(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		FriendID uint `json:"friendId"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.FriendID == 0 {
		badRequest(c, "请选择要添加的用户")
		return
	}
	if req.FriendID == cu.ID {
		badRequest(c, "不能添加自己为好友")
		return
	}
	var target model.User
	if err := h.db.First(&target, req.FriendID).Error; err != nil {
		notFound(c, "用户不存在")
		return
	}
	rel, f := h.relationBetween(cu.ID, req.FriendID)
	switch rel {
	case "friend":
		fail(c, 400, "你们已经是好友了")
		return
	case "requested":
		fail(c, 400, "你已向对方发送过申请，等待对方处理")
		return
	case "requested_me":
		// 对方也申请了我：直接互相成为好友
		if err := h.db.Model(&model.Friendship{}).Where("id = ?", f.ID).
			Updates(map[string]interface{}{"status": model.FriendAccepted, "updated_at": model.NowISO()}).Error; err != nil {
			serverError(c, "操作失败")
			return
		}
		c.JSON(200, gin.H{"success": true, "accepted": true})
		return
	}
	now := model.NowISO()
	fs := &model.Friendship{
		UserID:    cu.ID,
		FriendID:  req.FriendID,
		Status:    model.FriendPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.db.Create(fs).Error; err != nil {
		serverError(c, "发送失败")
		return
	}
	c.JSON(200, gin.H{"success": true, "accepted": false, "id": fs.ID})
}

// ListFriendRequests 我收到的与发出的待处理申请。
func (h *Handler) ListFriendRequests(c *gin.Context) {
	cu := currentUser(c)

	var received []model.Friendship
	h.db.Where("friend_id = ? AND status = ?", cu.ID, model.FriendPending).Order("id DESC").Find(&received)
	receivedItems := make([]gin.H, 0, len(received))
	for _, f := range received {
		var u model.User
		if err := h.db.First(&u, f.UserID).Error; err != nil {
			continue
		}
		item := h.userProfileJSON(u)
		item["requestId"] = f.ID
		item["createdAt"] = f.CreatedAt
		receivedItems = append(receivedItems, item)
	}

	var sent []model.Friendship
	h.db.Where("user_id = ? AND status = ?", cu.ID, model.FriendPending).Order("id DESC").Find(&sent)
	sentItems := make([]gin.H, 0, len(sent))
	for _, f := range sent {
		var u model.User
		if err := h.db.First(&u, f.FriendID).Error; err != nil {
			continue
		}
		item := h.userProfileJSON(u)
		item["requestId"] = f.ID
		item["createdAt"] = f.CreatedAt
		sentItems = append(sentItems, item)
	}
	c.JSON(200, gin.H{"received": receivedItems, "sent": sentItems})
}

// AcceptFriendRequest 接受好友申请。
func (h *Handler) AcceptFriendRequest(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	res := h.db.Model(&model.Friendship{}).
		Where("id = ? AND friend_id = ? AND status = ?", id, cu.ID, model.FriendPending).
		Update("status", model.FriendAccepted)
	if res.Error != nil {
		serverError(c, "操作失败")
		return
	}
	if res.RowsAffected == 0 {
		fail(c, 400, "该申请不存在或已处理")
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// CancelOrRejectFriendRequest 撤销我发出的申请，或拒绝收到的申请。
func (h *Handler) CancelOrRejectFriendRequest(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	res := h.db.Where("id = ? AND status = ? AND (user_id = ? OR friend_id = ?)",
		id, model.FriendPending, cu.ID, cu.ID).Delete(&model.Friendship{})
	if res.Error != nil {
		serverError(c, "操作失败")
		return
	}
	if res.RowsAffected == 0 {
		fail(c, 400, "该申请不存在或已处理")
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// ListFriends 我的好友列表。
func (h *Handler) ListFriends(c *gin.Context) {
	cu := currentUser(c)
	var rows []model.Friendship
	h.db.Where("status = ? AND (user_id = ? OR friend_id = ?)", model.FriendAccepted, cu.ID, cu.ID).
		Order("updated_at DESC").Find(&rows)

	friends := make([]gin.H, 0, len(rows))
	for _, f := range rows {
		var otherID uint
		if f.UserID == cu.ID {
			otherID = f.FriendID
		} else {
			otherID = f.UserID
		}
		var u model.User
		if err := h.db.First(&u, otherID).Error; err != nil {
			continue
		}
		item := h.userProfileJSON(u)
		item["friendSince"] = f.UpdatedAt
		friends = append(friends, item)
	}
	c.JSON(200, gin.H{"items": friends})
}

// RemoveFriend 解除好友关系（双向删除）。
func (h *Handler) RemoveFriend(c *gin.Context) {
	cu := currentUser(c)
	friendID, err := strconv.Atoi(c.Param("friendId"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	res := h.db.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)",
		cu.ID, friendID, friendID, cu.ID).Delete(&model.Friendship{})
	if res.Error != nil {
		serverError(c, "操作失败")
		return
	}
	if res.RowsAffected == 0 {
		fail(c, 400, "你们还不是好友")
		return
	}
	c.JSON(200, gin.H{"success": true})
}
