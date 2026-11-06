package handler

import (
	"strings"
	"testing"
)

// TestContentImageURLsSkipsDataURI 内联 base64 图不能被当成卡片封面返回，
// 否则列表接口会凭空多下发 MB 级的 data URI。
func TestContentImageURLsSkipsDataURI(t *testing.T) {
	content := `<p><img src="data:image/jpeg;base64,/9j/AAAAAAAA" /></p>` +
		`<p><img src="/api/uploads/images/a.png" /></p>` +
		`<p><img src="/api/uploads/images/b.jpg" /></p>`
	urls := contentImageURLs(content)
	if len(urls) != 2 {
		t.Fatalf("expected 2 non-data urls, got %d: %v", len(urls), urls)
	}
	for _, u := range urls {
		if strings.HasPrefix(u, "data:") {
			t.Errorf("data URI leaked into card images: %q", u[:40])
		}
	}
}

// TestContentImageURLsAllDataURI 全内联图的正文应返回空列表（不是死循环，也不返回 data URI）。
func TestContentImageURLsAllDataURI(t *testing.T) {
	content := `<img src="data:image/png;base64,AAAA" /><img src="data:image/png;base64,BBBB" />`
	urls := contentImageURLs(content)
	if len(urls) != 0 {
		t.Errorf("expected empty, got %v", urls)
	}
}

// TestContentImageURLsCap 仍应最多返回 3 张。
func TestContentImageURLsCap(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 6; i++ {
		b.WriteString(`<img src="/api/uploads/images/x` + string(rune('a'+i)) + `.png" />`)
	}
	if urls := contentImageURLs(b.String()); len(urls) != 3 {
		t.Errorf("expected cap of 3, got %d", len(urls))
	}
}

// TestContentImageURLsUnchangedBehavior 普通情况下的提取结果保持不变。
func TestContentImageURLsUnchangedBehavior(t *testing.T) {
	content := `<p>正文</p><img class="a" src="/api/uploads/images/one.png" alt="" />`
	urls := contentImageURLs(content)
	if len(urls) != 1 || urls[0] != "/api/uploads/images/one.png" {
		t.Errorf("got %v", urls)
	}
}
