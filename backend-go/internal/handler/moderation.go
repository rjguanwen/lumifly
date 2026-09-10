package handler

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

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

// ---------- 词库缓存 ----------
//
// 原先每次发布都要重建合并 map、读一次 system_settings、再逐词 strings.Contains。
// 内置词固定不变，自定义词仅在管理员保存时变化（本应用为单进程），
// 因此缓存装配好的列表，并在 SaveModerationTerms 后主动失效。
// 命中判定与输出顺序保持与原逐次装配一致（输出顺序由下方固定 cats 序列决定，与遍历序无关）。

type moderationTerm struct {
	cat  string
	term string
}

var (
	moderationMu     sync.RWMutex
	moderationCached []moderationTerm
)

// buildModerationTerms 展平内置 + 自定义词库，去除空项与首尾空白。
func buildModerationTerms(custom map[string][]string) []moderationTerm {
	out := make([]moderationTerm, 0, 64)
	for _, cat := range []string{ModCatTerror, ModCatViolence, ModCatPorn, ModCatPolitics} {
		for _, t := range builtinModerationTerms[cat] {
			if t = strings.TrimSpace(t); t != "" {
				out = append(out, moderationTerm{cat: cat, term: t})
			}
		}
		for _, t := range custom[cat] {
			if t = strings.TrimSpace(t); t != "" {
				out = append(out, moderationTerm{cat: cat, term: t})
			}
		}
	}
	// 原逻辑会遍历 merged 的所有类别，包括不在上面固定序列中的自定义分类名
	for cat, terms := range custom {
		if cat == ModCatTerror || cat == ModCatViolence || cat == ModCatPorn || cat == ModCatPolitics {
			continue
		}
		for _, t := range terms {
			if t = strings.TrimSpace(t); t != "" {
				out = append(out, moderationTerm{cat: cat, term: t})
			}
		}
	}
	return out
}

// moderationTerms 取缓存的词库（首次或失效后重装）。
func (h *Handler) moderationTerms() []moderationTerm {
	moderationMu.RLock()
	cached := moderationCached
	moderationMu.RUnlock()
	if cached != nil {
		return cached
	}
	built := buildModerationTerms(h.moderationCustomTerms())
	moderationMu.Lock()
	moderationCached = built
	moderationMu.Unlock()
	return built
}

// invalidateModerationTerms 词库变更后调用，使下次审查重新装配。
func invalidateModerationTerms() {
	moderationMu.Lock()
	moderationCached = nil
	moderationMu.Unlock()
}

// moderateText 对文本做敏感词审查，返回是否拦截与命中分类。
func (h *Handler) moderateText(text string) (bool, []string) {
	hit := map[string]bool{}
	for _, t := range h.moderationTerms() {
		if strings.Contains(text, t.term) {
			hit[t.cat] = true
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
	items := h.pubItemsJSON(pubs, cu.ID)
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
	// 词库已变，使缓存失效（下次审查重新装配）
	invalidateModerationTerms()
	c.JSON(200, gin.H{"custom": req.Custom})
}
