package handler

import (
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestWithThumbWidth(t *testing.T) {
	cases := []struct{ in, want string }{
		// 应用内上传图：追加 w
		{"/api/uploads/images/a.png", "/api/uploads/images/a.png?w=512"},
		// 已带其它查询参数：用 & 续接
		{"/api/uploads/images/a.png?v=1", "/api/uploads/images/a.png?v=1&w=512"},
		// 已经是缩略图：不重复加
		{"/api/uploads/images/a.png?w=256", "/api/uploads/images/a.png?w=256"},
		// 外链 / data URI：原样返回
		{"https://cdn.example.com/a.png", "https://cdn.example.com/a.png"},
		{"data:image/png;base64,AAAA", "data:image/png;base64,AAAA"},
	}
	for _, c := range cases {
		if got := withThumbWidth(c.in, 512); got != c.want {
			t.Errorf("withThumbWidth(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

// TestCardImageURLs 确认广场卡片的封面 URL 都带了缩略图参数。
func TestCardImageURLs(t *testing.T) {
	content := `<p>文字</p><img src="/api/uploads/images/a.png" /><img src="/api/uploads/images/b.jpg" /><img src="/api/uploads/images/c.png" /><img src="/api/uploads/images/d.png" />`
	urls := cardImageURLs(content)
	if len(urls) != 3 {
		t.Fatalf("expected 3 urls (capped), got %d: %v", len(urls), urls)
	}
	for _, u := range urls {
		if !strings.Contains(u, "?w=512") {
			t.Errorf("card url missing thumb param: %q", u)
		}
	}
}

// TestContentWithMediaUsesThumb 确认新发布快照里的图片走缩略图而不是原图。
func TestContentWithMediaUsesThumb(t *testing.T) {
	media := []gin.H{
		{"mimeType": "image/png", "url": "/api/uploads/images/a.png"},
		{"mimeType": "video/mp4", "url": "/api/uploads/videos/b.mp4"},
	}
	html := contentWithMedia("<p>正文</p>", media)

	if !strings.Contains(html, "/api/uploads/images/a.png?w=1024") {
		t.Errorf("image not rewritten to thumb: %s", html)
	}
	if strings.Contains(html, `src="/api/uploads/images/a.png"`) {
		t.Errorf("original image url still present: %s", html)
	}
	// 视频不参与缩略图改写
	if !strings.Contains(html, `src="/api/uploads/videos/b.mp4"`) {
		t.Errorf("video url should be untouched: %s", html)
	}
	// 图片应带显式宽度，避免正文渲染后跳动
	if !strings.Contains(html, `width="1024"`) {
		t.Errorf("image missing width attribute: %s", html)
	}
}
