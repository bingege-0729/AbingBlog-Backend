package handler

// Handler 聚合所有接口处理器，由 main 统一组装依赖后传给 router
type Handler struct {
	Article    *ArticleHandler
	Category   *CategoryHandler
	Tag        *TagHandler
	User       *UserHandler
	Health     *HealthHandler
	Metric     *SiteMetricHandler
	Project    *ProjectHandler
	FriendLink *FriendLinkHandler
	Visit      *VisitLogHandler
	Profile    *ProfileHandler
	Resume     *ResumeHandler
}
