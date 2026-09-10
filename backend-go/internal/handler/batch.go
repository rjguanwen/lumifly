package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lumiflybackend/internal/model"
)

// ---------- 批量关联加载 ----------
//
// 本文件的助手用于消除「列表接口逐行查标签/媒体/作者/统计」的 N+1 查询。
// 刻意保持与各逐行版本的输出完全一致：字段集合相同、排序规则相同、
// 没有关联时返回空数组（而不是 null）。表名/列名均来自代码内常量，不接受外部输入。

// idChunkSize 单条 IN 查询携带的最大参数个数。
// SQLite 对绑定变量数量有上限，列表接口单次 limit 最多 100，但日历汇总等场景
// 会一次性传入全部历史数据，这里分片以彻底避开该限制。
const idChunkSize = 900

// chunkIDs 将 id 列表按 size 分片。
func chunkIDs(ids []uint, size int) [][]uint {
	if len(ids) == 0 {
		return nil
	}
	out := make([][]uint, 0, (len(ids)+size-1)/size)
	for start := 0; start < len(ids); start += size {
		end := start + size
		if end > len(ids) {
			end = len(ids)
		}
		out = append(out, ids[start:end])
	}
	return out
}

// tagJSONRow 标签的对外结构（与 tagsByIDs 原有输出一致）。
func tagJSONRow(t *model.Tag) gin.H {
	return gin.H{
		"id": t.ID, "userId": t.UserID, "name": t.Name,
		"color": t.Color, "createdAt": t.CreatedAt,
	}
}

// tagsJSONByIDs 一次性取回多个标签 id 的对外结构。
func (h *Handler) tagsJSONByIDs(ids []uint) map[uint]gin.H {
	out := make(map[uint]gin.H, len(ids))
	if len(ids) == 0 {
		return out
	}
	var tags []model.Tag
	for _, part := range chunkIDs(ids, idChunkSize) {
		var batch []model.Tag
		h.db.Where("id IN ?", part).Find(&batch)
		tags = append(tags, batch...)
	}
	for i := range tags {
		t := &tags[i]
		out[t.ID] = tagJSONRow(t)
	}
	return out
}

// entityTagLink 关联表查询的接收行。
type entityTagLink struct {
	EntityID uint `gorm:"column:entity_id"`
	TagID    uint `gorm:"column:tag_id"`
}

// tagsByEntities 批量加载多个实体的标签：entityID -> tags JSON。
// table 为关联表名、column 为该表中指向实体的列名，均由调用方以常量传入。
// 每个实体的标签顺序为 tag_id 升序，与原「按关联表主键取 links + WHERE id IN」的结果一致。
// SQL 条数：关联 1 条 + 标签 1 条（分片时各加一片），与实体数量无关。
func (h *Handler) tagsByEntities(table, column string, entityIDs []uint) map[uint][]gin.H {
	out := make(map[uint][]gin.H, len(entityIDs))
	// 先补齐空数组，保证无标签的实体输出 [] 而非 null
	for _, id := range entityIDs {
		out[id] = []gin.H{}
	}
	if len(entityIDs) == 0 {
		return out
	}

	var links []entityTagLink
	for _, part := range chunkIDs(entityIDs, idChunkSize) {
		var batch []entityTagLink
		h.db.Table(table).
			Select(column + " AS entity_id, tag_id").
			Where(column+" IN ?", part).
			Order(column + " ASC, tag_id ASC").
			Find(&batch)
		links = append(links, batch...)
	}
	if len(links) == 0 {
		return out
	}

	seen := make(map[uint]struct{}, len(links))
	all := make([]uint, 0, len(links))
	for _, l := range links {
		if _, ok := seen[l.TagID]; !ok {
			seen[l.TagID] = struct{}{}
			all = append(all, l.TagID)
		}
	}
	tagJSON := h.tagsJSONByIDs(all)
	for _, l := range links {
		if j, ok := tagJSON[l.TagID]; ok {
			out[l.EntityID] = append(out[l.EntityID], j)
		}
	}
	return out
}

