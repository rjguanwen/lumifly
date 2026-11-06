package handler

import (
	"strings"
	"testing"
)

func TestExcerptOfRealRecords(t *testing.T) {
	// 真实富文本片段：验证卡片摘要不会漏出标签，也不会把段落粘连
	cases := []struct {
		in   string
		want string
	}{
		{"<p>Test record</p>", "Test record"},
		{"<p>这是一个测试日记。今天天气很好，是个美好的一天！</p>", "这是一个测试日记。今天天气很好，是个美好的一天！"},
		{"<p>第一段</p><p>第二段</p>", "第一段 第二段"},
		{"<p>a<br>b</p>", "a b"},
		{"<script>var x=1</script><p>正文</p>", "正文"},
		{"<style>.a{}</style><p>样式外</p>", "样式外"},
		{"<ul><li>甲</li><li>乙</li></ul>", "甲 乙"},
		{"<p><strong><em>粗斜体</em></strong></p>", "粗斜体"},
		{"", ""},
		{"   ", ""},
	}
	for _, c := range cases {
		if got := excerptOf(c.in, 120); got != c.want {
			t.Errorf("excerptOf(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestExcerptNeverContainsTags(t *testing.T) {
	long := "<p>" + strings.Repeat("文字", 100) + "</p><p>结尾</p>"
	got := excerptOf(long, 120)
	if strings.ContainsAny(got, "<>") {
		t.Errorf("excerpt leaked tags: %q", got)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("expected ellipsis suffix, got %q", got)
	}
	if n := len([]rune(got)); n > 121 {
		t.Errorf("excerpt too long: %d runes", n)
	}
}

func TestThumbURLFor(t *testing.T) {
	if got := thumbURLFor("image/png", "images/a.png"); got != "/api/uploads/images/a.png?w=256" {
		t.Errorf("image thumbURLFor=%q", got)
	}
	if got := thumbURLFor("video/mp4", "videos/a.mp4"); got != "" {
		t.Errorf("video thumbURLFor should be empty, got %q", got)
	}
}
