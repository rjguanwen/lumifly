package handler

import (
	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// squareReadWatermark 读取某用户的广场已读水位；为空（新账号）则视为当前时刻，避免历史内容被误标未读。
func (h *Handler) squareReadWatermark(cuID uint) string {
	var u model.User
	if err := h.db.Select("square_read_at").First(&u, cuID).Error; err != nil {
		return model.NowISO()
	}
	if u.SquareReadAt == "" {
		return model.NowISO()
	}
	return u.SquareReadAt
}

// advanceSquareRead 浏览公开帖后推进已读水位（只前进不后退）。
func (h *Handler) advanceSquareRead(cuID uint, upTo string) {
	if upTo == "" {
		return
	}
	cur := h.squareReadWatermark(cuID)
	if upTo <= cur {
		return
	}
	h.db.Model(&model.User{}).Where("id = ?", cuID).Update("square_read_at", upTo)
}

// UnreadBadges 顶部菜单红点统计：未处理好友请求数 + 广场新公开内容数。
func (h *Handler) UnreadBadges(c *gin.Context) {
	cu := currentUser(c)
	var reqCount int64
	h.db.Model(&model.Friendship{}).
		Where("friend_id = ? AND status = ?", cu.ID, model.FriendPending).Count(&reqCount)

	wm := h.squareReadWatermark(cu.ID)
	var sqCount int64
	h.db.Model(&model.Publication{}).
		Where("status = ? AND user_id <> ? AND updated_at > ?", model.PubStatusPublished, cu.ID, wm).Count(&sqCount)

	c.JSON(200, gin.H{"friendRequests": reqCount, "squareUnread": sqCount})
}
