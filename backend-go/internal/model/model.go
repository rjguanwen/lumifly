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
	RoleAdmin     = "admin"     // 管理员
	RoleModerator = "moderator" // 内容审核员（可审核广场内容）
	RoleUser      = "user"      // 普通用户
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
	Signature        string  `gorm:"size:255" json:"signature"`             // 个性签名
	CreatedAt        string  `gorm:"size:40" json:"createdAt"`
	UpdatedAt        string  `gorm:"size:40" json:"updatedAt"`
}

// ==================== Books ====================

// 阅读进度
const (
	BookStatusNotStarted = "not_started" // 未开始
	BookStatusReading    = "reading"     // 进行中
	BookStatusFinished   = "finished"    // 已读完
	BookStatusAbandoned  = "abandoned"   // 已中止
)

// Book 读书记录。
type Book struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"index;not null" json:"userId"`
	Name      string `gorm:"size:255;not null" json:"name"`
	Author    string `gorm:"size:128" json:"author"`
	Domain    string `gorm:"size:64" json:"domain"` // 所属领域
	Status    string `gorm:"size:16;default:not_started" json:"status"`
	Rating    int    `gorm:"default:0" json:"rating"` // 星级 0-5
	ReadYear  string `gorm:"size:4" json:"readYear"`  // 阅读年份 YYYY
	Recommend string `gorm:"type:text" json:"recommend"` // 推荐语
	Review    string `gorm:"type:text" json:"review"`    // 书评（HTML）
	CreatedAt string `gorm:"size:40" json:"createdAt"`
	UpdatedAt string `gorm:"size:40" json:"updatedAt"`
}

// BookTag 书籍-标签关联。
type BookTag struct {
	BookID uint `gorm:"primaryKey" json:"bookId"`
	TagID  uint `gorm:"primaryKey" json:"tagId"`
}

// BookDomain 书籍所属领域库（用户级，可维护）。
type BookDomain struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"uniqueIndex:uk_user_domain;not null" json:"userId"`
	Name      string `gorm:"size:32;uniqueIndex:uk_user_domain;not null" json:"name"`
	CreatedAt string `gorm:"size:40" json:"createdAt"`
}

// ==================== 广场 ====================

// 广场内容来源类型
const (
	PubRecord    = "record"    // 日记
	PubMilestone = "milestone" // 大事记
	PubIdea      = "idea"      // 想法灵感
	PubBook      = "book"      // 读书记录（书评）
	PubQuick     = "quick"     // 速记语录
)

// 广场帖发布状态
const (
	PubStatusPublished = "published" // 已通过（公开可见）
	PubStatusPending   = "pending"   // 待人工审核
	PubStatusRejected  = "rejected"  // 未通过审核
)

// Publication 广场发布帖（内容为发布时刻的快照）。
type Publication struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	UserID           uint   `gorm:"index;not null" json:"userId"`
	SourceType       string `gorm:"size:16;index;not null" json:"sourceType"` // record/milestone/idea/book/quick
	SourceID         uint   `gorm:"not null" json:"sourceId"`
	Title            string `gorm:"size:255" json:"title"`
	Preview          string `gorm:"type:text" json:"preview"` // 纯文本预览（搜索/摘要）
	Content          string `gorm:"type:text" json:"content"` // HTML 正文快照
	Author           string `gorm:"size:128" json:"author"`   // 书籍作者等
	Rating           int    `gorm:"default:0" json:"rating"`  // 书籍星级（其他为 0）
	Meta             string `gorm:"type:text" json:"meta"`    // 附加 JSON（领域/年份/进度/推荐语等）
	Status           string `gorm:"size:16;default:published" json:"status"` // published / pending / rejected
	ReviewCategories string `gorm:"size:128" json:"reviewCategories"`        // 命中类别（terror/violence/porn/politics，逗号分隔）
	CreatedAt        string `gorm:"size:40" json:"createdAt"`
	UpdatedAt        string `gorm:"size:40" json:"updatedAt"`
}

