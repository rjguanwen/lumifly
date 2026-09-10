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
// 原先先把全部评论行拉回内存再取最大值（行含 content 大字段），现在交给 SQL 聚合。
// MAX(id) 等于原「逐行比 id 取大」，无数据时 COALESCE 给 0，与原初值 mx=0 一致。
func (h *Handler) pairMaxCommentID(meID, otherID uint) uint {
	var rows []pairMaxRow
	h.db.Model(&model.CalendarComment{}).
		Select("COALESCE(MAX(id), 0) AS max_id").
		Where("(owner_user_id = ? AND author_user_id = ?) OR (owner_user_id = ? AND author_user_id = ?)",
			meID, otherID, otherID, meID).
		Scan(&rows)
	if len(rows) == 0 {
		return 0
	}
	return rows[0].MaxID
}

// pairMaxRow 聚合结果接收行。
type pairMaxRow struct {
	MaxID uint `gorm:"column:max_id"`
}

// pairCountRow 按「对方」分组计数的结果接收行。
type pairCountRow struct {
	OtherID uint  `gorm:"column:other_id"`
	Cnt     int64 `gorm:"column:cnt"`
}

// readIDsForPairs 一次取回我对多个好友的已读水位。
// 无记录的对不进结果，调用方取 map 零值得 0，与原 myReadID「查不到则返回 0」一致。
// (user_id, other_id) 上有唯一索引，每对最多一条；仍按 id 升序首次命中，与 First 完全对齐。
func (h *Handler) readIDsForPairs(meID uint, otherIDs []uint) map[uint]uint {
	out := make(map[uint]uint, len(otherIDs))
	if len(otherIDs) == 0 {
		return out
	}
	for _, part := range chunkIDs(otherIDs, idChunkSize) {
		var list []model.CalendarCommentRead
		h.db.Where("user_id = ? AND other_id IN ?", meID, part).Order("id ASC").Find(&list)
		for _, r := range list {
			if _, ok := out[r.OtherID]; !ok {
				out[r.OtherID] = r.LastCommentID
			}
		}
	}
	return out
}

// pairUnreadCounts 一次算出我与多个好友各自的未读往来评论数。
// 判定条件与原逐对 Count 逐字一致（同一个 OR 谓词 + id > 水位 + 排除自己发的评论），
// 只是把「按对循环」改成「按水位分桶 + GROUP BY 对方」：同一桶内水位相同，分组计数
// 与逐对计数等价。水位相同的对通常只有一个桶（多为 0），因此常见情况下只需 1 条 SQL。
func (h *Handler) pairUnreadCounts(meID uint, otherIDs []uint, lasts map[uint]uint) map[uint]int64 {
	out := make(map[uint]int64, len(otherIDs))
	buckets := make(map[uint][]uint)
	var bucketKeys []uint
	for _, o := range otherIDs {
		l := lasts[o]
		if _, ok := buckets[l]; !ok {
			bucketKeys = append(bucketKeys, l)
		}
		buckets[l] = append(buckets[l], o)
	}
	for _, l := range bucketKeys {
		for _, part := range chunkIDs(buckets[l], idChunkSize) {
			var rows []pairCountRow
			h.db.Model(&model.CalendarComment{}).
				Select("author_user_id AS other_id, COUNT(*) AS cnt").
				Where("((owner_user_id = ? AND author_user_id IN ?) OR (owner_user_id IN ? AND author_user_id = ?)) AND id > ? AND author_user_id <> ?",
					meID, part, part, meID, l, meID).
				Group("author_user_id").Scan(&rows)
			for _, r := range rows {
				out[r.OtherID] = r.Cnt
			}
		}
	}
	return out
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
	// 第一遍：按原顺序去重出好友列表（原实现边遍历边用 seen 去重，输出顺序由此决定）
	seen := map[uint]bool{}
	others := make([]uint, 0, len(rows))
	for _, f := range rows {
		otherID := otherFriendID(cu.ID, f)
		if seen[otherID] {
			continue
		}
		seen[otherID] = true
		others = append(others, otherID)
	}
	// 原先每位好友 2~3 条 SQL，现在水位、计数、用户资料各 1 条
	lasts := h.readIDsForPairs(cu.ID, others)
	unreads := h.pairUnreadCounts(cu.ID, others, lasts)
	needIDs := make([]uint, 0, len(others))
	for _, otherID := range others {
		if unreads[otherID] > 0 {
			needIDs = append(needIDs, otherID)
		}
	}
	users := h.userProfilesByIDs(needIDs)

	items := make([]gin.H, 0)
	for _, otherID := range others {
		unread := unreads[otherID]
		if unread == 0 {
			continue
		}
		u, ok := users[otherID]
		if !ok {
			continue
		}
		item := h.userProfileJSON(u)
		item["unread"] = unread
		items = append(items, item)
	}
	c.JSON(200, gin.H{"items": items})
}