// mediaByEntities 批量加载多个实体的媒体：entityID -> media JSON。
// 与逐行 mediaByEntity 一致：按 id 升序（即插入顺序），无媒体的实体输出 [] 而非 null。
func (h *Handler) mediaByEntities(entityType string, entityIDs []uint) map[uint][]gin.H {
	out := make(map[uint][]gin.H, len(entityIDs))
	for _, id := range entityIDs {
		out[id] = []gin.H{}
	}
	if len(entityIDs) == 0 {
		return out
	}
	grouped := make(map[uint][]model.Media, len(entityIDs))
	for _, part := range chunkIDs(entityIDs, idChunkSize) {
		var list []model.Media
		h.db.Where("entity_type = ? AND entity_id IN ?", entityType, part).
			Order("id ASC").Find(&list)
		for _, m := range list {
			if m.EntityID != nil {
				grouped[*m.EntityID] = append(grouped[*m.EntityID], m)
			}
		}
	}
	for id, list := range grouped {
		out[id] = mediaJSON(list)
	}
	return out
}

// userBrief 广场作者展示信息（与原逐行 SELECT 的三列一致）。
type userBrief struct {
	DisplayName string
	AvatarURL   string
	Signature   string
}

// usersBriefByIDs 批量取用户展示信息。查不到的 id 留空（原逐行版本忽略错误同样得到空值）。
// 注意：这里故意 Scan 进 model.User 而非自定义结构，avatar_url / signature 在旧库里可为 NULL，
// 交由 GORM 按原逐行 First(&u) 的同样方式处理，保证输出逐字段一致。
func (h *Handler) usersBriefByIDs(ids []uint) map[uint]userBrief {
	out := make(map[uint]userBrief, len(ids))
	if len(ids) == 0 {
		return out
	}
	for _, part := range chunkIDs(ids, idChunkSize) {
		var users []model.User
		h.db.Select("id, display_name, avatar_url, signature").Where("id IN ?", part).Find(&users)
		for i := range users {
			u := &users[i]
			brief := userBrief{DisplayName: u.DisplayName, Signature: u.Signature}
			if u.AvatarURL != nil {
				brief.AvatarURL = *u.AvatarURL
			}
			out[u.ID] = brief
		}
	}
	return out
}

// userProfilesByIDs 批量取公开资料（userProfileJSON）所需的四列。
// avatar_url 刻意保持 *string：这样 userProfileJSON 的输出与原「First(&u, id) 后取 u.AvatarURL」
// 逐字节一致（无头像时为 null，而不是空串）。查不到的 id 不进结果，由调用方按原有语义处理。
func (h *Handler) userProfilesByIDs(ids []uint) map[uint]model.User {
	out := make(map[uint]model.User, len(ids))
	if len(ids) == 0 {
		return out
	}
	for _, part := range chunkIDs(ids, idChunkSize) {
		var users []model.User
		h.db.Select("id, display_name, avatar_url, signature").Where("id IN ?", part).Find(&users)
		for i := range users {
			out[users[i].ID] = users[i]
		}
	}
	return out
}

// pubStat 广场帖的点赞数/评论数/当前用户是否已赞。
type pubStat struct {
	LikeCount    int64
	CommentCount int64
	Liked        bool
}

// countRow 分组计数结果的接收行。
type countRow struct {
	PublicationID uint  `gorm:"column:publication_id"`
	Cnt           int64 `gorm:"column:cnt"`
}

// countsByPubIDs 按帖子分组计数（count 为 0 的帖子不会出现在结果里）。
func (h *Handler) countsByPubIDs(modelValue interface{}, pubIDs []uint) map[uint]int64 {
	out := make(map[uint]int64, len(pubIDs))
	if len(pubIDs) == 0 {
		return out
	}
	for _, part := range chunkIDs(pubIDs, idChunkSize) {
		var rows []countRow
		h.db.Model(modelValue).
			Select("publication_id, COUNT(*) AS cnt").
			Where("publication_id IN ?", part).
			Group("publication_id").
			Scan(&rows)
		for _, r := range rows {
			out[r.PublicationID] = r.Cnt
		}
	}
	return out
}

// likedPubIDs 取当前用户已点赞的帖子集合。
func (h *Handler) likedPubIDs(pubIDs []uint, cuID uint) map[uint]bool {
	out := make(map[uint]bool, len(pubIDs))
	if cuID == 0 || len(pubIDs) == 0 {
		return out
	}
	for _, part := range chunkIDs(pubIDs, idChunkSize) {
		var ids []uint
		h.db.Model(&model.PublicationLike{}).Where("publication_id IN ? AND user_id = ?", part, cuID).
			Pluck("publication_id", &ids)
		for _, id := range ids {
			out[id] = true
		}
	}
	return out
}

