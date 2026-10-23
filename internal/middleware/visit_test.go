package middleware

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// 回归用例：浏览器发非 ASCII 头时，"·" 会按 latin1 落成单字节 0xB7，
// Go 拿到的是非法 UTF-8。以前直接写库会报 1366，把整个访问日志写丢。
func TestSanitizeHeaderDropsInvalidUTF8(t *testing.T) {
	raw := "Windows \xb7 Chrome" // 中间那个字节是 "·" 的 latin1 编码，不是合法 UTF-8
	if utf8.ValidString(raw) {
		t.Fatal("用例本身写错了：这段应该是非法 UTF-8")
	}

	got := sanitizeHeader(raw, 128)
	if !utf8.ValidString(got) {
		t.Fatalf("清洗后仍是非法 UTF-8: %q", got)
	}
	if got != "Windows  Chrome" {
		t.Fatalf("期望非法字节被丢弃，得到 %q", got)
	}
}

// 长度必须按字符数截，按字节切会把多字节字符劈开，同样写不进 utf8mb4。
func TestSanitizeHeaderTruncatesByRune(t *testing.T) {
	got := sanitizeHeader(strings.Repeat("广金", 40), 64)
	if !utf8.ValidString(got) {
		t.Fatalf("截断后不是合法 UTF-8: %q", got)
	}
	if n := utf8.RuneCountInString(got); n != 64 {
		t.Fatalf("期望 64 个字符，得到 %d", n)
	}
}

// 前端把中文昵称 percent-encode 后送过来，后端要还原。
func TestDecodeHeader(t *testing.T) {
	if got := decodeHeader("%E5%BB%96%E5%B1%95%E6%BB%A8"); got != "廖展滨" {
		t.Fatalf("percent-decoding 失败: %q", got)
	}
	// 没有百分号就原样返回，别把正常的 % 吃掉
	if got := decodeHeader("100% pure"); got != "100% pure" {
		t.Fatalf("不含 %% 时不应改动: %q", got)
	}
	// 非法编码不 panic，原样返回
	if got := decodeHeader("bad%zz"); got != "bad%zz" {
		t.Fatalf("非法编码应原样返回: %q", got)
	}
}

// 控制字符（\x00 等）MySQL 也不收，要清掉。
func TestSanitizeHeaderStripsControlChars(t *testing.T) {
	got := sanitizeHeader("  a\x00b\tc  ", 32)
	if got != "ab c" {
		t.Fatalf("期望清掉 \\x00、制表符折成空格并 trim，得到 %q", got)
	}
}
