package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"log"
	"net"
	"net/url"
	"strings"
	"unicode/utf8"

	"abingblog-backend/internal/model"
	"abingblog-backend/internal/repository"

	"github.com/gin-gonic/gin"
)

// VisitTracker 访问记录中间件：记录每次前台请求"是谁、何时、访问了什么"。
// 访客身份两级识别：
//  1. 前端 localStorage 生成的 UUID，通过 X-Visitor-ID 请求头带上（首选）
//  2. 没带头的，降级用 sha256(IP + User-Agent)
//
// 异步落库：go func 里写 MySQL，不阻塞响应。博客流量下安全；
// 量大了换 buffered channel + 单 worker（对应 Java 的线程池 + 队列）。
func VisitTracker(repo *repository.VisitLogRepo) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 后台接口与部署探活不算访客
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/v1/admin") || path == "/api/v1/healthz" {
			c.Next()
			return
		}

		ua := c.Request.UserAgent()
		ip := normalizeIP(c.ClientIP())

		// 1. 确定访客身份
		key := sanitizeHeader(c.GetHeader("X-Visitor-ID"), 64)
		if key == "" {
			sum := sha256.Sum256([]byte(ip + "|" + ua))
			key = hex.EncodeToString(sum[:])[:64]
		}
		// 昵称允许中文，前端用 encodeURIComponent 送过来，这里还原
		nickname := sanitizeHeader(decodeHeader(c.GetHeader("X-Visitor-Name")), 64)
		device := sanitizeHeader(c.GetHeader("X-Visitor-Device"), 128)

		// 2. 先取出需要的数据再起 goroutine！
		//    gin.Context 在请求结束后会被回收复用，异步代码里绝不能再碰 c。
		entry := model.VisitLog{
			VisitorKey: key,
			Nickname:   nickname,
			Device:     device,
			Path:       path,
			IP:         ip,
			UserAgent:  sanitizeHeader(ua, 512),
			Referer:    sanitizeHeader(c.Request.Header.Get("Referer"), 512),
		}

		// 3. 异步写库，不阻塞响应
		go func() {
			defer func() {
				if r := recover(); r != nil { // goroutine 里的 panic 不会自动冒泡，必须自己兜住
					log.Printf("记录访问日志失败(panic): %v", r)
				}
			}()
			if err := repo.Create(&entry); err != nil {
				log.Printf("记录访问日志失败: %v", err)
			}
		}()

		// 4. 放行，继续后面的处理
		c.Next()
	}
}

// sanitizeHeader 把 HTTP 头转成能安全写进 utf8mb4 列的字符串。
//
// 为什么需要：HTTP 头本质是字节流。浏览器发非 ASCII 字符（比如 "·" U+00B7）时，
// 会按 ISO-8859-1 落成单字节 0xB7，Go 读到的就是一段非法 UTF-8；
// 直接 INSERT 到 utf8mb4 列会报 1366 Incorrect string value。
// 另外长度必须按「字符数」截断——按字节切会把多字节字符劈成两半，同样写不进库。
func sanitizeHeader(s string, max int) string {
	// 先修非法 UTF-8，再清控制字符。
	// 顺序不能反：strings.Map 遇到非法字节会把它当成 U+FFFD 吐回来，
	// 先 Map 的话这一步等于把坏字节"洗白"成合法的替换符，ToValidUTF8 就删不掉了。
	s = strings.ToValidUTF8(strings.TrimSpace(s), "")
	// 控制字符（含 \x00）MySQL 也不收；制表符折成空格
	s = strings.Map(func(r rune) rune {
		switch {
		case r == '\t':
			return ' '
		case r < 0x20 || r == 0x7f:
			return -1
		default:
			return r
		}
	}, s)
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

// decodeHeader 还原 percent-encoding（前端送中文昵称时会编码）。
// 没带 % 就不动，避免把正常的百分号吃掉。
func decodeHeader(s string) string {
	if !strings.ContainsRune(s, '%') {
		return s
	}
	if decoded, err := url.QueryUnescape(s); err == nil {
		return decoded
	}
	return s
}

// normalizeIP 把 IPv6 回环/映射地址规整成可读的 IPv4（::1 → 127.0.0.1、::ffff:1.2.3.4 → 1.2.3.4），
// 避免后台把 ::1 这类地址当成"乱码"展示。无法解析的原始串原样返回。
func normalizeIP(s string) string {
	ip := net.ParseIP(strings.TrimSpace(s))
	if ip == nil {
		return s
	}
	if v4 := ip.To4(); v4 != nil {
		return v4.String()
	}
	if ip.IsLoopback() {
		return "127.0.0.1"
	}
	return s
}
