package router

import (
	"abingblog-backend/internal/handler"
	"abingblog-backend/internal/middleware"
	"abingblog-backend/internal/repository"

	"github.com/gin-gonic/gin"
)

// Setup 注册所有路由。路由分组即权限边界：
//   - /api/v1 公共接口：游客可用，只暴露已发布内容
//   - /api/v1/admin 后台接口：挂 JWT 中间件，未登录一律 401
//
// CORS 中间件挂在最外层（gin.Default 之后、路由分组之前），保证预检 OPTIONS
// 和所有响应都带上跨域头；allowOrigins 是前端 dev server 白名单。
func Setup(h *handler.Handler, secret string, allowOrigins []string, visitRepo *repository.VisitLogRepo) *gin.Engine {
	r := gin.Default()
	r.Use(middleware.Cors(allowOrigins))

	v1 := r.Group("/api/v1")
	// 公共接口挂访问记录中间件：自动记录游客行为（admin/healthz 在中间件内部跳过）
	v1.Use(middleware.VisitTracker(visitRepo))
	{
		v1.GET("/articles", h.Article.List)
		v1.GET("/articles/:id", h.Article.Get)
		v1.GET("/categories", h.Category.List)
		v1.GET("/tags", h.Tag.List)
		v1.GET("/projects", h.Project.List)
		v1.GET("/profile", h.Profile.Get)
		// 简历：公开只读元信息 + 下载本体，上传/删除在后台
		v1.GET("/resume", h.Resume.Meta)
		v1.GET("/resume/file", h.Resume.File)
		v1.GET("/healthz", h.Health.Check)
		v1.GET("/visits", h.Metric.Total)
		v1.POST("/visits/start", h.Metric.RecordStart)
		v1.POST("/login", h.User.Login)

		// 友链：前台只读上架列表；访客凭 X-Visitor-ID 自助管理自己的提交
		v1.GET("/friend-links", h.FriendLink.List)
		v1.GET("/friend-links/mine", h.FriendLink.Mine)
		v1.POST("/friend-links/mine", h.FriendLink.Create)
		v1.PUT("/friend-links/mine/:id", h.FriendLink.UpdateMine)
		v1.DELETE("/friend-links/mine/:id", h.FriendLink.DeleteMine)
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthRequired(secret))
		{
			admin.GET("/articles", h.Article.AdminList)
			admin.GET("/articles/:id", h.Article.AdminGet)
			admin.POST("/articles", h.Article.Create)
			admin.PUT("/articles/:id", h.Article.Update)
			admin.DELETE("/articles/:id", h.Article.Delete)
			admin.PUT("/articles/:id/status", h.Article.UpdateStatus)

			admin.POST("/categories", h.Category.Create)
			admin.PUT("/categories/:id", h.Category.Update)
			admin.DELETE("/categories/:id", h.Category.Delete)

			admin.POST("/tags", h.Tag.Create)
			admin.PUT("/tags/:id", h.Tag.Update)
			admin.DELETE("/tags/:id", h.Tag.Delete)

			admin.GET("/projects", h.Project.AdminList)
			admin.POST("/projects", h.Project.Create)
			admin.PUT("/projects/:id", h.Project.Update)
			admin.DELETE("/projects/:id", h.Project.Delete)

			// 友链审核：PUT 时 body 带 status（1 通过 / 2 驳回）即为审核动作
			admin.GET("/friend-links", h.FriendLink.AdminList)
			admin.PUT("/friend-links/:id", h.FriendLink.AdminUpdate)
			admin.DELETE("/friend-links/:id", h.FriendLink.AdminDelete)

			admin.GET("/visits", h.Visit.List)
			admin.GET("/visits/summary", h.Visit.Summary)

			// 个人档案：公开接口只读，写操作必须在后台（挂 JWT）
			admin.GET("/profile", h.Profile.AdminGet)
			admin.PUT("/profile", h.Profile.Update)

			// 简历文件：POST 上传/覆盖（multipart，字段名 file），DELETE 删除
			admin.POST("/resume", h.Resume.Upload)
			admin.DELETE("/resume", h.Resume.Delete)
		}
	}
	return r
}
