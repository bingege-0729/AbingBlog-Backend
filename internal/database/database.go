package database

import (
	"fmt"
	"log"

	"abingblog-backend/internal/config"
	"abingblog-backend/internal/model"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Init 连接 MySQL → 确保数据库存在 → 自动迁移表结构
// 开发期直接用 AutoMigrate；上线后应换成正式的迁移工具（如 golang-migrate）
func Init(cfg *config.Config) *gorm.DB {
	// 1. 先不带库名连接，创建数据库（若不存在）——首次部署无需手动建库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=%s&parseTime=True&loc=Local",
		cfg.MySQL.User, cfg.MySQL.Password, cfg.MySQL.Host, cfg.MySQL.Port, cfg.MySQL.Charset)
	boot, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Fatalf("连接 MySQL 失败: %v", err)
	}
	createSQL := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s COLLATE %s_unicode_ci",
		cfg.MySQL.DBName, cfg.MySQL.Charset, cfg.MySQL.Charset)
	if err := boot.Exec(createSQL).Error; err != nil {
		log.Fatalf("创建数据库失败: %v", err)
	}
	if sqlDB, err := boot.DB(); err == nil {
		_ = sqlDB.Close()
	}

	// 2. 正式连接业务库；开发期打印 SQL，方便观察调用链
	db, err := gorm.Open(mysql.Open(cfg.MySQL.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("连接业务数据库失败: %v", err)
	}

	// 3. 自动迁移：创建/更新表结构（含多对多中间表 article_tags）
	if err := db.AutoMigrate(
		&model.User{},
		&model.SiteMetric{},
		&model.Category{},
		&model.Tag{},
		&model.Article{},
		&model.Project{},
		&model.FriendLink{},
		&model.VisitLog{},
		&model.Profile{},
		&model.ResumeFile{},
	); err != nil {
		log.Fatalf("自动迁移失败: %v", err)
	}
	return db
}
