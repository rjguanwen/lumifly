<template>
  <div class="editor-shell">
    <div ref="hostEl" class="rich-host"></div>
    <input ref="fileEl" type="file" accept="image/*" class="hidden" @change="onPickImageFile" />
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

const hostEl = ref(null)
const fileEl = ref(null)
let quill = null
let syncing = false

// 所见即所得工具栏（图标直观，标题下拉可选）
const TOOLBAR = [
  [{ header: [false, 1, 2, 3] }],
  ['bold', 'italic', 'underline', 'strike'],
  ['blockquote', 'code-block'],
  [{ list: 'ordered' }, { list: 'bullet' }],
  ['link', 'image'],
  ['clean'],
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
      toolbar: TOOLBAR,
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

  // 自定义「插入图片 / 链接」按钮行为
  const toolbar = hostEl.value.closest('.editor-shell')?.querySelector('.ql-toolbar')
  toolbar?.querySelector('.ql-image')?.addEventListener('click', (e) => {
    e.preventDefault()
    fileEl.value?.click()
  })
  toolbar?.querySelector('.ql-link')?.addEventListener('click', (e) => {
    e.preventDefault()
    promptLink()
  })
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
    if (!url || !quill) return
    const sel = quill.getSelection() || { index: quill.getLength(), length: 0 }
    quill.insertEmbed(sel.index, 'image', url)
    quill.setSelection(sel.index + 1, 0)
  } catch {
    ElMessage.error('图片上传失败')
  }
}

// 粘贴/拖放图片：走服务端上传后只插入 URL，不内联 base64。
// 替换语义与 Quill 原实现一致：若粘贴时选区非空，先删除选中内容再插入。
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
  if (!urls.length) {
    ElMessage.error('图片上传失败')
    return
  }
  if (range.length > 0) quill.deleteText(range.index, range.length, 'silent')
  let at = range.index
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
