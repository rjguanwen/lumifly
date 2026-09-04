package handler

import (
	"strings"

	"github.com/gin-gonic/gin"

	"lumiflybackend/internal/model"
)

// UpdateProfile 更新个人资料（JSON: displayName/birthDate/expectedLifespan）。
// 设置出生日期后视为完成初始设置。
func (h *Handler) UpdateProfile(c *gin.Context) {
	cu := currentUser(c)
	var req struct {
		DisplayName      *string `json:"displayName"`
		BirthDate        *string `json:"birthDate"`
		ExpectedLifespan *int    `json:"expectedLifespan"`
		Signature        *string `json:"signature"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		badRequest(c, "请求参数有误")
		return
	}
	var u model.User
	if err := h.db.First(&u, cu.ID).Error; err != nil {
		notFound(c, "用户不存在")
		return
	}

	updates := map[string]interface{}{"updated_at": model.NowISO()}
	if req.DisplayName != nil && strings.TrimSpace(*req.DisplayName) != "" {
		updates["display_name"] = strings.TrimSpace(*req.DisplayName)
	}
	if req.BirthDate != nil {
		if strings.TrimSpace(*req.BirthDate) == "" {
			updates["birth_date"] = nil
		} else {
			updates["birth_date"] = *req.BirthDate
			updates["is_setup_complete"] = true
		}
	}
	if req.ExpectedLifespan != nil {
		if *req.ExpectedLifespan < 1 || *req.ExpectedLifespan > 150 {
			badRequest(c, "预期寿命需在 1-150 之间")
			return
		}
		updates["expected_lifespan"] = *req.ExpectedLifespan
	}
	if req.Signature != nil {
		sig := strings.TrimSpace(*req.Signature)
		if len([]rune(sig)) > 255 {
			badRequest(c, "个性签名最多 255 字")
			return
		}
		updates["signature"] = sig
	}
	if err := h.db.Model(&model.User{}).Where("id = ?", cu.ID).Updates(updates).Error; err != nil {
		serverError(c, "保存失败")
		return
	}
	h.db.First(&u, cu.ID)
	c.JSON(200, userJSON(&u))
}
