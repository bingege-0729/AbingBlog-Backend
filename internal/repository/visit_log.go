package repository

import (
	"time"

	"abingblog-backend/internal/model"

	"gorm.io/gorm"
)

// VisitLogRepo 访问记录数据访问层：只负责 SQL，不含业务逻辑
type VisitLogRepo struct {
	db *gorm.DB
}

func NewVisitLogRepo(db *gorm.DB) *VisitLogRepo { return &VisitLogRepo{db: db} }

// Create 写入一条访问记录（由中间件异步调用）
func (r *VisitLogRepo) Create(l *model.VisitLog) error {
	return r.db.Create(l).Error
}

// VisitQuery 明细列表查询条件
type VisitQuery struct {
	Page     int
	PageSize int
	Keyword  string // 模糊匹配昵称或路径
}

// List 分页查访问明细，按时间倒序
func (r *VisitLogRepo) List(q VisitQuery) ([]model.VisitLog, int64, error) {
	query := r.db.Model(&model.VisitLog{})
	if q.Keyword != "" {
		kw := "%" + q.Keyword + "%"
		query = query.Where("nickname LIKE ? OR path LIKE ?", kw, kw)
	}
	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	list := make([]model.VisitLog, 0)
	err := query.Session(&gorm.Session{}).
		Order("created_at DESC, id DESC").
		Offset((q.Page - 1) * q.PageSize).Limit(q.PageSize).
		Find(&list).Error
	return list, total, err
}

// VisitorSummary 按访客聚合的一行（最近访客列表用）
type VisitorSummary struct {
	VisitorKey string    `json:"visitor_key"`
	Nickname   string    `json:"nickname"`
	Device     string    `json:"device"`
	UserAgent  string    `json:"user_agent"`
	IP         string    `json:"ip"`
	Visits     int64     `json:"visits"`
	LastSeenAt time.Time `json:"last_seen_at"`
}

// SummarizeByVisitor 按访客聚合：每个 visitor_key 的访问次数、最近一次时间、设备、IP
func (r *VisitLogRepo) SummarizeByVisitor(limit int) ([]VisitorSummary, error) {
	out := make([]VisitorSummary, 0)
	err := r.db.Model(&model.VisitLog{}).
		Select("visitor_key, MAX(nickname) AS nickname, MAX(device) AS device, MAX(user_agent) AS user_agent, MAX(ip) AS ip, COUNT(*) AS visits, MAX(created_at) AS last_seen_at").
		Group("visitor_key").
		Order("last_seen_at DESC").
		Limit(limit).
		Find(&out).Error
	return out, err
}

// TopPath 最常访问的接口
type TopPath struct {
	Path   string `json:"path"`
	Visits int64  `json:"visits"`
}

// TopPaths 访问次数 TOP N 接口
func (r *VisitLogRepo) TopPaths(limit int) ([]TopPath, error) {
	out := make([]TopPath, 0)
	err := r.db.Model(&model.VisitLog{}).
		Select("path, COUNT(*) AS visits").
		Group("path").
		Order("visits DESC").
		Limit(limit).
		Find(&out).Error
	return out, err
}

// VisitStats 汇总数字
type VisitStats struct {
	Total         int64 `json:"total"`          // 累计访问次数
	Today         int64 `json:"today"`          // 今日访问次数
	DistinctUsers int64 `json:"distinct_users"` // 累计独立访客
	TodayDistinct int64 `json:"today_distinct"` // 今日独立访客
}

// Stats 汇总统计
func (r *VisitLogRepo) Stats() (VisitStats, error) {
	var s VisitStats
	today0 := time.Now().Format("2006-01-02") // Go 的日期格式就是 2006-01-02，不是 yyyy-MM-dd
	// 一次查询算完四个指标。不能对同一个 struct 连续 Scan 两次：
	// GORM 的 Scan 在 ScanInitialized 模式下会先把目标结构体清零再写入，
	// 第二次 Scan 会把第一次写进去的 total/distinct_users 清回 0。
	err := r.db.Model(&model.VisitLog{}).Select(
		"COUNT(*) AS total",
		"COUNT(DISTINCT visitor_key) AS distinct_users",
		"COUNT(CASE WHEN created_at >= ? THEN 1 END) AS today",
		"COUNT(DISTINCT CASE WHEN created_at >= ? THEN visitor_key END) AS today_distinct",
		today0, today0,
	).Scan(&s).Error
	return s, err
}

// DistinctUsers 独立访客总数（公共 /visits 接口的 distinct_users 用，表空返回 0）
func (r *VisitLogRepo) DistinctUsers() (int64, error) {
	var n int64
	err := r.db.Model(&model.VisitLog{}).Distinct("visitor_key").Count(&n).Error
	return n, err
}
