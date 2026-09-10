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

// relationFrom 由一条已有的好友关系记录推导关系字符串（判定与 relationBetween 完全一致）。
func relationFrom(meID uint, f *model.Friendship) string {
	if f == nil {
		return "none"
	}
	if f.Status == model.FriendAccepted {
		return "friend"
	}
	if f.UserID == meID {
		return "requested" // 我发起的待处理申请
	}
	return "requested_me" // 对方发给我的申请
}

// relationBetween 计算我与另一用户的当前关系：friend/requested/requested_me/none。
func (h *Handler) relationBetween(meID, otherID uint) (string, *model.Friendship) {
	var f model.Friendship
	if err := h.db.Where("(user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)", meID, otherID, otherID, meID).
		First(&f).Error; err != nil {
		return "none", nil
	}
	return relationFrom(meID, &f), &f
}

// otherFriendID 取一条好友关系记录中「对方」那一端的用户 id。
func otherFriendID(meID uint, f model.Friendship) uint {
	if f.UserID == meID {
		return f.FriendID
	}
	return f.UserID
}

// relationsBetween 一次算出我与多个用户的关系，语义与逐个调 relationBetween 完全一致。
// 同一对用户可能存在两个方向的两条记录，原实现用 First（按主键升序取首条），
// 这里同样按 id 升序取首次命中，保证取到的是同一条记录。
func (h *Handler) relationsBetween(meID uint, otherIDs []uint) map[uint]model.Friendship {
	out := make(map[uint]model.Friendship, len(otherIDs))
	if len(otherIDs) == 0 {
		return out
	}
	for _, part := range chunkIDs(otherIDs, idChunkSize) {
		var rows []model.Friendship
		h.db.Where("(user_id IN ? AND friend_id = ?) OR (user_id = ? AND friend_id IN ?)", part, meID, meID, part).
			Order("id ASC").Find(&rows)
		for _, f := range rows {
			other := f.UserID
			if other == meID {
				other = f.FriendID
			}
			if _, ok := out[other]; !ok {
				out[other] = f
			}
		}
	}
	return out
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
	// 原先每人一条关系查询（最多 20 条），现在合计一条
	candidateIDs := make([]uint, 0, len(users))
	for _, u := range users {
		candidateIDs = append(candidateIDs, u.ID)
	}
	rels := h.relationsBetween(cu.ID, candidateIDs)
	for _, u := range users {
		item := h.userProfileJSON(u)
		if f, ok := rels[u.ID]; ok {
			item["relation"] = relationFrom(cu.ID, &f)
			item["relationId"] = f.ID
		} else {
			item["relation"] = "none"
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
	// 原先每条申请一次用户查询，收成一次批量；输出顺序仍按 received 行进
	recvIDs := make([]uint, 0, len(received))
	for _, f := range received {
		recvIDs = append(recvIDs, f.UserID)
	}
	recvUsers := h.userProfilesByIDs(recvIDs)
	receivedItems := make([]gin.H, 0, len(received))
	for _, f := range received {
		u, ok := recvUsers[f.UserID]
		if !ok {
			continue
		}
		item := h.userProfileJSON(u)
		item["requestId"] = f.ID
		item["createdAt"] = f.CreatedAt
		receivedItems = append(receivedItems, item)
	}

	var sent []model.Friendship
	h.db.Where("user_id = ? AND status = ?", cu.ID, model.FriendPending).Order("id DESC").Find(&sent)
	sentIDs := make([]uint, 0, len(sent))
	for _, f := range sent {
		sentIDs = append(sentIDs, f.FriendID)
	}
	sentUsers := h.userProfilesByIDs(sentIDs)
	sentItems := make([]gin.H, 0, len(sent))
	for _, f := range sent {
		u, ok := sentUsers[f.FriendID]
		if !ok {
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

	friendIDs := make([]uint, 0, len(rows))
	for _, f := range rows {
		friendIDs = append(friendIDs, otherFriendID(cu.ID, f))
	}
	userMap := h.userProfilesByIDs(friendIDs)

	friends := make([]gin.H, 0, len(rows))
	for i, f := range rows {
		u, ok := userMap[friendIDs[i]]
		if !ok {
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