// PublicationLike 广场点赞（一人一帖一次）。
type PublicationLike struct {
	ID            uint `gorm:"primaryKey" json:"id"`
	PublicationID uint `gorm:"uniqueIndex:uk_pub_user;not null" json:"publicationId"`
	UserID        uint `gorm:"uniqueIndex:uk_pub_user;not null" json:"userId"`
	CreatedAt     string `gorm:"size:40" json:"createdAt"`
}

// PublicationComment 广场评论。
type PublicationComment struct {
	ID            uint   `gorm:"primaryKey" json:"id"`
	PublicationID uint   `gorm:"index;not null" json:"publicationId"`
	UserID        uint   `gorm:"index;not null" json:"userId"`
	Content       string `gorm:"size:1000;not null" json:"content"`
	CreatedAt     string `gorm:"size:40" json:"createdAt"`
}

// SystemSetting 系统设置（key-value，与用户无关）。
type SystemSetting struct {
	Key   string `gorm:"primaryKey;size:64" json:"key"`
	Value string `gorm:"size:255" json:"value"`
}

// 系统设置键
const (
	SettingRegistrationEnabled = "registration_enabled" // "true"/"false"
	SettingModerationTerms     = "moderation_terms"     // 自定义敏感词库 JSON map[category][]term
)

// ==================== 邀请注册 ====================

// 邀请状态
const (
	InviteStatusPending    = "pending"    // 待接受
	InviteStatusRegistered = "registered" // 已注册（链接已被使用）
	InviteStatusRevoked    = "revoked"    // 已撤销
)

// Invitation 邀请注册记录（管理员邀请指定邮箱注册）。
type Invitation struct {
	ID        uint    `gorm:"primaryKey" json:"id"`
	Email     string  `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Token     string  `gorm:"size:64;uniqueIndex;not null" json:"-"` // 随机邀请令牌（仅存库，不回传）
	InvitedBy uint    `gorm:"not null" json:"invitedBy"`             // 发起邀请的管理员 id
	Status    string  `gorm:"size:16;default:pending" json:"status"` // pending / registered / revoked
	ExpiresAt string  `gorm:"size:40;not null" json:"expiresAt"`
	CreatedAt string  `gorm:"size:40;not null" json:"createdAt"`
	UsedAt    *string `gorm:"size:40" json:"usedAt"` // 实际完成注册的时间
}

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

// ==================== 好友 ====================

const (
	FriendPending  = "pending"  // 待对方处理
	FriendAccepted = "accepted" // 已是好友
)

// Friendship 好友关系（user_id 为申请发起方；双向 pending 时接受即成为好友）。
type Friendship struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"uniqueIndex:uk_user_friend;not null" json:"userId"` // 申请发起方
	FriendID  uint   `gorm:"uniqueIndex:uk_user_friend;not null" json:"friendId"`
	Status    string `gorm:"size:16;default:pending" json:"status"`
	CreatedAt string `gorm:"size:40" json:"createdAt"`
	UpdatedAt string `gorm:"size:40" json:"updatedAt"`
}

// ==================== QuickNotes ====================

// QuickNote 速记语录（≤360 字纯文本）。
type QuickNote struct {
	ID        uint   `gorm:"primaryKey" json:"id"`
	UserID    uint   `gorm:"index;not null" json:"userId"`
	Content   string `gorm:"type:text;not null" json:"content"`
	CreatedAt string `gorm:"size:40" json:"createdAt"`
	UpdatedAt string `gorm:"size:40" json:"updatedAt"`
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

// QuickNoteTag 速记语录-标签关联。
type QuickNoteTag struct {
	QuickNoteID uint `gorm:"primaryKey" json:"quickNoteId"`
	TagID       uint `gorm:"primaryKey" json:"tagId"`
}

// PlanTag 规划-标签关联。
type PlanTag struct {
	PlanID uint `gorm:"primaryKey" json:"planId"`
	TagID  uint `gorm:"primaryKey" json:"tagId"`
}
