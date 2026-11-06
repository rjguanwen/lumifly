package handler

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/draw"
	// 解码器注册：image.Decode 只认已注册的格式。
	// png/gif 必须显式空白引入（jpeg 因上面直接使用而自动注册），webp 来自 x/image。
	_ "image/gif"
	_ "image/png"
	_ "golang.org/x/image/webp"
)

// ---------- 图片缩略图 ----------
//
// 背景：列表页把附件渲染成 64×64 / 80×80 的小方格，但 <img src> 直接指向原图。
// 库里单张截图常见 2~5 MB（2730×1535 约 419 万像素），一个 64px 的格子要下载
// 整张原图并解码 419 万像素再缩到 64px，首屏几十张卡片会把主线程压满。
//
// 做法：/api/uploads/images/xxx.png?w=256 —— 首次请求时生成缩略图落盘到
// uploads/.thumbs/<w>/<原相对路径>，之后直接命中文件。文件名是 UUID，
// 内容永不变化，因此缩略图可以用 immutable 强缓存。
//
// 约束（都是刻意的）：
//   - 尺寸走白名单，不接受任意值，避免被刷出无数份缓存文件；
//   - 只放大不缩小时直接复用原图（原图比目标还小，再编码只会更糊更慢）；
//   - 生成失败时回落到原图（宁可慢，也不能让图片整体裂掉）；
//   - 生成过程带 singleflight，避免首屏几十个并发请求重复解码同一张图。

// thumbWidths 允许的缩略图宽度白名单（像素）。列表最高用到 256（覆盖 2x DPR 的 64/80 格
// 与详情页约 230px 的网格），512 供未来的大卡片使用。
var thumbWidths = map[int]bool{32: true, 64: true, 128: true, 256: true, 512: true}

// thumbJpegQuality 缩略图编码质量。缩略图只用于展示，82 在该尺寸下肉眼无损。
const thumbJpegQuality = 82

// parseThumbWidth 解析 ?w= 参数；不在白名单内返回 0（表示按原图返回）。
func parseThumbWidth(raw string) int {
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || !thumbWidths[n] {
		return 0
	}
	return n
}

// thumbPath 返回缩略图缓存文件的绝对路径。
// rel 形如 images/xxx.png，缓存为 <uploadDir>/.thumbs/<w>/images/xxx.jpg。
func (h *Handler) thumbPath(rel string, w int) string {
	base := strings.TrimSuffix(rel, filepath.Ext(rel))
	return filepath.Join(h.uploadDir, ".thumbs", strconv.Itoa(w), filepath.FromSlash(base)+".jpg")
}

// 同一张缩略图的并发生成合并成一次：用路径做 key 的简单去重。
var (
	thumbLocksMu sync.Mutex
	thumbLocks   = map[string]*sync.Mutex{}
)

func thumbLock(key string) *sync.Mutex {
	thumbLocksMu.Lock()
	defer thumbLocksMu.Unlock()
	m, ok := thumbLocks[key]
	if !ok {
		m = &sync.Mutex{}
		thumbLocks[key] = m
	}
	return m
}

// ensureThumb 确保缩略图存在并返回其路径。
// 返回值 ok=false 表示无法/无需生成，调用方应回落到原图。
func (h *Handler) ensureThumb(fullSrc, rel string, w int) (string, bool) {
	dst := h.thumbPath(rel, w)
	if fileNonEmpty(dst) {
		return dst, true
	}

	mu := thumbLock(dst)
	mu.Lock()
	defer mu.Unlock()
	// 双重检查：等锁期间可能已被同请求的兄弟协程生成
	if fileNonEmpty(dst) {
		return dst, true
	}

	if err := makeThumb(fullSrc, dst, w); err != nil {
		return "", false
	}
	return dst, true
}

// makeThumb 读取原图、等比缩放到宽度 w、编码为 JPEG 写入 dst。
// 采用「先写临时文件再 rename」：避免并发读到半截文件，也避免写失败留下损坏缓存。
func makeThumb(src, dst string, w int) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		// gif/webp 解码器已注册；失败说明文件损坏或格式不支持
		return err
	}
	srcW, srcH := img.Bounds().Dx(), img.Bounds().Dy()
	if srcW == 0 || srcH == 0 {
		return errors.New("empty image")
	}
	// 原图不比目标宽：直接用原图（调用方按 ok=false 回落）
	if srcW <= w {
		return errors.New("source smaller than target")
	}

	// 等比缩放，高度向上取整避免竖向 1px 缺失
	hgt := int(float64(srcH) * float64(w) / float64(srcW))
	if hgt < 1 {
		hgt = 1
	}
	dstImg := image.NewRGBA(image.Rect(0, 0, w, hgt))
	// gif 的首帧可能只有调色板信息，统一先铺白底，避免 PNG/GIF 透明区转 JPEG 后变黑
	draw.Draw(dstImg, dstImg.Bounds(), image.White, image.Point{}, draw.Src)

	// CatmullRom 比 ApproxBiLinear 慢一些但边缘明显更干净；缩略图只生成一次，值得。
	draw.CatmullRom.Scale(dstImg, dstImg.Bounds(), img, img.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dstImg, &jpeg.Options{Quality: thumbJpegQuality}); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	if err := os.WriteFile(tmp, buf.Bytes(), 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// thumbURLFor 返回某媒体文件的缩略图访问 URL。
// 仅对图片生成；非图片返回空串，由调用方决定回落到原图还是留空。
func thumbURLFor(mimeType, filePath string) string {
	if !strings.HasPrefix(mimeType, "image/") {
		return ""
	}
	return "/api/uploads/" + filePath + "?w=256"
}

func fileNonEmpty(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Size() > 0
}
