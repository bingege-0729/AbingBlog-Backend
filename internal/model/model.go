package model

import (
	"time"

	"gorm.io/gorm"
)

// User 管理员用户（JWT 迭代时启用，先建表备用）。博客只有站长一个用户
type User struct {
	ID        uint   `gorm:"primaryKey"`
	Username  string `gorm:"size:64;uniqueIndex"`
	Password  string `gorm:"size:255"` // bcrypt 哈希，绝不存明文
	CreatedAt time.Time
}

// SiteMetric 保存站点级运行指标；ID 固定为 1，便于原子递增。
type SiteMetric struct {
	ID         uint `gorm:"primaryKey"`
	StartCount uint `gorm:"not null;default:0"`
}

// Category 文章分类
type Category struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:64;uniqueIndex"`
	Slug      string `gorm:"size:64;uniqueIndex"` // URL 友好的英文标识，用于列表筛选
	Sort      int    `gorm:"default:0"`           // 排序权重，越小越靠前
	CreatedAt time.Time
}

// Tag 文章标签（与文章多对多，GORM 自动生成 article_tags 中间表）
type Tag struct {
	ID        uint   `gorm:"primaryKey"`
	Name      string `gorm:"size:64;uniqueIndex"`
	CreatedAt time.Time
}

// Article 文章：正文存 Markdown 原文，由前端负责渲染
type Article struct {
	ID          uint       `gorm:"primaryKey"`
	Title       string     `gorm:"size:255"`
	Content     string     `gorm:"type:longtext"` // Markdown 原文
	Summary     string     `gorm:"size:500"`
	Cover       string     `gorm:"size:500"`
	URL         string     `gorm:"size:500"` // 外链：发布到掘金/知乎等平台的原文地址（可空）
	CategoryID  uint       // 0 表示未分类
	Category    *Category  // Preload 时填充
	Tags        []Tag      `gorm:"many2many:article_tags;"`
	Status      string     `gorm:"size:16;default:draft"` // draft 草稿 / published 已发布
	ViewCount   int        `gorm:"default:0"`
	PublishedAt *time.Time // 首次发布时间
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"` // 软删除：DELETE 只打标记，数据可恢复
}

// Comment 评论:对于文章或者是博客的评论，或者游水发言
type Comment struct {
	ID        uint      `gorm:"primary_key;AUTO_INCREMENT"`
	ParentID  *uint     `gorm:"index;default null"` //为nil时表示顶级评论
	Author    string    `gorm:"type:varchar(64);not null"`
	Content   string    `gorm:"type:text;not null"`
	CreatedAt time.Time `gorm:"index"`
}

// Project 作品/项目：门户「SELECTED WORK」列表的数据源。
// 结构对齐前端 ProjectSummary（slug/name/detail/stack/githubUrl/demoUrl），
// 复用 article 的 status（draft/published）与 sort 排序约定。
type Project struct {
	ID        uint   `gorm:"primaryKey"`
	Slug      string `gorm:"size:64;uniqueIndex"` // URL 友好标识，前端用作 key
	Name      string `gorm:"size:128"`
	Detail    string `gorm:"size:500"` // 一句话简介
	Stack     string `gorm:"size:255"` // 技术栈展示串，如 "GO · VUE · MYSQL"
	GithubURL string `gorm:"size:500"`
	DemoURL   string `gorm:"size:500"`
	Sort      int    `gorm:"default:0"`             // 排序权重，越小越靠前
	Status    string `gorm:"size:16;default:draft"` // draft 草稿 / published 已发布
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"` // 软删除
}

// FriendLink 友链：访客可自助提交（按 X-Visitor-ID 认领归属），管理员审核后才展示。
// Status 状态机：0 待审核 → 1 已上架 / 2 已驳回；访客修改自己的友链会打回 0 重新审核。
const (
	FriendLinkPending  = 0 // 待审核（提交后的初始状态）
	FriendLinkApproved = 1 // 已上架（前台可见）
	FriendLinkRejected = 2 // 已驳回（管理员否决，前台不可见）
)

