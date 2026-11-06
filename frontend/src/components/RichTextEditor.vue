<template>
  <div class="editor-shell">
    <div ref="hostEl" class="rich-host"></div>
    <input ref="fileEl" type="file" :accept="IMAGE_ACCEPT" class="hidden" @change="onPickImageFile" />
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import Quill from 'quill'
import 'quill/dist/quill.snow.css'
import { uploadApi } from '../api'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '开始写作...' },
})

const emit = defineEmits(['update:modelValue'])

// 编辑器插图只允许后端能接收的图片格式。原先写 accept="image/*" 会放进 BMP/HEIC/TIFF
// 等格式，用户选中后才被 /api/upload 拒绝，白等一次上传失败。
// 同时列出扩展名与 MIME：Windows 选择框按扩展名过滤更可靠。
const IMAGE_ACCEPT = [
  '.jpg',
  '.jpeg',
  '.png',
  '.gif',
  '.webp',
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
].join(',')

const hostEl = ref(null)
const fileEl = ref(null)
let quill = null
let syncing = false
// 组件是否已卸载。上传是异步的，回调返回时组件可能已被 destroy-on-close 销毁，
// 必须靠这个标记判断"还能不能碰 quill"（不能只看 quill 是否为 null）。
let disposed = false
// .ql-link 的点击监听器与其元素引用（卸载时按同一引用 removeEventListener）
let linkHandler = null
let linkEl = null

// 所见即所得工具栏（图标直观，标题下拉可选）
//
// 注意 'image' 这一项：Quill 的默认 image handler 是 selectLocalImage()，
// 它会自己创建一个隐藏 <input type=file> 并弹出选择框，而且选完的图片用
// FileReader.readAsDataURL 内联成 base64 塞进正文（一张截图 1~3 MB 字符）。
// 我们不要这套行为，改用下面的 handler 指向自己的上传流程。
// 若这里保留默认的 'image'，同时又像以前那样在 onMounted 里给 .ql-image
// 额外加一个 click 监听去点自己的 input，就会一次点击弹出**两个**选择框。
const TOOLBAR = [
  [{ header: [false, 1, 2, 3] }],
  ['bold', 'italic', 'underline', 'strike'],
  ['blockquote', 'code-block'],
  [{ list: 'ordered' }, { list: 'bullet' }],
  ['link'],
  ['clean'],
  // image 按钮由下面自定义的 handler 接管（不传字符串 'image'，避免注册默认行为）
  [{ image: true }],
]

function currentHtml() {
  try {
    return quill ? (quill.getSemanticHTML() || '') : (props.modelValue || '')
  } catch {
    return props.modelValue || ''
  }
}

function emitHtml() {
  if (syncing) return
  const v = currentHtml()
  if (v !== props.modelValue) emit('update:modelValue', v)
}

function setHtmlSilent(html) {
  if (!quill) return
  syncing = true
  try {
    if (!html) {
      quill.setContents([], 'silent')
    } else {
      quill.clipboard.dangerouslyPasteHTML(html, 'silent')
    }
  } catch {
    /* 忽略异常 HTML */
  } finally {
    syncing = false
  }
}

onMounted(() => {
  if (!hostEl.value) return
  quill = new Quill(hostEl.value, {
    theme: 'snow',
    placeholder: props.placeholder,
    modules: {
      toolbar: {
        container: TOOLBAR,
        // 「图片」按钮的唯一入口：点自己的隐藏 input，走 /api/upload 上传后只把 URL
        // 插进正文。写了 handler 之后 Quill 不再执行内置 selectLocalImage，
        // 所以不会再有第二个文件选择框，也不会产生 base64 内联图。
        handlers: {
          image: () => {
            fileEl.value?.click()
          },
        },
      },
      clipboard: { matchVisual: false },
      // 覆盖 Quill 默认的 uploader.handler：默认实现用 FileReader.readAsDataURL 把
      // 粘贴/拖放进来的图片直接内联成 base64 文本塞进正文（一张截图就是 1~3 MB 字符），
      // 既拖慢编辑与渲染，又会被广场快照整段复制进 publications.content。
      // 改成与工具栏「图片」按钮一致：先传服务器，正文里只留 URL。
      // 何时触发仍完全由 Quill 自己判定（粘贴无 html 的图片 / 粘贴单个 img / 拖放）。
      uploader: {
        // 与后端 /api/upload 接受的图片类型对齐（默认为 png/jpeg，gif/webp 会被丢掉）
        mimetypes: ['image/png', 'image/jpeg', 'image/gif', 'image/webp'],
        handler: (range, files) => { onPasteOrDropImages(range, files) },
      },
    },
  })
  quill.on('text-change', () => emitHtml())
  if (props.modelValue) setHtmlSilent(props.modelValue)

  // 「插入链接」按钮：Quill 默认会弹一个自带的提示框（且样式不可控），
  // 这里改成自己的 promptLink（要求先选中文字）。
  // 注意不要再给 .ql-image 加 click 监听 —— 图片按钮已由上面的 toolbar.handlers.image
  // 独占，重复挂监听会导致一次点击弹两个文件选择框。
  const toolbar = hostEl.value.closest('.editor-shell')?.querySelector('.ql-toolbar')
  // 保存元素引用：卸载时 hostEl 可能已脱离文档，不能再靠查询找回来
  linkEl = toolbar?.querySelector('.ql-link') || null
  linkHandler = (e) => {
    e.preventDefault()
    promptLink()
  }
  linkEl?.addEventListener('click', linkHandler)
})

