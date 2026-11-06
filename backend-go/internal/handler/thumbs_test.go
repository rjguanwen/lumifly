package handler

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseThumbWidth(t *testing.T) {
	cases := map[string]int{
		"":    0,
		"256": 256,
		"64":  64,
		"512": 512,
		"999": 0,
		"abc": 0,
		"0":   0,
		"-64": 0,
	}
	for in, want := range cases {
		if got := parseThumbWidth(in); got != want {
			t.Errorf("parseThumbWidth(%q)=%d want %d", in, got, want)
		}
	}
}

func TestIsImageExt(t *testing.T) {
	for _, ext := range []string{".png", ".PNG", ".jpg", ".jpeg", ".gif", ".webp"} {
		if !isImageExt(ext) {
			t.Errorf("isImageExt(%q) should be true", ext)
		}
	}
	for _, ext := range []string{".mp4", ".webm", ".pdf", ""} {
		if isImageExt(ext) {
			t.Errorf("isImageExt(%q) should be false", ext)
		}
	}
}

func TestThumbPath(t *testing.T) {
	h := &Handler{uploadDir: filepath.Join("C:", "up")}
	got := h.thumbPath("images/abc.png", 256)
	want := filepath.Join("C:", "up", ".thumbs", "256", "images", "abc.jpg")
	if got != want {
		t.Errorf("thumbPath=%q want %q", got, want)
	}
}

func TestMakeThumbRealFile(t *testing.T) {
	src := filepath.Join("..", "..", "uploads", "images", "1875f166-7ea1-4853-b02c-a61c0f2bca51.png")
	if _, err := os.Stat(src); err != nil {
		t.Skip("sample image not present")
	}
	dst := filepath.Join(t.TempDir(), "out.jpg")
	if err := makeThumb(src, dst, 256); err != nil {
		t.Fatalf("makeThumb: %v", err)
	}
	st, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	srcSt, _ := os.Stat(src)
	t.Logf("thumb size=%d bytes, original=%d bytes, ratio=%.1f%%",
		st.Size(), srcSt.Size(), float64(st.Size())*100/float64(srcSt.Size()))
	if st.Size() >= srcSt.Size() {
		t.Errorf("thumb not smaller than original")
	}
}

func TestExcerptOf(t *testing.T) {
	got := excerptOf("<p>你好 <b>世界</b></p><p>第二段</p>", 100)
	want := "你好 世界 第二段"
	if got != want {
		t.Errorf("excerptOf=%q want %q", got, want)
	}
	// 标签内的实体不应被当作标签删除
	got2 := excerptOf("<p>a &lt;b&gt; c</p>", 100)
	if got2 != "a <b> c" {
		t.Errorf("excerptOf entity=%q", got2)
	}
	// 超长截断
	long := ""
	for i := 0; i < 200; i++ {
		long += "字"
	}
	if r := []rune(excerptOf(long, 10)); len(r) != 11 {
		t.Errorf("truncate len=%d want 11 (10 + ellipsis)", len(r))
	}
}