// titleDateRow 标题/日期批量查询的接收行。
type titleDateRow struct {
	ID    uint   `gorm:"column:id"`
	Title string `gorm:"column:title"`
	Date  string `gorm:"column:date"`
}

// titlesDatesByIDs 批量取标题与日期。dateCol 由调用方以代码内常量传入（不接受外部输入）。
// 查不到的 id 不进结果，调用方按原「First 失败则留空」的语义处理。
func (h *Handler) titlesDatesByIDs(modelValue interface{}, dateCol string, ids []uint) map[uint]titleDateRow {
	out := make(map[uint]titleDateRow, len(ids))
	if len(ids) == 0 {
		return out
	}
	for _, part := range chunkIDs(ids, idChunkSize) {
		var rows []titleDateRow
		h.db.Model(modelValue).Select("id, title, "+dateCol+" AS date").
			Where("id IN ?", part).Scan(&rows)
		for _, r := range rows {
			out[r.ID] = r
		}
	}
	return out
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

	// 原先每条评论、每组归属人各一次用户查询；这里合并为一次（含末尾的好友资料）
	seenUID := map[uint]bool{}
	uidList := make([]uint, 0, len(cms)*2+1)
	for _, cm := range cms {
		for _, id := range [2]uint{cm.AuthorUserID, cm.OwnerUserID} {
			if !seenUID[id] {
				seenUID[id] = true
				uidList = append(uidList, id)
			}
		}
	}
	if uint(friendID) != 0 && !seenUID[uint(friendID)] {
		uidList = append(uidList, uint(friendID))
	}
	users := h.userProfilesByIDs(uidList)

	// 先按内容归类评论
	commentsByKey := map[string][]gin.H{}
	var order []string
	for _, cm := range cms {
		// 查不到用户时与原实现一样用零值（名字为空、头像为 null）
		u := users[cm.AuthorUserID]
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

	// 批量取各分组对应内容的标题与日期（原先每组一条 SQL）
	var recIDs, msIDs []uint
	for _, key := range order {
		parts := strings.SplitN(key, ":", 3)
		tid, _ := strconv.ParseUint(parts[2], 10, 64)
		if parts[1] == "record" {
			recIDs = append(recIDs, uint(tid))
		} else {
			msIDs = append(msIDs, uint(tid))
		}
	}
	recTitles := h.titlesDatesByIDs(&model.Record{}, "record_date", recIDs)
	msTitles := h.titlesDatesByIDs(&model.Milestone{}, "event_date", msIDs)

	groups := make([]gin.H, 0, len(order))
	for _, key := range order {
		parts := strings.SplitN(key, ":", 3)
		ownerID, _ := strconv.ParseUint(parts[0], 10, 64)
		typ := parts[1]
		tid, _ := strconv.ParseUint(parts[2], 10, 64)
		var title, date string
		if typ == "record" {
			if td, ok := recTitles[uint(tid)]; ok {
				title, date = td.Title, td.Date
			}
			if title == "" {
				title = "（无标题日记）"
			}
		} else {
			if td, ok := msTitles[uint(tid)]; ok {
				title, date = td.Title, td.Date
			}
			if title == "" {
				title = "（无标题大事）"
			}
		}
		groups = append(groups, gin.H{
			"ownerUserId": ownerID, "ownerName": users[uint(ownerID)].DisplayName,
			"isMine": ownerID == uint64(cu.ID), "targetType": typ, "targetId": tid,
			"title": title, "date": date, "comments": commentsByKey[key],
		})
	}
	c.JSON(200, gin.H{
		"friend": h.userProfileJSON(users[uint(friendID)]),
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
