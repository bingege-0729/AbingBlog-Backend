package main

import (
	"abingblog-backend/internal/cache"
	"fmt"
	"log"
	"time"
	_ "time/tzdata" // 嵌入 IANA 时区数据：极简容器里没有 tzdata 时 LoadLocation 也能用

	"abingblog-backend/internal/config"
	"abingblog-backend/internal/database"
	"abingblog-backend/internal/handler"
	"abingblog-backend/internal/repository"
	"abingblog-backend/internal/router"
	"abingblog-backend/internal/service"
)

func main() {
	// 启动配置
	cfg := config.Load()

	// 校准应用时区：容器默认 UTC，博客面向中文用户，统一用配置的时区（默认 Asia/Shanghai）。
	// 影响：时间写入/读出、JSON 序列化偏移，以及「今日」统计边界（time.Now 的日期）。
	setTimeZone(cfg.Server.Timezone)

	cache.Init(cfg)
	db := database.Init(cfg) // 连接 MySQL（自动建库建表）

	// 依赖注入，调用链：HTTP 请求 → handler → service → repository → GORM → MySQL
	articleRepo := repository.NewArticleRepo(db)
	categoryRepo := repository.NewCategoryRepo(db)
	tagRepo := repository.NewTagRepo(db)
	userRepo := repository.NewUserRepo(db)
	metricRepo := repository.NewSiteMetricRepo(db)
	projectRepo := repository.NewProjectRepo(db)
	friendLinkRepo := repository.NewFriendLinkRepo(db)
	visitLogRepo := repository.NewVisitLogRepo(db)
	profileRepo := repository.NewProfileRepo(db)
	resumeRepo := repository.NewResumeRepo(db)

	userSvc := service.NewUserService(userRepo, cfg)
	// 启动时确保初始管理员存在（账号已存在则跳过）
	if err := userSvc.SeedAdmin(); err != nil {
		log.Fatalf("初始化管理员失败: %v", err)
	}

	profileSvc := service.NewProfileService(profileRepo)
	// 首次启动写入一份初始档案：后台打开就是有内容的表单，而不是空白
	if err := profileSvc.EnsureSeed(); err != nil {
		log.Printf("初始化个人档案失败（不影响启动）: %v", err)
	}

	// 拦截链启动
	h := &handler.Handler{
		Article:    handler.NewArticleHandler(service.NewArticleService(articleRepo, categoryRepo, tagRepo)),
		Category:   handler.NewCategoryHandler(service.NewCategoryService(categoryRepo)),
		Tag:        handler.NewTagHandler(service.NewTagService(tagRepo)),
		User:       handler.NewUserHandler(userSvc),
		Health:     handler.NewHealthHandler(db),
		Metric:     handler.NewSiteMetricHandler(service.NewSiteMetricService(metricRepo, visitLogRepo)),
		Project:    handler.NewProjectHandler(service.NewProjectService(projectRepo)),
		FriendLink: handler.NewFriendLinkHandler(service.NewFriendLinkService(friendLinkRepo)),
		Visit:      handler.NewVisitLogHandler(service.NewVisitLogService(visitLogRepo)),
		Profile:    handler.NewProfileHandler(profileSvc),
		Resume:     handler.NewResumeHandler(service.NewResumeService(resumeRepo, cfg.Upload.Dir)),
	}

	// 启动
	r := router.Setup(h, cfg.JWT.Secret, cfg.CORS.AllowOrigins, visitLogRepo)
	log.Printf("server running at http://localhost:%d", cfg.Server.Port)
	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		log.Fatal(err)
	}
}

// setTimeZone 把进程的本地时区设成指定值；空值/加载失败时回退到系统默认。
// 必须放在 database.Init 之前调用，这样 DSN 里 loc=Local 才能解析成目标时区。
func setTimeZone(name string) {
	if name == "" {
		name = "Asia/Shanghai"
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		log.Printf("加载时区 %q 失败，回退到系统时区: %v", name, err)
		return
	}
	time.Local = loc
}
