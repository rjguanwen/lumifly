package handler

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/mail"
	"lumiflybackend/internal/model"
)

// 邀请链接有效期（天）。
const inviteValidDays = 7

var inviteEmailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

// mailConfig 组装 SMTP 配置。
func (h *Handler) mailConfig() mail.Config {
	return mail.Config{
		Host:     h.cfg.SMTPHost,
		Port:     h.cfg.SMTPPort,
		User:     h.cfg.SMTPUser,
		Pass:     h.cfg.SMTPPass,
		From:     h.cfg.SMTPFrom,
		FromName: h.cfg.SMTPFromName,
	}
}

// randomToken 生成 URL 安全的随机邀请令牌。
func randomToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// 极低概率兜底：保证唯一
		return strings.ReplaceAll(model.NowISO(), ":", "-") + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(b)
}

// inviteExpiry 计算邀请过期时间（7 天后）。
func inviteExpiry() string {
	return time.Now().UTC().AddDate(0, 0, inviteValidDays).Format("2006-01-02T15:04:05.000Z")
}

// invitationExpired 判断邀请是否已过期。
func invitationExpired(expiresAt string) bool {
	t, err := time.Parse("2006-01-02T15:04:05.000Z", expiresAt)
	if err != nil {
		return true
	}
	return time.Now().UTC().After(t)
}

// markInviteUsed 将该邮箱处于"待接受"的邀请标记为已使用（注册成功/邮箱已占用时调用）。
func (h *Handler) markInviteUsed(email string) {
	h.db.Model(&model.Invitation{}).
		Where("email = ? AND status = ?", email, model.InviteStatusPending).
		Updates(map[string]interface{}{
			"status":  model.InviteStatusRegistered,
			"used_at": model.NowISO(),
		})
}

// InviteInfo 公开：校验邀请令牌并返回受邀邮箱与邀请人，供注册页预填。
func (h *Handler) InviteInfo(c *gin.Context) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		fail(c, 400, "缺少邀请令牌")
		return
	}
	var inv model.Invitation
	if err := h.db.Where("token = ?", token).First(&inv).Error; err != nil {
		fail(c, 400, "邀请链接无效，请联系管理员")
		return
	}
	if invitationExpired(inv.ExpiresAt) {
		fail(c, 400, "邀请链接已过期，请联系管理员重新邀请")
		return
	}
	switch inv.Status {
	case model.InviteStatusRegistered:
		fail(c, 400, "该邀请已被使用，请直接登录")
		return
	case model.InviteStatusRevoked:
		fail(c, 400, "该邀请已被撤销，请联系管理员")
		return
	}
	inviterName := ""
	var inviter model.User
	if err := h.db.First(&inviter, inv.InvitedBy).Error; err == nil {
		inviterName = inviter.DisplayName
	}
	c.JSON(200, gin.H{
		"email":                inv.Email,
		"invitedByDisplayName": inviterName,
		"expiresAt":            inv.ExpiresAt,
	})
}