type FriendLink struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"size:64"`           // 站点名称
	Avatar      string         `json:"avatar" gorm:"size:500"`        // 头像地址
	URL         string         `json:"url" gorm:"size:500"`           // 站点地址
	Description string         `json:"description" gorm:"size:255"`   // 一句话简介
	Status      int            `json:"status" gorm:"default:0"`       // 状态机，见上方常量
	Sort        int            `json:"sort" gorm:"default:0"`         // 排序权重，越小越靠前（管理员上架时调整）
	OwnerID     string         `json:"owner_id" gorm:"size:64;index"` // 提交者访客 UUID（X-Visitor-ID）
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"` // 软删除：访客删除只打标记
}

// Profile 个人档案：单行表（ID 固定为 1），给门户右侧「档案 / 技术栈 / 近况」三张卡供数。
// Stacks / Awards / Interests 是短字符串列表，用 JSON 列存（GORM serializer），
// 比为几个短列表再建三张关联表划算：它们总是整组读写，不需要单独查询或排序。
type Profile struct {
	ID        uint     `gorm:"primaryKey"`
	Headline  string   `gorm:"size:60"`                   // 一句话定位，如「后端开发 · AI 应用方向」
	Location  string   `gorm:"size:40"`                   // 所在地
	Summary   string   `gorm:"size:500"`                  // 简介
	Stacks    []string `gorm:"type:json;serializer:json"` // 技术栈
	Awards    []string `gorm:"type:json;serializer:json"` // 奖项；为空时前端自动隐藏该行
	Doing     string   `gorm:"size:60"`                   // 最近在做
	Interests []string `gorm:"type:json;serializer:json"` // 兴趣
	// 经历时间线：每条 "时间段 | 标题 | 说明"（如 "2024.09 – 2028.06 | 广东金融学院 · 计算机科学与技术 | 本科在读"）。
	// 用字符串而不是子表：条目就两三条，整组读写，后台一行一条最好维护。
	Experiences []string `gorm:"type:json;serializer:json"`
	// 站点默认配色：sand（暖纸·朱砂，浅色）/ ink（石墨·青，深色）。
	// 存在档案表里而不是单独建配置表：它就是这个站点的一个展示属性，
	// 跟着档案一起读写，省一次查询也省一张表。
	Theme     string `gorm:"size:16;default:sand"`
	UpdatedAt time.Time
}

// ResumeFile 简历 PDF 的元信息：单行表（ID 固定为 1）。
// 文件本身落盘在配置的上传目录里，数据库只记「有没有、叫什么、多大、什么时候换的」——
// 把 PDF 塞进 BLOB 会让每次查询都拖着一坨二进制，没必要。
type ResumeFile struct {
	ID        uint   `gorm:"primaryKey"`
	Filename  string `gorm:"size:255"`           // 原始文件名，仅用于展示
	Size      int64  `gorm:"not null;default:0"` // 字节数
	UpdatedAt time.Time
}

// VisitLog 单次访问记录：明细表，"谁（visitor_key）什么时候访问了什么（path）"。
// 访客身份两级识别：前端 UUID（X-Visitor-ID）或降级 sha256(IP+UA)，见 middleware.VisitTracker。
type VisitLog struct {
	ID         uint64    `json:"id" gorm:"primaryKey"`
	VisitorKey string    `json:"visitor_key" gorm:"size:64;index"` // 访客标识
	Nickname   string    `json:"nickname" gorm:"size:64"`          // 访客昵称（X-Visitor-Name 头，可空）
	Device     string    `json:"device" gorm:"size:128"`           // 设备/系统串（X-Visitor-Device 头，如 "iPhone · iOS 17 · Safari"）
	Path       string    `json:"path" gorm:"size:255"`             // 访问的接口路径
	IP         string    `json:"ip" gorm:"size:64"`
	UserAgent  string    `json:"user_agent" gorm:"size:512"`
	Referer    string    `json:"referer" gorm:"size:512"`
	CreatedAt  time.Time `json:"created_at" gorm:"index"` // 访问时间
}
