package service

import (
	"strings"
	"testing"
)

// TestCleanList 覆盖档案列表字段的清洗规则。
// 这里只测纯函数，不碰数据库：cleanList 是 Update 里唯一有分支的部分，
// 用表驱动把「trim / 丢空项 / 单项超长 / 条数超限 / 非 nil 返回」都钉住。
func TestCleanList(t *testing.T) {
	t.Run("trim 并丢掉空项", func(t *testing.T) {
		got, err := cleanList("技术栈", []string{" Go ", "", "   ", "Java"})
		if err != nil {
			t.Fatalf("不应报错: %v", err)
		}
		if len(got) != 2 || got[0] != "Go" || got[1] != "Java" {
			t.Fatalf("清洗结果不符: %#v", got)
		}
	})

	t.Run("空输入返回空切片而不是 nil", func(t *testing.T) {
		got, err := cleanList("奖项", nil)
		if err != nil {
			t.Fatalf("不应报错: %v", err)
		}
		if got == nil {
			t.Fatal("必须返回非 nil 切片，否则 JSON 会序列化成 null")
		}
		if len(got) != 0 {
			t.Fatalf("期望空切片，实际 %#v", got)
		}
	})

	t.Run("单项超长被拒绝", func(t *testing.T) {
		long := strings.Repeat("奖", profileItemMax+1)
		if _, err := cleanList("奖项", []string{long}); err == nil {
			t.Fatal("超过单项上限应报错")
		}
	})

	t.Run("单项恰好等于上限允许通过", func(t *testing.T) {
		edge := strings.Repeat("奖", profileItemMax)
		if _, err := cleanList("奖项", []string{edge}); err != nil {
			t.Fatalf("等于上限不应报错: %v", err)
		}
	})

	t.Run("条数超过上限被拒绝", func(t *testing.T) {
		items := make([]string, 0, profileListMax+1)
		for i := 0; i <= profileListMax; i++ {
			items = append(items, "item")
		}
		if _, err := cleanList("技术栈", items); err == nil {
			t.Fatal("超过条数上限应报错")
		}
	})

	t.Run("空串不计入条数", func(t *testing.T) {
		items := make([]string, 0, profileListMax*2)
		for i := 0; i < profileListMax*2; i++ {
			items = append(items, "  ")
		}
		got, err := cleanList("技术栈", items)
		if err != nil {
			t.Fatalf("空串不该触发条数上限: %v", err)
		}
		if len(got) != 0 {
			t.Fatalf("期望全部被丢弃，实际 %#v", got)
		}
	})
}
