package handler

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// 敏感词分类
const (
	ModCatTerror   = "terror"   // 恐怖
	ModCatViolence = "violence" // 暴力
	ModCatPorn     = "porn"     // 色情
	ModCatPolitics = "politics" // 政治敏感
)

// builtinModerationTerms 内置默认词库（可被管理员自定义词库扩充）。
var builtinModerationTerms = map[string][]string{
	ModCatTerror:   {"恐怖袭击", "炸弹袭击", "爆炸装置", "自杀式袭击", "人肉炸弹", "制造爆炸"},
	ModCatViolence: {"血腥", "杀人碎尸", "砍头", "斩首", "肢解", "枪杀", "灭门", "屠杀", "虐待儿童", "连环杀手", "教唆杀人"},
	ModCatPorn:     {"色情网站", "淫秽视频", "裸聊", "露骨性爱", "出售色情", "色情直播", "招嫖"},
	ModCatPolitics: {}, // 政治类默认交由管理员在词库管理中补充配置
}

// moderationCategoryLabels 类别中文名（供审核页展示）。
var moderationCategoryLabels = map[string]string{
	ModCatTerror:   "恐怖",
	ModCatViolence: "暴力",
	ModCatPorn:     "色情",
	ModCatPolitics: "政治敏感",
}

// moderationCustomTerms 读取管理员自定义词库（settings.moderation_terms，JSON map）。
func (h *Handler) moderationCustomTerms() map[string][]string {
	custom := map[string][]string{}
	raw := h.getSetting(model.SettingModerationTerms, "")
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &custom)
	}
	return custom
}

// moderateText 对文本做敏感词审查，返回是否拦截与命中分类。
func (h *Handler) moderateText(text string) (bool, []string) {
	hit := map[string]bool{}
	merged := map[string][]string{}
	for cat, terms := range builtinModerationTerms {
		merged[cat] = append([]string{}, terms...)
	}
	for cat, terms := range h.moderationCustomTerms() {
		merged[cat] = append(merged[cat], terms...)
	}
	for cat, terms := range merged {
		for _, t := range terms {
			t = strings.TrimSpace(t)
			if t != "" && strings.Contains(text, t) {
				hit[cat] = true
			}
		}
	}
	if len(hit) == 0 {
		return false, nil
	}
	cats := make([]string, 0, len(hit))
	for _, c := range []string{ModCatTerror, ModCatViolence, ModCatPorn, ModCatPolitics} {
		if hit[c] {
			cats = append(cats, c)
		}
	}
	return true, cats
}

// GetModerationQueue 审核队列（待审核内容，按提交时间倒序）。
func (h *Handler) GetModerationQueue(c *gin.Context) {
	cu := currentUser(c)
	var pubs []model.Publication
	if err := h.db.Where("status = ?", model.PubStatusPending).Order("id DESC").Find(&pubs).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	items := make([]gin.H, 0, len(pubs))
	for i := range pubs {
		items = append(items, h.pubItemJSON(&pubs[i], cu.ID))
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

// ApprovePublication 审核通过。
func (h *Handler) ApprovePublication(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var p model.Publication
	if err := h.db.First(&p, id).Error; err != nil {
		notFound(c, "内容不存在")
		return
	}
	if p.Status != model.PubStatusPending {
		fail(c, 400, "该内容不在待审核状态")
		return
	}
	if err := h.db.Model(&model.Publication{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": model.PubStatusPublished, "review_categories": "", "updated_at": model.NowISO(),
	}).Error; err != nil {
		serverError(c, "操作失败")
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// RejectPublication 审核驳回（用户可修改源内容后重新提交）。
func (h *Handler) RejectPublication(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var p model.Publication
	if err := h.db.First(&p, id).Error; err != nil {
		notFound(c, "内容不存在")
		return
	}
	if p.Status != model.PubStatusPending {
		fail(c, 400, "该内容不在待审核状态")
		return
	}
	if err := h.db.Model(&model.Publication{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status": model.PubStatusRejected, "updated_at": model.NowISO(),
	}).Error; err != nil {
		serverError(c, "操作失败")
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// GetModerationTerms 返回内置与自定义词库（审核/管理页面展示）。审核员可读。
func (h *Handler) GetModerationTerms(c *gin.Context) {
	c.JSON(200, gin.H{
		"builtin": builtinModerationTerms,
		"custom":  h.moderationCustomTerms(),
		"labels":  moderationCategoryLabels,
	})
}

// SaveModerationTerms 覆盖保存自定义词库（仅管理员）。
func (h *Handler) SaveModerationTerms(c *gin.Context) {
	var req struct {
		Custom map[string][]string `json:"custom"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	if req.Custom == nil {
		req.Custom = map[string][]string{}
	}
	raw, _ := json.Marshal(req.Custom)
	if err := h.setSetting(model.SettingModerationTerms, string(raw)); err != nil {
		serverError(c, "保存失败")
		return
	}
	c.JSON(200, gin.H{"custom": req.Custom})
}
