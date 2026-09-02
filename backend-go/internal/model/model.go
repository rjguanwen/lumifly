package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// NowISO 生成与旧版 life-recorder 一致的 UTC ISO8601 时间字符串。
func NowISO() string {
	return time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
}

// HashPassword 生成 bcrypt 密码哈希（兼容旧版 $2b$ 格式）。
func HashPassword(password string) string {
	hashed, _ := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(hashed)
}

// CheckPassword 校验密码（兼容旧版 bcrypt 哈希）。
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// ==================== Users ====================

// 用户角色
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// User 用户。
type User struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	Email            string  `gorm:"size:255;uniqueIndex;not null" json:"email"`
	PasswordHash     string  `gorm:"size:255;not null" json:"-"`
	DisplayName      string  `gorm:"size:128;not null" json:"displayName"`
	BirthDate        *string `gorm:"size:16" json:"birthDate"`
	ExpectedLifespan int     `gorm:"default:80" json:"expectedLifespan"`
	AvatarURL        *string `gorm:"size:255" json:"avatarUrl"`
	IsSetupComplete  bool    `gorm:"default:false" json:"isSetupComplete"`
	Role             string  `gorm:"size:16;default:user" json:"role"`      // admin / user
	IsActive         bool    `gorm:"default:true" json:"isActive"`          // 是否启用（管理员可停用普通用户）
	PasswordHint     string  `gorm:"size:128" json:"-"`                     // 密码提示词
	SecurityQuestion string  `gorm:"size:128" json:"-"`                     // 找回安全问题
	SecurityAnswer   string  `gorm:"size:128" json:"-"`                     // 安全答案（存小写，不返回）
	CreatedAt        string  `gorm:"size:40" json:"createdAt"`
	UpdatedAt        string  `gorm:"size:40" json:"updatedAt"`
}

// SystemSetting 系统设置（key-value，与用户无关）。
type SystemSetting struct {
	Key   string `gorm:"primaryKey;size:64" json:"key"`
	Value string `gorm:"size:255" json:"value"`
}

// 系统设置键
const (
	SettingRegistrationEnabled = "registration_enabled" // "true"/"false"
)

// ==================== Milestones ====================

// Milestone 人生大事记。
type Milestone struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	UserID      uint    `gorm:"index;not null" json:"userId"`
	Title       string  `gorm:"size:255;not null" json:"title"`
	Description string  `gorm:"type:text" json:"description"`
	EventDate   string  `gorm:"size:16;index;not null" json:"eventDate"`
	EndDate     *string `gorm:"size:16" json:"endDate"`
	Category    string  `gorm:"size:32;default:other" json:"category"`
	Importance  int     `gorm:"default:3" json:"importance"`
	CreatedAt   string  `gorm:"size:40" json:"createdAt"`
	UpdatedAt   string  `gorm:"size:40" json:"updatedAt"`
}

// ==================== Records ====================

// Record 日常记录。
type Record struct {
	ID         uint   `gorm:"primaryKey" json:"id"`
	UserID     uint   `gorm:"index;not null" json:"userId"`
	Title      string `gorm:"size:255" json:"title"`
	Content    string `gorm:"type:text" json:"content"`
	RecordDate string `gorm:"size:16;index;not null" json:"recordDate"`
	Mood       string `gorm:"size:16" json:"mood"`
	Weather    string `gorm:"size:32" json:"weather"`
	CreatedAt  string `gorm:"size:40" json:"createdAt"`
	UpdatedAt  string `gorm:"size:40" json:"updatedAt"`
}

// ==================== Plans ====================

// Plan 未来规划。
type Plan struct {
	ID                uint    `gorm:"primaryKey" json:"id"`
	UserID            uint    `gorm:"index;not null" json:"userId"`
	Title             string  `gorm:"size:255;not null" json:"title"`
	Description       string  `gorm:"type:text" json:"description"`
	TargetDate        *string `gorm:"size:16" json:"targetDate"`
	StartDate         *string `gorm:"size:16" json:"startDate"`
	Status            string  `gorm:"size:16;default:not_started" json:"status"`
	Priority          string  `gorm:"size:16;default:medium" json:"priority"`
	LinkedMilestoneID *uint   `gorm:"index" json:"linkedMilestoneId"`
	CreatedAt         string  `gorm:"size:40" json:"createdAt"`
	UpdatedAt         string  `gorm:"size:40" json:"updatedAt"`
}

// ==================== Ideas ====================

// Idea 想法灵感。
type Idea struct {
	ID               uint    `gorm:"primaryKey" json:"id"`
	UserID           uint    `gorm:"index;not null" json:"userId"`
	Title            string  `gorm:"size:255" json:"title"`
	Content          string  `gorm:"type:text" json:"content"`
	LinkedPlanID     *uint   `gorm:"index" json:"linkedPlanId"`
	LinkedMilestoneID *uint  `gorm:"index" json:"linkedMilestoneId"`
	CreatedAt        string  `gorm:"size:40" json:"createdAt"`
	UpdatedAt        string  `gorm:"size:40" json:"updatedAt"`
}

// ==================== Media ====================

// Media 上传的文件（图片/视频）。
type Media struct {
	ID         uint    `gorm:"primaryKey" json:"id"`
	UserID     uint    `gorm:"index;not null" json:"userId"`
	FilePath   string  `gorm:"size:255;not null" json:"filePath"`   // 相对路径 images/xxx.png
	FileName   string  `gorm:"size:255;not null" json:"fileName"`   // 原始文件名
	MimeType   string  `gorm:"size:64;not null" json:"mimeType"`
	FileSize   int64   `gorm:"not null" json:"fileSize"`
	EntityType *string `gorm:"size:16;index" json:"entityType"` // record / milestone / idea
	EntityID   *uint   `gorm:"index" json:"entityId"`
	CreatedAt  string  `gorm:"size:40" json:"createdAt"`
}

// ==================== Tags ====================

// Tag 标签。
type Tag struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	UserID    uint    `gorm:"index;not null" json:"userId"`
	Name      string  `gorm:"size:32;not null" json:"name"`
	Color     *string `gorm:"size:16" json:"color"`
	CreatedAt string  `gorm:"size:40" json:"createdAt"`
}

// ==================== Junction Tables ====================

// RecordTag 记录-标签关联。
type RecordTag struct {
	RecordID uint `gorm:"primaryKey" json:"recordId"`
	TagID    uint `gorm:"primaryKey" json:"tagId"`
}

// MilestoneTag 大事记-标签关联。
type MilestoneTag struct {
	MilestoneID uint `gorm:"primaryKey" json:"milestoneId"`
	TagID       uint `gorm:"primaryKey" json:"tagId"`
}

// IdeaTag 灵感-标签关联。
type IdeaTag struct {
	IdeaID uint `gorm:"primaryKey" json:"ideaId"`
	TagID  uint `gorm:"primaryKey" json:"tagId"`
}

// PlanTag 规划-标签关联。
type PlanTag struct {
	PlanID uint `gorm:"primaryKey" json:"planId"`
	TagID  uint `gorm:"primaryKey" json:"tagId"`
}