// pubStatsByIDs 一次取回多个帖子的点赞数/评论数/已赞状态。
// 原先每个帖子 3 条 SQL，这里合计 3 条。
func (h *Handler) pubStatsByIDs(pubIDs []uint, cuID uint) map[uint]pubStat {
	out := make(map[uint]pubStat, len(pubIDs))
	likes := h.countsByPubIDs(&model.PublicationLike{}, pubIDs)
	comments := h.countsByPubIDs(&model.PublicationComment{}, pubIDs)
	liked := h.likedPubIDs(pubIDs, cuID)
	for _, id := range pubIDs {
		out[id] = pubStat{
			LikeCount:    likes[id],
			CommentCount: comments[id],
			Liked:        liked[id],
		}
	}
	return out
}

// ---------- 写入侧批量 ----------

// tagLinkTable 返回内容类型对应的标签关联表与实体列名。
// 映射全部为代码内常量（不接受外部输入），因此可安全拼接进 SQL。
func tagLinkTable(entityType string) (table, column string, ok bool) {
	switch entityType {
	case "record":
		return "record_tags", "record_id", true
	case "milestone":
		return "milestone_tags", "milestone_id", true
	case "idea":
		return "idea_tags", "idea_id", true
	case "quick":
		return "quick_note_tags", "quick_note_id", true
	case "book":
		return "book_tags", "book_id", true
	}
	return "", "", false
}

// insertTagLinks 在给定事务内用多行 INSERT 写入标签关联。
// 原先逐条 Create 会产生 N 个独立写事务（各一次 fsync），现在合入外层事务并一次写完。
func insertTagLinks(tx *gorm.DB, table, column string, entityID uint, tagIDs []uint) {
	if len(tagIDs) == 0 {
		return
	}
	for _, part := range chunkIDs(tagIDs, idChunkSize) {
		rows := make([]map[string]interface{}, 0, len(part))
		for _, tid := range part {
			rows = append(rows, map[string]interface{}{column: entityID, "tag_id": tid})
		}
		tx.Table(table).Create(rows)
	}
}

// createTagLinks 追加标签关联（不删旧），单事务完成。
// 与逐条版本的错误处理保持一致：不中断后续语句、不向上报错。
func (h *Handler) createTagLinks(entityType string, entityID uint, tagIDs []uint) {
	table, column, ok := tagLinkTable(entityType)
	if !ok || len(tagIDs) == 0 {
		return
	}
	h.db.Transaction(func(tx *gorm.DB) error {
		insertTagLinks(tx, table, column, entityID, tagIDs)
		return nil
	})
}

// replaceTagLinks 全量替换某实体的标签关联（删旧 + 批量写新），单事务完成。
func (h *Handler) replaceTagLinks(entityType string, entityID uint, tagIDs []uint) {
	table, column, ok := tagLinkTable(entityType)
	if !ok {
		return
	}
	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Exec("DELETE FROM "+table+" WHERE "+column+" = ?", entityID)
		insertTagLinks(tx, table, column, entityID, tagIDs)
		return nil
	})
}

// remountMediaOnUpdate 更新实体时重挂媒体：先解除旧关联再挂新关联，单事务完成。
// 原先逐条 UPDATE 各是一个写事务；这里合并为两条语句（目标值相同且 WHERE 条件互斥，结果一致）。
func (h *Handler) remountMediaOnUpdate(userID uint, entityType string, entityID uint, mediaIDs []uint) {
	h.db.Transaction(func(tx *gorm.DB) error {
		tx.Model(&model.Media{}).
			Where("user_id = ? AND entity_type = ? AND entity_id = ?", userID, entityType, entityID).
			Updates(map[string]interface{}{"entity_type": nil, "entity_id": nil})
		if len(mediaIDs) > 0 {
			tx.Model(&model.Media{}).Where("id IN ? AND user_id = ?", mediaIDs, userID).
				Updates(map[string]interface{}{"entity_type": entityType, "entity_id": entityID})
		}
		return nil
	})
}
