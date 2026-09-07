package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"lumiflybackend/internal/config"
	"lumiflybackend/internal/middleware"
)

// Handler 持有依赖：数据库、配置、认证。
type Handler struct {
	db       *gorm.DB
	cfg      *config.Config
	auth     *middleware.Auth
	uploadDir string
}

func New(db *gorm.DB, cfg *config.Config, auth *middleware.Auth) *Handler {
	return &Handler{
		db:        db,
		cfg:       cfg,
		auth:      auth,
		uploadDir: config.ResolvePath(cfg.UploadDir),
	}
}

// currentUser 从上下文获取当前登录用户。
func currentUser(c *gin.Context) *middleware.UserContext {
	v, ok := c.Get("user")
	if !ok {
		return nil
	}
	return v.(*middleware.UserContext)
}

// RegisterRoutes 注册全部路由。
func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")

	// 公开接口
	api.POST("/auth/login", h.Login)
	api.POST("/auth/register", h.Register)
	api.POST("/auth/logout", h.Logout)
	api.GET("/auth/forgot", h.GetPasswordRecovery)
	api.POST("/auth/forgot/reset", h.ResetPassword)
	api.POST("/auth/forgot/send", h.SendForgotEmail)
	api.POST("/auth/reset", h.ResetPasswordByToken)
	api.GET("/auth/invite/info", h.InviteInfo)

	// 上传文件静态访问（公开，与旧版一致）
	api.GET("/uploads/*filepath", h.ServeFile)

	// 需要登录
	user := api.Group("", h.auth.RequireUser())
	{
		user.GET("/auth/me", h.Me)
		user.GET("/auth/security", h.GetSecurityInfo)
		user.PUT("/auth/security", h.SetSecurityInfo)
		user.PUT("/auth/password", h.ChangePassword)
		user.PUT("/profile", h.UpdateProfile)
		user.PUT("/profile/avatar", h.UploadAvatar)

		// 记录
		user.GET("/records", h.ListRecords)
		user.POST("/records", h.CreateRecord)
		user.GET("/records/:id", h.GetRecord)
		user.PUT("/records/:id", h.UpdateRecord)
		user.DELETE("/records/:id", h.DeleteRecord)

		// 大事记
		user.GET("/milestones", h.ListMilestones)
		user.POST("/milestones", h.CreateMilestone)
		user.GET("/milestones/:id", h.GetMilestone)
		user.PUT("/milestones/:id", h.UpdateMilestone)
		user.DELETE("/milestones/:id", h.DeleteMilestone)

		// 规划
		user.GET("/plans", h.ListPlans)
		user.POST("/plans", h.CreatePlan)
		user.GET("/plans/:id", h.GetPlan)
		user.PUT("/plans/:id", h.UpdatePlan)
		user.DELETE("/plans/:id", h.DeletePlan)

		// 灵感
		user.GET("/ideas", h.ListIdeas)
		user.POST("/ideas", h.CreateIdea)
		user.GET("/ideas/:id", h.GetIdea)
		user.PUT("/ideas/:id", h.UpdateIdea)
		user.DELETE("/ideas/:id", h.DeleteIdea)

		// 速记语录
		user.GET("/quick-notes", h.ListQuickNotes)
		user.POST("/quick-notes", h.CreateQuickNote)
		user.PUT("/quick-notes/:id", h.UpdateQuickNote)
		user.DELETE("/quick-notes/:id", h.DeleteQuickNote)

		// 好友
		user.GET("/users/search", h.SearchUsers)
		user.GET("/friends", h.ListFriends)
		user.GET("/friends/requests", h.ListFriendRequests)
		user.POST("/friends/requests", h.SendFriendRequest)
		user.DELETE("/friends/requests/:id", h.CancelOrRejectFriendRequest)
		user.POST("/friends/requests/:id/accept", h.AcceptFriendRequest)
		user.DELETE("/friends/:friendId", h.RemoveFriend)

		// 人生日历分享与私密评价
		user.GET("/calendar/shares", h.MyCalendarShares)
		user.POST("/calendar/shares", h.AddCalendarShare)
		user.DELETE("/calendar/shares/:friendId", h.RemoveCalendarShare)
		user.GET("/calendar/shares/received", h.ReceivedCalendarShares)
		user.GET("/calendar/shared/:userId/summary", h.SharedCalendarSummary)
		user.GET("/calendar/shared/:userId/period", h.SharedCalendarPeriod)
		user.GET("/calendar/comments", h.ListCalendarComments)
		user.POST("/calendar/comments", h.AddCalendarComment)
		user.DELETE("/calendar/comments/:id", h.DeleteCalendarComment)
		user.GET("/calendar/comment-center/unread", h.CommentCenterUnread)
		user.GET("/calendar/comment-center", h.CommentCenterList)
		user.POST("/calendar/comment-center/read", h.CommentCenterRead)

		// 标签
		user.GET("/tags", h.ListTags)
		user.POST("/tags", h.CreateTag)

		// 读书记录
		user.GET("/books", h.ListBooks)
		user.POST("/books", h.CreateBook)
		user.GET("/books/:id", h.GetBook)
		user.PUT("/books/:id", h.UpdateBook)
		user.DELETE("/books/:id", h.DeleteBook)

		// 书籍领域库
		user.GET("/book-domains", h.ListBookDomains)
		user.POST("/book-domains", h.CreateBookDomain)
		user.PUT("/book-domains/:id", h.RenameBookDomain)
		user.DELETE("/book-domains/:id", h.DeleteBookDomain)

		// 广场
		user.POST("/publications", h.Publish)
		user.GET("/publications", h.ListSquare)
		user.GET("/publications/mine", h.Mine)
		user.GET("/publications/:id", h.GetSquarePost)
		user.DELETE("/publications/:id", h.Unpublish)
		user.POST("/publications/:id/like", h.ToggleLike)
		user.GET("/publications/:id/comments", h.ListComments)
		user.POST("/publications/:id/comments", h.AddComment)
		user.DELETE("/publications/comment/:id", h.DeleteComment)

		// 日历汇总
		user.GET("/calendar/summary", h.CalendarSummary)

		// 上传
		user.POST("/upload", h.UploadFile)
		user.DELETE("/media/:id", h.DeleteMedia)

		// 菜单红点未读统计
		user.GET("/badges/unread", h.UnreadBadges)

		// 内容审核（审核员/管理员）
		mod := user.Group("/moderation", h.auth.RequireModerator())
		{
			mod.GET("/queue", h.GetModerationQueue)
			mod.POST("/queue/:id/approve", h.ApprovePublication)
			mod.POST("/queue/:id/reject", h.RejectPublication)
			mod.GET("/terms", h.GetModerationTerms)
		}

		// 管理员专属
		admin := user.Group("", h.auth.RequireAdmin())
		{
			admin.GET("/admin/users", h.ListUsers)
			admin.PATCH("/admin/users/:id", h.SetUserActive)
			admin.PATCH("/admin/users/:id/role", h.SetUserRole)
			admin.GET("/admin/settings", h.GetSettings)
			admin.PUT("/admin/settings/registration", h.SetRegistrationEnabled)
			admin.PUT("/admin/moderation/terms", h.SaveModerationTerms)
			// 邀请注册
			admin.POST("/admin/invites", h.CreateInvites)
			admin.GET("/admin/invites", h.ListInvites)
			admin.POST("/admin/invites/:id/revoke", h.RevokeInvite)
		}
	}
}
