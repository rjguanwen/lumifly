package handler

import (
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/mail"
	"lumiflybackend/internal/model"
)

// 发送找回邮件的最小间隔（防止刷接口）
var forgotSendLocks sync.Map

// userJSON 输出用户（不含密码）。
func userJSON(u *model.User) gin.H {
	return gin.H{
		"id":               u.ID,
		"email":            u.Email,
		"displayName":      u.DisplayName,
		"birthDate":        u.BirthDate,
		"expectedLifespan": u.ExpectedLifespan,
		"avatarUrl":        u.AvatarURL,
		"isSetupComplete":  u.IsSetupComplete,
		"role":             u.Role,
		"isActive":         u.IsActive,
	}
}

// getSetting / setSetting 读写系统设置（与用户无关）。
func (h *Handler) getSetting(key, def string) string {
	var s model.SystemSetting
	if err := h.db.Where("key = ?", key).First(&s).Error; err != nil {
		return def
	}
	return s.Value
}

func (h *Handler) setSetting(key, value string) error {
	return h.db.Save(&model.SystemSetting{Key: key, Value: value}).Error
}

// Login 登录（JSON: email/password）。
func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Password == "" {
		badRequest(c, "请输入邮箱和密码")
		return
	}
	var u model.User
	if err := h.db.Where("email = ?", email).First(&u).Error; err != nil {
		unauthorized(c, "邮箱或密码错误")
		return
	}
	if !model.CheckPassword(u.PasswordHash, req.Password) {
		unauthorized(c, "邮箱或密码错误")
		return
	}
	if !u.IsActive {
		fail(c, 403, "账号已被停用，请联系管理员")
		return
	}
	token, err := h.auth.CreateToken(u.ID, u.Email)
	if err != nil {
		serverError(c, "生成凭证失败")
		return
	}
	c.JSON(200, gin.H{"token": token, "user": userJSON(&u)})
}

