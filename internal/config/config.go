package config

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"strconv"
	"strings"
)

// Config 全局配置结构，对应 configs/config.yaml
type Config struct {
	Server ServerConfig `yaml:"server"`
	MySQL  MySQLConfig  `yaml:"mysql"`
	JWT    JWTConfig    `yaml:"jwt"`
	Admin  AdminConfig  `yaml:"admin"`
	CORS   CORSConfig   `yaml:"cors"`
	Redis  RedisConfig  `yaml:"redis"`
	Upload UploadConfig `yaml:"upload"`
}

// UploadConfig 上传文件的落盘目录。简历 PDF 存这里，
// 容器部署时挂 volume，避免重新发版把文件冲掉（文件不进数据库，只存元信息）。
type UploadConfig struct {
	Dir string `yaml:"dir"`
}

// RedisConfig 缓存连接配置。生产环境用环境变量覆盖 host/password
type RedisConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"` // 无密码则留空
	DB       int    `yaml:"db"`       // 逻辑库编号，默认 0
}

// Addr 拼接 go-redis 需要的 "host:port"
func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// ServerConfig HTTP 服务配置
type ServerConfig struct {
	Port     int    `yaml:"port"`
	Timezone string `yaml:"timezone"` // 应用时区，默认 Asia/Shanghai，用于校准时间显示与「今日」统计边界
}

// CORSConfig 跨域白名单。允许的前端来源列表，生产环境用环境变量覆盖
type CORSConfig struct {
	AllowOrigins []string `yaml:"allow_origins"`
}

// JWTConfig 签发/校验 token 的配置
type JWTConfig struct {
	Secret      string `yaml:"secret"`       // 签名密钥，生产用环境变量覆盖，绝不用默认值
	ExpireHours int    `yaml:"expire_hours"` // token 有效期（小时）
}

// Validate 校验 JWT 配置是否可用于生产签发。
// HS256 要求密钥至少 256 位（32 字节），过短的密钥易被暴力破解。
func (j JWTConfig) Validate() error {
	if len(j.Secret) < 32 {
		return fmt.Errorf("jwt secret 太短：需至少 32 字节，当前 %d 字节", len(j.Secret))
	}
	return nil
}

// AdminConfig 初始管理员账号。启动 seed 用，账号已存在则跳过；生产用环境变量注入
type AdminConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

// MySQLConfig 数据库连接配置
type MySQLConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	DBName   string `yaml:"dbname"`
	Charset  string `yaml:"charset"`
}

// DSN 拼接 go-sql-driver/mysql 的连接串（parseTime 让 time.Time 直接映射 DATETIME）
func (m MySQLConfig) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local",
		m.User, m.Password, m.Host, m.Port, m.DBName, m.Charset)
}

// Load 读取 configs/config.yaml 得到默认值，再用环境变量覆盖（生产/容器部署）。
// 没有任何隐式魔法：先解析 yaml，再逐个字段看对应环境变量是否设置。
func Load() *Config {
	data, err := os.ReadFile("configs/config.yaml")
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	applyEnvOverrides(&cfg)
	// 上传目录不给默认值就是空字符串，这里兜一个，省得每个调用点都判空
	if cfg.Upload.Dir == "" {
		cfg.Upload.Dir = "./storage"
	}
	return &cfg
}

// applyEnvOverrides 用环境变量覆盖 yaml 里的默认值。
// 约定：每个配置项对应一个大写环境变量；只有当环境变量“已设置”时才覆盖，
// 空字符串也算“未设置”，避免容器里没传的变量把默认值冲成空。
func applyEnvOverrides(cfg *Config) {
	setInt(&cfg.Server.Port, "SERVER_PORT")
	setStr(&cfg.Server.Timezone, "SERVER_TIMEZONE")

	setStr(&cfg.MySQL.Host, "MYSQL_HOST")
	setInt(&cfg.MySQL.Port, "MYSQL_PORT")
	setStr(&cfg.MySQL.User, "MYSQL_USER")
	setStr(&cfg.MySQL.Password, "MYSQL_PASSWORD")
	setStr(&cfg.MySQL.DBName, "MYSQL_DBNAME")
	setStr(&cfg.MySQL.Charset, "MYSQL_CHARSET")

	setStr(&cfg.Redis.Host, "REDIS_HOST")
	setInt(&cfg.Redis.Port, "REDIS_PORT")
	setStr(&cfg.Redis.Password, "REDIS_PASSWORD")
	setInt(&cfg.Redis.DB, "REDIS_DB")

	setStr(&cfg.JWT.Secret, "JWT_SECRET")
	setInt(&cfg.JWT.ExpireHours, "JWT_EXPIRE_HOURS")

	setStr(&cfg.Upload.Dir, "UPLOAD_DIR")

	setStr(&cfg.Admin.Username, "ADMIN_USERNAME")
	setStr(&cfg.Admin.Password, "ADMIN_PASSWORD")

	// CORS 白名单：逗号分隔的多个来源
	if v, ok := os.LookupEnv("CORS_ALLOW_ORIGINS"); ok && v != "" {
		cfg.CORS.AllowOrigins = splitAndTrim(v)
	}
}

// setStr：环境变量已设置且非空时，覆盖字符串字段
func setStr(dst *string, key string) {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		*dst = v
	}
}

// setInt：环境变量已设置且能解析为整数时，覆盖整型字段；解析失败直接 Fatal 暴露配置错误
func setInt(dst *int, key string) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Fatalf("环境变量 %s=%q 不是合法整数: %v", key, v, err)
	}
	*dst = n
}

// splitAndTrim 把 "a, b ,c" 切成 ["a","b","c"]
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
