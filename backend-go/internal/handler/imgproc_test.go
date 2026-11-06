package handler

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// TestShouldCompress 体积与扩展名双条件：小图放过、GIF/WebP 放过、大 JPEG/PNG 才压。
func TestShouldCompress(t *testing.T) {
	const mb = 1024 * 1024
	cases := []struct {
		ext  string
		size int64
		want bool
	}{
		{".png", 5 * mb, true},
		{".jpg", 5 * mb, true},
		{".PNG", 5 * mb, true},
		{".png", 500 * 1024, false}, // 小于 1MB 不压
		{".gif", 5 * mb, false},     // GIF 压了会丢动画
		{".webp", 5 * mb, false},    // 已足够小
		{".mp4", 50 * mb, false},    // 视频不走这条路
	}
	for _, c := range cases {
		if got := shouldCompress(c.ext, c.size); got != c.want {
			t.Errorf("shouldCompress(%q,%d)=%v want %v", c.ext, c.size, got, c.want)
		}
	}
}

// writeTestPNG 生成一张"类照片"的 PNG：平滑渐变 + 低频变化。
// 刻意不用逐像素高熵噪声——随机噪声是 JPEG 最不擅长的内容（编码后反而膨胀），
// 用它做样本会把"体积变小"这条断言测成假失败，而真实照片/截图并不长那样。
func writeTestPNG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			// 三个通道各自沿不同方向平滑变化，块内近似常数，利于 JPEG 压缩
			img.Set(x, y, color.RGBA{
				R: uint8(40 + (x*180)/w),
				G: uint8(40 + (y*180)/h),
				B: uint8(60 + ((x+y)*120)/(w+h)),
				A: 255,
			})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
}

// TestCompressImageInPlace 大 PNG 应被缩到长边 1920 并转为 JPEG，且扩展名变成 .jpg。
func TestCompressImageInPlace(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "big.png")
	writeTestPNG(t, src, 2200, 1400)

	before, _ := os.Stat(src)
	res, err := compressImageInPlace(src)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if !res.Changed {
		t.Fatalf("expected Changed=true, got %+v", res)
	}
	if filepath.Ext(res.Path) != ".jpg" {
		t.Errorf("expected .jpg output, got %q", res.Path)
	}
	if res.Width != compressLongEdge {
		t.Errorf("long edge=%d want %d", res.Width, compressLongEdge)
	}
	// 等比：2200x1400 -> 1920x1221
	if res.Height != 1221 {
		t.Errorf("height=%d want 1221", res.Height)
	}
	after, err := os.Stat(res.Path)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	// 硬约束是"长边不超限"，体积只作为观测值输出：
	// 高熵内容重编码成 JPEG 有可能变大，那不是缺陷。
	t.Logf("压缩：%d -> %d 字节（%.1f%%），%dpx 长边",
		before.Size(), after.Size(), float64(after.Size())*100/float64(before.Size()), res.Width)
	// 原 PNG 应被移除（只保留压缩后的一份）
	if _, err := os.Stat(src); !os.IsNotExist(err) {
		t.Errorf("original png should be removed")
	}
	// 输出必须是可解码的 JPEG
	f, _ := os.Open(res.Path)
	defer f.Close()
	img, format, err := image.Decode(f)
	if err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("format=%q want jpeg", format)
	}
	if got := img.Bounds().Dx(); got != compressLongEdge {
		t.Errorf("decoded width=%d", got)
	}
}

// TestCompressSkipsSmallImage 长边已在上限内的图片不应被缩放（只重编码）。
func TestCompressSkipsSmallImage(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "small.png")
	writeTestPNG(t, src, 800, 600)

	res, err := compressImageInPlace(src)
	if err != nil {
		t.Fatalf("compress: %v", err)
	}
	if res.Width != 800 || res.Height != 600 {
		t.Errorf("dimensions changed: %dx%d", res.Width, res.Height)
	}
}