function promptLink() {
  if (!quill) return
  const sel = quill.getSelection()
  if (!sel || sel.length === 0) {
    ElMessage.warning('请先选中要加链接的文字')
    return
  }
  const url = window.prompt('请输入链接地址（http:// 或 https://）')
  if (!url) return
  quill.format('link', url)
}

async function onPickImageFile(e) {
  const file = e.target.files?.[0]
  if (fileEl.value) fileEl.value.value = ''
  if (!file) return
  try {
    const result = await uploadApi.upload(file)
    const url = result?.url || ''
    // 与 onPasteOrDropImages 同理：上传期间弹窗可能已被关闭（destroy-on-close），
    // 此时 quill 已为 null，必须直接丢弃结果而不是继续插入。
    if (!url || disposed || !quill) return
    const maxIndex = Math.max(0, quill.getLength() - 1)
    const sel = quill.getSelection() || { index: maxIndex, length: 0 }
    const at = Math.min(sel.index, maxIndex)
    quill.insertEmbed(at, 'image', url)
    quill.setSelection(at + 1, 0)
  } catch {
    if (!disposed) ElMessage.error('图片上传失败')
  }
}

// 粘贴/拖放图片：走服务端上传后只插入 URL，不内联 base64。
// 替换语义与 Quill 原实现一致：若粘贴时选区非空，先删除选中内容再插入。
//
// ⚠️ 上传是异步的，而所有引用此编辑器的弹窗都带 destroy-on-close：
// 用户在"上传中"关掉弹窗（或切换路由）会触发 onBeforeUnmount，把 quill 置为 null。
// 此时若直接 quill.insertEmbed 就会抛
//   TypeError: Cannot read properties of null (reading 'insertEmbed')
// 所以 await 返回后必须重新确认组件仍存活（disposed 标记 + quill 非空）。
async function onPasteOrDropImages(range, files) {
  if (!quill || !files?.length) return
  const urls = []
  let failed = 0
  for (const file of files) {
    try {
      const result = await uploadApi.upload(file)
      const url = result?.url || ''
      if (url) urls.push(url)
      else failed += 1
    } catch {
      failed += 1
    }
  }
  // 上传期间组件可能已卸载：此时静默丢弃结果（文件已落库，只是不再插入正文）
  if (disposed || !quill) return
  if (!urls.length) {
    ElMessage.error('图片上传失败')
    return
  }
  // 段落可能在上传期间被用户改动，索引需要按当前文档长度收敛，避免越界
  const safeIndex = Math.min(range.index, Math.max(0, quill.getLength() - 1))
  const safeLength = Math.min(range.length, Math.max(0, quill.getLength() - 1 - safeIndex))
  if (safeLength > 0) quill.deleteText(safeIndex, safeLength, 'silent')
  let at = safeIndex
  for (const url of urls) {
    quill.insertEmbed(at, 'image', url)
    at += 1
  }
  quill.setSelection(at, 0)
  if (failed) ElMessage.error(`${failed} 张图片上传失败`)
}

// 外部赋值（例如编辑回显）时同步到编辑器
watch(
  () => props.modelValue,
  (val) => {
    if (!quill) return
    const v = val || ''
    if (v !== currentHtml()) setHtmlSilent(v)
  },
)

onBeforeUnmount(() => {
  // 先置标记：让所有 pending 的上传回调知道"不要再碰 quill"
  disposed = true
  if (linkEl && linkHandler) {
    try {
      linkEl.removeEventListener('click', linkHandler)
    } catch {
      /* ignore */
    }
  }
  linkEl = null
  linkHandler = null
  try {
    quill?.destroy()
  } catch {
    /* ignore */
  }
  quill = null
})
</script>

<style scoped>
.editor-shell {
  width: 100%;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(16, 24, 40, 0.04);
  transition: border-color 0.15s ease, box-shadow 0.15s ease;
  overflow: hidden;
}

.editor-shell:hover {
  border-color: #d1d5db;
}

.editor-shell:focus-within {
  border-color: #5b7cff;
  box-shadow: 0 0 0 3px rgba(91, 124, 255, 0.15);
}

/* Quill 主题样式微调，使其融入现有外壳 */
:deep(.ql-toolbar.ql-snow) {
  border: none;
  border-bottom: 1px solid #eef1f5;
  background: linear-gradient(180deg, #f9fafb 0%, #f3f4f6 100%);
  border-radius: 10px 10px 0 0;
  padding: 8px 10px;
}

:deep(.ql-container.ql-snow) {
  border: none;
  font-family: inherit;
  font-size: 14px;
  line-height: 1.7;
}

:deep(.ql-editor) {
  min-height: 280px;
  max-height: 520px;
  overflow-y: auto;
  padding: 14px 18px;
  color: #1f2937;
}

:deep(.ql-editor.ql-blank::before) {
  color: #9ca3af;
  font-style: normal;
}

:deep(.ql-snow .ql-picker.ql-header .ql-picker-label::before),
:deep(.ql-snow .ql-picker.ql-header .ql-picker-item::before) {
  content: attr(data-label);
}
</style>