// Register 注册（JSON: email/password/displayName）。注册开关关闭时拒绝。
func (h *Handler) Register(c *gin.Context) {
	if h.getSetting(model.SettingRegistrationEnabled, "true") != "true" {
		fail(c, 403, "系统已暂停新用户注册，请联系管理员")
		return
	}
	var req struct {
		Email       string `json:"email"`
		Password    string `json:"password"`
		DisplayName string `json:"displayName"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.DisplayName = strings.TrimSpace(req.DisplayName)

	if req.Email == "" || req.Password == "" || req.DisplayName == "" {
		badRequest(c, "邮箱、密码和昵称均为必填")
		return
	}
	if len(req.Password) < 8 {
		badRequest(c, "密码至少需要 8 个字符")
		return
	}
	emailRe := regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)
	if !emailRe.MatchString(req.Email) {
		badRequest(c, "邮箱格式不正确")
		return
	}
	var count int64
	h.db.Model(&model.User{}).Where("email = ?", req.Email).Count(&count)
	if count > 0 {
		fail(c, 409, "该邮箱已注册")
		return
	}

	now := model.NowISO()
	u := &model.User{
		Email:            req.Email,
		PasswordHash:     model.HashPassword(req.Password),
		DisplayName:      req.DisplayName,
		ExpectedLifespan: 80,
		Role:             model.RoleUser,
		IsActive:         true,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	if err := h.db.Create(u).Error; err != nil {
		serverError(c, "注册失败，请稍后重试")
		return
	}
	token, err := h.auth.CreateToken(u.ID, u.Email)
	if err != nil {
		serverError(c, "生成凭证失败")
		return
	}
	c.JSON(201, gin.H{"token": token, "user": userJSON(u)})
}

// Me 当前登录用户信息。
func (h *Handler) Me(c *gin.Context) {
	cu := currentUser(c)
	var u model.User
	if err := h.db.First(&u, cu.ID).Error; err != nil {
		notFound(c, "用户不存在")
		return
	}
	c.JSON(200, userJSON(&u))
}

// Logout 退出登录（无状态 JWT，前端清除本地 token 即可）。
func (h *Handler) Logout(c *gin.Context) {
	c.JSON(200, gin.H{"success": true})
}

// ChangePassword 修改自己的密码（JSON: oldPassword/newPassword）。
func (h *Handler) ChangePassword(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		badRequest(c, "请填写原密码与新密码")
		return
	}
	if len(req.NewPassword) < 8 {
		badRequest(c, "新密码至少需要 8 个字符")
		return
	}
	var u model.User
	if err := h.db.First(&u, cu.ID).Error; err != nil {
		notFound(c, "用户不存在")
		return
	}
	if !model.CheckPassword(u.PasswordHash, req.OldPassword) {
		fail(c, 400, "原密码不正确")
		return
	}
	if model.CheckPassword(u.PasswordHash, req.NewPassword) {
		badRequest(c, "新密码不能与原密码相同")
		return
	}
	updates := map[string]interface{}{
		"password_hash": model.HashPassword(req.NewPassword),
		"updated_at":    model.NowISO(),
	}
	if err := h.db.Model(&model.User{}).Where("id = ?", cu.ID).Updates(updates).Error; err != nil {
		serverError(c, "密码修改失败")
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// ==================== 密码提示与问答找回 ====================

// GetSecurityInfo 读取自己的安全设置（不含答案）。
func (h *Handler) GetSecurityInfo(c *gin.Context) {
	cu := currentUser(c)
	var u model.User
	if err := h.db.First(&u, cu.ID).Error; err != nil {
		notFound(c, "用户不存在")
		return
	}
	c.JSON(200, gin.H{
		"password_hint":     u.PasswordHint,
		"security_question": u.SecurityQuestion,
		"has_security":      u.SecurityAnswer != "",
	})
}

// SetSecurityInfo 设置/清除密码提示与安全问答（支持部分更新）。
//  - passwordHint：仅更新提示词。
//  - 显式传入 securityAnswer 时：若与 securityQuestion 同时非空则保存问答；
//    若均为空则清除已保存的安全问答（不允许只传其一）。
func (h *Handler) SetSecurityInfo(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		PasswordHint     *string `json:"passwordHint"`
		SecurityQuestion *string `json:"securityQuestion"`
		SecurityAnswer   *string `json:"securityAnswer"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	updates := map[string]interface{}{"updated_at": model.NowISO()}
	if req.PasswordHint != nil {
		updates["password_hint"] = strings.TrimSpace(*req.PasswordHint)
	}
	if req.SecurityAnswer != nil {
		question := ""
		if req.SecurityQuestion != nil {
			question = strings.TrimSpace(*req.SecurityQuestion)
		}
		answer := strings.ToLower(strings.TrimSpace(*req.SecurityAnswer))
		if question == "" && answer != "" {
			badRequest(c, "请选择安全问题")
			return
		}
		updates["security_question"] = question
		updates["security_answer"] = answer
	}
	if len(updates) == 1 {
		badRequest(c, "没有需要保存的内容")
		return
	}
	if err := h.db.Model(&model.User{}).Where("id = ?", cu.ID).Updates(updates).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// GetPasswordRecovery 公开：根据邮箱返回密码提示词与安全问题（不含答案）。
func (h *Handler) GetPasswordRecovery(c *gin.Context) {
	email := strings.ToLower(strings.TrimSpace(c.Query("email")))
	if email == "" {
		badRequest(c, "请输入邮箱")
		return
	}
	var u model.User
	if err := h.db.Where("email = ?", email).First(&u).Error; err != nil {
		notFound(c, "该邮箱未注册")
		return
	}
	c.JSON(200, gin.H{
		"exists":            true,
		"password_hint":     u.PasswordHint,
		"security_question": u.SecurityQuestion,
		"has_security":      u.SecurityAnswer != "",
	})
}

// ResetPassword 公开：通过安全问题答案重置密码。
func (h *Handler) ResetPassword(c *gin.Context) {
	var req struct {
		Email       string `json:"email"`
		Answer      string `json:"answer"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" || req.Answer == "" || req.NewPassword == "" {
		badRequest(c, "请填写邮箱、答案和新密码")
		return
	}
	if len(req.NewPassword) < 8 {
		badRequest(c, "新密码至少需要 8 个字符")
		return
	}
	var u model.User
	if err := h.db.Where("email = ?", email).First(&u).Error; err != nil {
		notFound(c, "该邮箱未注册")
		return
	}
	if u.SecurityAnswer == "" || u.SecurityQuestion == "" {
		fail(c, 400, "该账号未设置安全问题，无法通过问答找回密码；请联系管理员")
		return
	}
	if strings.ToLower(strings.TrimSpace(req.Answer)) != u.SecurityAnswer {
		fail(c, 400, "安全问题回答不正确")
		return
	}
	if err := h.db.Model(&model.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
		"password_hash": model.HashPassword(req.NewPassword),
		"updated_at":    model.NowISO(),
	}).Error; err != nil {
		serverError(c, "重置失败")
		return
	}
	c.JSON(200, gin.H{"success": true})
}

// ==================== 邮箱自助找回 ====================

// SendForgotEmail 公开：向注册邮箱发送密码重置链接（有效期 30 分钟）。
// 未配置 SMTP 时进入开发模式：直接返回重置令牌供本地测试。
func (h *Handler) SendForgotEmail(c *gin.Context) {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		badRequest(c, "请输入邮箱")
		return
	}

	// 频率限制：同一邮箱 60 秒内只能发送一次
	if last, ok := forgotSendLocks.Load(email); ok {
		if t, ok2 := last.(time.Time); ok2 && time.Since(t) < time.Minute {
			fail(c, 429, "已发送过，请 1 分钟后再试")
			return
		}
	}

	var u model.User
	if err := h.db.Where("email = ?", email).First(&u).Error; err != nil {
		notFound(c, "该邮箱未注册")
		return
	}
	if !u.IsActive {
		fail(c, 403, "账号已被停用，请联系管理员")
		return
	}

	token, err := h.auth.SignResetToken(u.Email)
	if err != nil {
		serverError(c, "生成重置凭证失败")
		return
	}
	resetURL := strings.TrimRight(h.cfg.AppBaseURL, "/") + "/reset-password?token=" + token
	forgotSendLocks.Store(email, time.Now())

	smtpCfg := mail.Config{
		Host:     h.cfg.SMTPHost,
		Port:     h.cfg.SMTPPort,
		User:     h.cfg.SMTPUser,
		Pass:     h.cfg.SMTPPass,
		From:     h.cfg.SMTPFrom,
		FromName: h.cfg.SMTPFromName,
	}

	if !smtpCfg.Enabled() {
		// 开发模式：邮件功能未配置，直接返回令牌，便于本地联调
		log.Println("[找回密码] 未配置 SMTP，开发模式生成重置链接：", resetURL)
		c.JSON(200, gin.H{"sent": false, "dev": true, "reset_url": resetURL})
		return
	}

	subject := "「飞光」密码重置"
	body := buildResetMailHTML(u.DisplayName, resetURL)
	if err := smtpCfg.Send(u.Email, subject, body); err != nil {
		log.Printf("[找回密码] 发送邮件失败：%v", err)
		serverError(c, "邮件发送失败，请稍后重试或联系管理员")
		return
	}
	c.JSON(200, gin.H{"sent": true})
}

// ResetPasswordByToken 公开：使用重置令牌设置新密码。
func (h *Handler) ResetPasswordByToken(c *gin.Context) {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	if req.Token == "" || req.NewPassword == "" {
		badRequest(c, "缺少令牌或新密码")
		return
	}
	if len(req.NewPassword) < 8 {
		badRequest(c, "新密码至少需要 8 个字符")
		return
	}
	email, err := h.auth.VerifyResetToken(req.Token)
	if err != nil {
		fail(c, 400, "重置链接无效或已过期，请重新发起找回")
		return
	}
	var u model.User
	if err := h.db.Where("email = ?", email).First(&u).Error; err != nil {
		notFound(c, "账号不存在")
		return
	}
	if !u.IsActive {
		fail(c, 403, "账号已被停用，请联系管理员")
		return
	}
	if model.CheckPassword(u.PasswordHash, req.NewPassword) {
		badRequest(c, "新密码不能与原密码相同")
		return
	}
	if err := h.db.Model(&model.User{}).Where("id = ?", u.ID).Updates(map[string]interface{}{
		"password_hash": model.HashPassword(req.NewPassword),
		"updated_at":    model.NowISO(),
	}).Error; err != nil {
		serverError(c, "重置失败")
		return
	}
	c.JSON(200, gin.H{"success": true})
}

func buildResetMailHTML(displayName, resetURL string) string {
	name := escapeHTML(displayName)
	return `<div style="max-width:560px;margin:0 auto;font-family:-apple-system,'Segoe UI',Roboto,'Helvetica Neue',Arial,'PingFang SC','Microsoft YaHei',sans-serif;background:#f4f6fb;padding:24px">
  <div style="background:#fff;border-radius:12px;padding:28px 32px;border:1px solid #e5e7eb">
    <div style="font-size:20px;font-weight:700;color:#1f2937">飞光 · Lumifly</div>
    <p style="color:#6b7280;font-size:13px;margin:4px 0 20px">人生如逆旅，我亦是行人</p>
    <p style="font-size:15px;color:#374151">你好，` + name + `：</p>
    <p style="font-size:15px;color:#374151">我们收到了你的密码重置申请。请点击下方按钮，在 30 分钟内完成密码重置：</p>
    <p style="text-align:center;margin:26px 0">
      <a href="` + resetURL + `" style="display:inline-block;background:#3b5bdb;color:#fff;text-decoration:none;padding:11px 34px;border-radius:8px;font-size:15px">重置密码</a>
    </p>
    <p style="font-size:13px;color:#6b7280">如果按钮无法点击，请复制以下链接到浏览器打开：</p>
    <p style="font-size:12px;color:#4b5563;word-break:break-all;background:#f3f4f6;border-radius:6px;padding:10px 12px">` + resetURL + `</p>
    <p style="font-size:13px;color:#9ca3af;margin-top:20px">如非本人操作，请忽略此邮件，你的密码不会发生变化。</p>
  </div>
</div>`
}

func escapeHTML(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;")
	return r.Replace(s)
}
