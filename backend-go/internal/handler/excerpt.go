package handler

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

// ---------- 列表摘要（excerpt） ----------
//
// 背景：列表卡片原先直接把整段富文本 v-html 出来。即使有 line-clamp，
// 浏览器仍要为每条记录解析 HTML、建整棵 DOM、跑 prose 排版；一次 30 条卡片
// 会明显挤占主线程（切页时表现得像"点菜单没反应"）。
//
// 现在列表接口额外下发 excerpt（纯文本、已截断），列表用它做文本插值；
// 正文 content 仍保留在接口里（编辑弹窗需要它），详情页也仍用富文本渲染。

var (
	// 整段脚本/样式先删掉（含内容），避免把代码文本混进摘要
	reDropBlocks = regexp.MustCompile(`(?is)<(script|style)\b[^>]*>.*?</(script|style)>`)
	// 行内换行语义的标签转成空格，避免摘要里两段文字粘连
	reBr        = regexp.MustCompile(`(?is)<br\s*/?>`)
	reBlockEnds = regexp.MustCompile(`(?is)</(p|div|li|h[1-6]|blockquote|tr|td|th)>`)
	// 其余标签一律删除
	reAnyTag = regexp.MustCompile(`(?s)<[^>]*>`)
	// 连续空白折叠
	reSpaces = regexp.MustCompile(`\s+`)
)

// excerptOf 把富文本正文压成一段纯文本摘要，最多 maxRunes 个字符。
// 纯文本输入（不含标签）会原样折叠空白返回，便于同一份逻辑复用于多个实体。
func excerptOf(content string, maxRunes int) string {
	if content == "" {
		return ""
	}
	s := content
	if strings.ContainsAny(s, "<&") {
		s = reDropBlocks.ReplaceAllString(s, " ")
		s = reBr.ReplaceAllString(s, " ")
		s = reBlockEnds.ReplaceAllString(s, " ")
		s = reAnyTag.ReplaceAllString(s, "")
		// 标签删完才反转义：先反转义会把 &lt;img&gt; 重新变成标签再被误删
		s = html.UnescapeString(s)
	}
	s = reSpaces.ReplaceAllString(s, " ")
	s = strings.TrimSpace(s)
	if maxRunes <= 0 {
		return s
	}
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	// 尽量在空格处断开，避免把词切断
	cut := string(r[:maxRunes])
	if i := strings.LastIndexFunc(cut, unicode.IsSpace); i > maxRunes/2 {
		cut = cut[:i]
	}
	return strings.TrimSpace(cut) + "…"
}