// CreateInvites 创建邀请并向指定邮箱发送邀请邮件（仅管理员）。
// body: {"emails": ["a@x.com", "b@x.com"]}
// 对同一邮箱重复邀请会刷新其令牌与有效期（相当于重发）；已注册邮箱会被拒绝。
func (h *Handler) CreateInvites(c *gin.Context) {
	cu := currentUser(c)
	var me model.User
	meName := ""
	if err := h.db.First(&me, cu.ID).Error; err == nil {
		meName = me.DisplayName
	}
	var req struct {
		Emails []string `json:"emails"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Emails) == 0 {
		badRequest(c, "请至少提供一个邮箱")
		return
	}
	if len(req.Emails) > 20 {
		badRequest(c, "单次最多邀请 20 个邮箱")
		return
	}

	smtpCfg := h.mailConfig()
	results := make([]gin.H, 0, len(req.Emails))

	for _, raw := range req.Emails {
		email := strings.ToLower(strings.TrimSpace(raw))
		res := gin.H{"email": email, "ok": false}

		if email == "" || !inviteEmailRe.MatchString(email) {
			res["reason"] = "邮箱格式不正确"
			results = append(results, res)
			continue
		}
		var count int64
		h.db.Model(&model.User{}).Where("email = ?", email).Count(&count)
		if count > 0 {
			res["reason"] = "该邮箱已注册"
			results = append(results, res)
			continue
		}

		token := randomToken()
		expiresAt := inviteExpiry()
		now := model.NowISO()

		// 若已有邀请记录（含已撤销/已过期的），刷新为新的待接受邀请
		var existing model.Invitation
		hasExisting := h.db.Where("email = ?", email).First(&existing).Error == nil
		var snap model.Invitation
		if hasExisting {
			snap = existing
		}

		if hasExisting {
			if err := h.db.Model(&model.Invitation{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
				"email":      email,
				"token":      token,
				"invited_by": cu.ID,
				"status":     model.InviteStatusPending,
				"expires_at": expiresAt,
				"created_at": existing.CreatedAt,
				"used_at":    nil,
			}).Error; err != nil {
				res["reason"] = "保存邀请失败"
				results = append(results, res)
				continue
			}
		} else {
			inv := &model.Invitation{
				Email:     email,
				Token:     token,
				InvitedBy: cu.ID,
				Status:    model.InviteStatusPending,
				ExpiresAt: expiresAt,
				CreatedAt: now,
			}
			if err := h.db.Create(inv).Error; err != nil {
				res["reason"] = "保存邀请失败"
				results = append(results, res)
				continue
			}
		}

		inviteURL := strings.TrimRight(h.cfg.AppBaseURL, "/") + "/register?invite=" + token

		if !smtpCfg.Enabled() {
			// 开发模式：未配置 SMTP，不真实发信，直接返回邀请链接便于本地测试
			log.Println("[邀请注册] 未配置 SMTP，开发模式生成邀请链接：", inviteURL)
			res["ok"] = true
			res["dev"] = true
			res["invite_url"] = inviteURL
			results = append(results, res)
			continue
		}

		subject := "「飞光」邀请你加入"
		body := buildInviteMailHTML(email, meName, inviteURL)
		if err := smtpCfg.Send(email, subject, body); err != nil {
			log.Printf("[邀请注册] 发送邀请邮件失败：%v", err)
			// 回滚：新建的删除记录，已存在的恢复其原快照，避免留下不可达的邀请
			if hasExisting {
				h.db.Model(&model.Invitation{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
					"token":      snap.Token,
					"invited_by": snap.InvitedBy,
					"status":     snap.Status,
					"expires_at": snap.ExpiresAt,
					"created_at": snap.CreatedAt,
					"used_at":    snap.UsedAt,
				})
			} else {
				h.db.Delete(&model.Invitation{}, "token = ?", token)
			}
			res["reason"] = "邮件发送失败，请检查 SMTP 配置"
			results = append(results, res)
			continue
		}

		res["ok"] = true
		results = append(results, res)
	}

	c.JSON(200, gin.H{"results": results})
}

// ListInvites 邀请记录列表（仅管理员）。
func (h *Handler) ListInvites(c *gin.Context) {
	var invites []model.Invitation
	if err := h.db.Order("id DESC").Find(&invites).Error; err != nil {
		serverError(c, "查询失败")
		return
	}

	// 收集邀请人信息
	inviterNames := map[uint]gin.H{}
	seen := map[uint]bool{}
	var uidList []uint
	for _, iv := range invites {
		if !seen[iv.InvitedBy] {
			seen[iv.InvitedBy] = true
			uidList = append(uidList, iv.InvitedBy)
		}
	}
	if len(uidList) > 0 {
		var users []model.User
		if err := h.db.Where("id IN ?", uidList).Find(&users).Error; err == nil {
			for _, u := range users {
				inviterNames[u.ID] = gin.H{"id": u.ID, "email": u.Email, "displayName": u.DisplayName}
			}
		}
	}

	items := make([]gin.H, 0, len(invites))
	for _, iv := range invites {
		item := gin.H{
			"id":        iv.ID,
			"email":     iv.Email,
			"status":    iv.Status,
			"createdAt": iv.CreatedAt,
			"expiresAt": iv.ExpiresAt,
			"usedAt":    iv.UsedAt,
			"expired":   iv.Status == model.InviteStatusPending && invitationExpired(iv.ExpiresAt),
		}
		if n, ok := inviterNames[iv.InvitedBy]; ok {
			item["invitedBy"] = n
		}
		items = append(items, item)
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

// RevokeInvite 撤销一条待接受的邀请（仅管理员）。撤销后该链接立即失效。
func (h *Handler) RevokeInvite(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	var inv model.Invitation
	if err := h.db.First(&inv, id).Error; err != nil {
		notFound(c, "邀请记录不存在")
		return
	}
	if inv.Status != model.InviteStatusPending {
		fail(c, 400, "仅待接受的邀请可以撤销")
		return
	}
	if err := h.db.Model(&model.Invitation{}).Where("id = ?", id).Update("status", model.InviteStatusRevoked).Error; err != nil {
		serverError(c, "操作失败")
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// buildInviteMailHTML 邀请注册邮件正文。
func buildInviteMailHTML(email, inviterName, inviteURL string) string {
	inviter := escapeHTML(inviterName)
	return `<div style="max-width:560px;margin:0 auto;font-family:-apple-system,'Segoe UI',Roboto,'Helvetica Neue',Arial,'PingFang SC','Microsoft YaHei',sans-serif;background:#f4f6fb;padding:24px">
  <div style="background:#fff;border-radius:12px;padding:28px 32px;border:1px solid #e5e7eb">
    <div style="font-size:20px;font-weight:700;color:#1f2937">飞光 · Lumifly</div>
    <p style="color:#6b7280;font-size:13px;margin:4px 0 20px">人生如逆旅，我亦是行人</p>
    <p style="font-size:15px;color:#374151">你好：</p>
    <p style="font-size:15px;color:#374151">` + inviter + ` 邀请你加入「飞光」，创建专属于你的人生记录空间。请点击下方按钮完成注册（链接 7 天内有效，且仅限本邮件送达的邮箱 ` + escapeHTML(email) + ` 使用）：</p>
    <p style="text-align:center;margin:26px 0">
      <a href="` + inviteURL + `" style="display:inline-block;background:#3b5bdb;color:#fff;text-decoration:none;padding:11px 34px;border-radius:8px;font-size:15px">接受邀请并注册</a>
    </p>
    <p style="font-size:13px;color:#6b7280">如果按钮无法点击，请复制以下链接到浏览器打开：</p>
    <p style="font-size:12px;color:#4b5563;word-break:break-all;background:#f3f4f6;border-radius:6px;padding:10px 12px">` + inviteURL + `</p>
    <p style="font-size:13px;color:#9ca3af;margin-top:20px">如非本人操作，请忽略此邮件。</p>
  </div>
</div>`
}
