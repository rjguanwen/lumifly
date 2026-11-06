package handler

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCompressOnRealUploads 用库里真实的 29 张上传图片验证压缩策略：
// 逐张压缩后比较体积，统计"变大"的比例。
// 若占比明显，说明无条件转 JPEG 的策略需要调整（例如保留 PNG 时择优）。
func TestCompressOnRealUploads(t *testing.T) {
	dir := filepath.Join("..", "..", "uploads", "images")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skip("no uploads dir")
	}

	var totalBefore, totalAfter int64
	var grew, shrank, skipped int
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		src := filepath.Join(dir, e.Name())
		ext := strings.ToLower(filepath.Ext(e.Name()))
		st, _ := os.Stat(src)
		if !shouldCompress(ext, st.Size()) {
			skipped++
			continue
		}

		// 复制一份到临时目录再压，绝不改动真实上传文件
		tmpDir := t.TempDir()
		dst := filepath.Join(tmpDir, e.Name())
		data, err := os.ReadFile(src)
		if err != nil {
			continue
		}
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			continue
		}

		before := st.Size()
		res, err := compressImageInPlace(dst)
		if err != nil {
			t.Logf("%s: compress error: %v", e.Name(), err)
			continue
		}
		after := before
		if res.Changed {
			if fi, err := os.Stat(res.Path); err == nil {
				after = fi.Size()
			}
		}
		totalBefore += before
		totalAfter += after
		if after > before {
			grew++
			t.Logf("GREW  %s %d -> %d (+%.1f%%)", e.Name(), before, after,
				float64(after-before)*100/float64(before))
		} else {
			shrank++
		}
	}

	if totalBefore == 0 {
		t.Skip("nothing compressible")
	}
	t.Logf("总计：%d 张压缩（%d 变小 / %d 变大），%d 张跳过；%d -> %d 字节（%.1f%%）",
		grew+shrank, shrank, grew, skipped, totalBefore, totalAfter,
		float64(totalAfter)*100/float64(totalBefore))

	// 实测基线（2026-11-06，17 张真实图片）：体积降到 10.7%，0 张变大。
	// 这里用 0 变大的硬断言守住"不能因为参数调整（质量、缩放阈值）而让某些图反而膨胀"。
	if grew > 0 {
		t.Errorf("%d 张图片压缩后变大，需要调整策略", grew)
	}
	// 再守一条总量红线：压缩后不应超过原来的 50%（当前实测 10.7%）
	if totalAfter*2 > totalBefore {
		t.Errorf("压缩后总量 %d 超过原总量的一半 %d", totalAfter, totalBefore/2)
	}
}
