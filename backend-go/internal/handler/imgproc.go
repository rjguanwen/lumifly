package handler

import (
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	// 解码器注册：image.Decode 只认已注册的格式（jpeg 因直接使用而自动注册）
	_ "image/gif"
	_ "image/png"
	_ "golang.org/x/image/webp"
)

// ---------- 上传时的图片压缩 ----------
//
// 背景：原始实现把上传的图片原样落盘。实测库里 29 张图共 59.8 MB，平均 2.1 MB，
// 最大 4.8 MB（2730×1535 的 PNG 截图）——手机截图与 4K 屏幕截图存成 PNG 极其浪费，
// 同样内容转 JPEG(q=85) 通常只有几百 KB。
//
// 做法：超过阈值（长边 > 1920 或体积 > 1 MB）的图片重新编码为 JPEG。
// 落盘只保留压缩后的一份，不额外存原图 —— 理由是：
//  1) 应用内所有展示尺寸都远小于 1920（缩略图 256、详情页约 900），保留原图无人受益；
//  2) 多存一份原图会让磁盘体积翻倍，与"节省体积"的初衷相反。
//
// 有损重编码只发生一次（上传时），之后缩略图都从压缩后的文件生成。

const (
	// compressLongEdge 长边超过该值则缩放（1920 足够 2x DPR 下的全宽详情图）
	compressLongEdge = 1920
	// compressMinSize 体积超过该值才考虑重编码（小于 1MB 的图收益有限）
	compressMinSize = 1024 * 1024
	// compressQuality JPEG 质量，85 在照片类内容上肉眼无损
	compressQuality = 85
)

// compressResult 描述压缩结果。Changed=false 表示保持原文件不动。
type compressResult struct {
	Changed bool
	Path    string // 最终落盘路径（扩展名可能由 .png 变为 .jpg）
	Width   int
	Height  int
	Size    int64
}

// shouldCompress 判断是否需要重编码：仅 JPEG/PNG 走压缩（GIF 会丢动画，WebP 已足够小）。
// 同时返回是否值得尝试 —— 小图与动画 GIF 直接放过。
func shouldCompress(ext string, size int64) bool {
	if size <= compressMinSize {
		return false
	}
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg", ".png":
		return true
	}
	return false
}

// compressImageInPlace 就地把 path 处的图片重编码为 JPEG。
// 采用「写临时文件 → rename 覆盖」，确保任何一步失败都不会破坏原文件。
func compressImageInPlace(path string) (compressResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return compressResult{}, err
	}
	cfg, _, err := image.DecodeConfig(f)
	f.Close()
	if err != nil {
		return compressResult{}, err
	}
	srcW, srcH := cfg.Width, cfg.Height
	if srcW <= 0 || srcH <= 0 {
		return compressResult{}, nil
	}

	// 是否真的需要缩放
	needResize := srcW > compressLongEdge || srcH > compressLongEdge
	outW, outH := srcW, srcH
	if needResize {
		if srcW >= srcH {
			outW = compressLongEdge
			outH = int(float64(srcH) * float64(compressLongEdge) / float64(srcW))
		} else {
			outH = compressLongEdge
			outW = int(float64(srcW) * float64(compressLongEdge) / float64(srcH))
		}
		if outW < 1 {
			outW = 1
		}
		if outH < 1 {
			outH = 1
		}
	}

	f2, err := os.Open(path)
	if err != nil {
		return compressResult{}, err
	}
	img, _, err := image.Decode(f2)
	f2.Close()
	if err != nil {
		return compressResult{}, err
	}

	dst := image.NewRGBA(image.Rect(0, 0, outW, outH))
	// 透明区铺白：PNG 带 alpha 时转 JPEG 否则会变黑
	draw.Draw(dst, dst.Bounds(), image.White, image.Point{}, draw.Src)
	if needResize {
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Over, nil)
	} else {
		draw.Draw(dst, dst.Bounds(), img, img.Bounds().Min, draw.Over)
	}

	tmp := path + ".cmp"
	out, err := os.Create(tmp)
	if err != nil {
		return compressResult{}, err
	}
	if err := jpeg.Encode(out, dst, &jpeg.Options{Quality: compressQuality}); err != nil {
		out.Close()
		_ = os.Remove(tmp)
		return compressResult{}, err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return compressResult{}, err
	}

	st, err := os.Stat(tmp)
	if err != nil {
		_ = os.Remove(tmp)
		return compressResult{}, err
	}
	// 体积判断只在「没有缩放」时生效：
	//   - needResize 时，长边超限是硬约束，即使体积没变小也必须采用缩放结果
	//     （否则 4000px 的图永远不会被缩，解码开销问题依旧存在）；
	//   - 未缩放时，若重编码反而更大（高熵内容）就放弃，保留原文件。
	orig, _ := os.Stat(path)
	if !needResize && orig != nil && st.Size() >= orig.Size() {
		_ = os.Remove(tmp)
		return compressResult{Width: srcW, Height: srcH, Size: orig.Size()}, nil
	}

	// JPEG 编码后扩展名必须是 .jpg，否则 ServeFile 会按原扩展名给出错误的 Content-Type
	newPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".jpg"
	if err := os.Rename(tmp, newPath); err != nil {
		_ = os.Remove(tmp)
		return compressResult{}, err
	}
	if newPath != path {
		_ = os.Remove(path)
	}
	return compressResult{Changed: true, Width: outW, Height: outH, Size: st.Size(), Path: newPath}, nil
}
