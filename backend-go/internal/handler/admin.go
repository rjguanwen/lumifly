package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// ListUsers 用户列表（仅管理员）。
func (h *Handler) ListUsers(c *gin.Context) {
	var users []model.User
	if err := h.db.Order("id ASC").Find(&users).Error; err != nil {
		serverError(c, "查询失败")
		return
	}
	items := make([]gin.H, 0, len(users))
	for i := range users {
		items = append(items, userJSON(&users[i]))
	}
	c.JSON(200, gin.H{"items": items, "total": len(items)})
}

// SetUserActive 停用/启用普通用户（仅管理员）。
// 规则：不能停用管理员；不能停用/启用自己。
func (h *Handler) SetUserActive(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	if uint(id) == cu.ID {
		fail(c, 400, "不能对自己执行此操作")
		return
	}
	var req struct {
		IsActive *bool `json:"isActive"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.IsActive == nil {
		badRequest(c, "缺少 isActive 参数")
		return
	}
	var target model.User
	if err := h.db.First(&target, id).Error; err != nil {
		notFound(c, "用户不存在")
		return
	}
	if target.Role == model.RoleAdmin {
		fail(c, 400, "不能停用管理员账号")
		return
	}
	if err := h.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_active": *req.IsActive,
		"updated_at": model.NowISO(),
	}).Error; err != nil {
		serverError(c, "操作失败")
		return
	}
	h.db.First(&target, id)
	c.JSON(200, userJSON(&target))
}

// SetUserRole 设置用户角色（普通用户/审核员，仅管理员）。
// 规则：管理员角色不可被修改；不能修改自己的角色。
func (h *Handler) SetUserRole(c *gin.Context) {
	cu := currentUser(c)
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		badRequest(c, "无效的 id")
		return
	}
	if uint(id) == cu.ID {
		fail(c, 400, "不能修改自己的角色")
		return
	}
	var req struct {
		Role string `json:"role"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	if req.Role != model.RoleUser && req.Role != model.RoleModerator {
		badRequest(c, "角色只能是 user 或 moderator")
		return
	}
	var target model.User
	if err := h.db.First(&target, id).Error; err != nil {
		notFound(c, "用户不存在")
		return
	}
	if target.Role == model.RoleAdmin {
		fail(c, 400, "管理员账号不能修改角色")
		return
	}
	if err := h.db.Model(&model.User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"role": req.Role, "updated_at": model.NowISO(),
	}).Error; err != nil {
		serverError(c, "操作失败")
		return
	}
	h.db.First(&target, id)
	c.JSON(200, userJSON(&target))
}

// GetSettings 读取系统设置（注册开关等，仅管理员）。
func (h *Handler) GetSettings(c *gin.Context) {
	enabled := h.getSetting(model.SettingRegistrationEnabled, "true") == "true"
	c.JSON(200, gin.H{"registration_enabled": enabled})
}

// SetRegistrationEnabled 设置注册开关（仅管理员）。
func (h *Handler) SetRegistrationEnabled(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Enabled == nil {
		badRequest(c, "缺少 enabled 参数")
		return
	}
	val := "false"
	if *req.Enabled {
		val = "true"
	}
	if err := h.setSetting(model.SettingRegistrationEnabled, val); err != nil {
		serverError(c, "保存失败")
		return
	}
	c.JSON(200, gin.H{"registration_enabled": *req.Enabled})
}
