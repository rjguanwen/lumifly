<template>
  <div class="editor-shell">
    <!-- 模式切换栏 -->
    <div class="editor-topbar">
      <el-radio-group v-model="mode" size="small">
        <el-radio-button value="rich">富文本</el-radio-button>
        <el-radio-button value="markdown">Markdown</el-radio-button>
      </el-radio-group>
      <div v-if="mode === 'markdown'" class="flex items-center gap-3">
        <span class="text-xs text-gray-400 hidden sm:inline"># 标题 · **加粗** · - 列表 · ``` 代码</span>
        <el-checkbox v-model="showPreview" size="small">实时预览</el-checkbox>
      </div>
    </div>

    <!-- ===== 富文本模式：内容撑开高度，滚动交由外层弹窗 ===== -->
    <template v-if="mode === 'rich'">
      <div v-if="editor" class="editor-toolbar">
        <template v-for="(group, gi) in toolbarGroups" :key="gi">
          <div v-if="gi > 0" class="tb-divider" />
          <el-tooltip
            v-for="action in group"
            :key="action.title"
            :content="action.title"
            placement="top"
            :show-after="150"
          >
            <button
              type="button"
              class="editor-tb-btn"
              :class="[action.textCls, { 'is-active': action.isActive() }]"
              @mousedown.prevent
              @click="action.command()"
            >
              <span>{{ action.text }}</span>
            </button>
          </el-tooltip>
        </template>
        <div class="tb-divider" />
        <el-tooltip content="插入图片" placement="top" :show-after="150">
          <button type="button" class="editor-tb-btn" @mousedown.prevent @click="triggerImageUpload">
            <el-icon :size="14"><Picture /></el-icon>
          </button>
        </el-tooltip>
      </div>

      <div class="tiptap-editor editor-area" @click="focusEditor">
        <EditorContent :editor="editor" />
      </div>
      <input ref="fileInput" type="file" accept="image/*" class="hidden" @change="handleImageUpload" />
    </template>

    <!-- ===== Markdown 模式：左右分栏 编辑 | 预览 ===== -->
    <template v-else>
      <div class="md-panes" :class="{ 'is-split': showPreview }">
        <textarea
          v-model="mdText"
          class="md-editor-textarea md-pane-input"
          :placeholder="placeholder"
          spellcheck="false"
        />
        <div v-if="showPreview" class="md-editor-preview md-pane-preview" v-html="mdHtml" />
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useEditor, EditorContent } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Image from '@tiptap/extension-image'
import Link from '@tiptap/extension-link'
import Placeholder from '@tiptap/extension-placeholder'
import { marked } from 'marked'
import TurndownService from 'turndown'
import { uploadApi } from '../api'
import { ElMessage } from 'element-plus'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '开始写作...' },
})

const emit = defineEmits(['update:modelValue'])

const mode = ref('rich')
const showPreview = ref(true)
const fileInput = ref(null)
const mdText = ref('')

// ---------- 转换工具 ----------
const turndown = new TurndownService({ headingStyle: 'atx', codeBlockStyle: 'fenced', bulletListMarker: '-' })
const renderMd = (md) => marked.parse(md || '')

const mdHtml = computed(() => renderMd(mdText.value))

function emitHtml(html) {
  const v = html || ''
  if (v !== props.modelValue) emit('update:modelValue', v)
}

// ---------- 编辑器 ----------
const editor = useEditor({
  content: props.modelValue || '',
  extensions: [
    StarterKit,
    Image,
    Link.configure({ openOnClick: false }),
    Placeholder.configure({ placeholder: props.placeholder }),
  ],
  editorProps: {
    attributes: { spellcheck: 'false' },
  },
  onUpdate({ editor }) {
    if (mode.value === 'rich') emitHtml(editor.getHTML())
  },
})

function focusEditor() {
  editor.value?.commands.focus()
}

// ---------- 富文本工具栏 ----------
const toolbarGroups = computed(() => {
  const e = editor.value
  if (!e) return []
  return [
    [
      { title: '粗体', text: 'B', textCls: 'font-bold', command: () => e.chain().focus().toggleBold().run(), isActive: () => e.isActive('bold') },
      { title: '斜体', text: 'I', textCls: 'italic', command: () => e.chain().focus().toggleItalic().run(), isActive: () => e.isActive('italic') },
      { title: '删除线', text: 'S', textCls: 'line-through', command: () => e.chain().focus().toggleStrike().run(), isActive: () => e.isActive('strike') },
    ],
    [
      { title: '标题', text: 'H2', command: () => e.chain().focus().toggleHeading({ level: 2 }).run(), isActive: () => e.isActive('heading', { level: 2 }) },
      { title: '小标题', text: 'H3', command: () => e.chain().focus().toggleHeading({ level: 3 }).run(), isActive: () => e.isActive('heading', { level: 3 }) },
    ],
    [
      { title: '无序列表', text: '≡', command: () => e.chain().focus().toggleBulletList().run(), isActive: () => e.isActive('bulletList') },
      { title: '有序列表', text: '1.', command: () => e.chain().focus().toggleOrderedList().run(), isActive: () => e.isActive('orderedList') },
    ],
    [
      { title: '引用', text: '❝', command: () => e.chain().focus().toggleBlockquote().run(), isActive: () => e.isActive('blockquote') },
      { title: '代码块', text: '</>', textCls: 'font-mono', command: () => e.chain().focus().toggleCodeBlock().run(), isActive: () => e.isActive('codeBlock') },
    ],
  ]
})

// ---------- 模式切换 ----------
watch(mode, async (to, from) => {
  if (to === from) return
  if (to === 'markdown') {
    const html = editor.value ? editor.value.getHTML() : props.modelValue || ''
    mdText.value = turndown.turndown(html)
    await nextTick()
    focusMarkdown()
  } else {
    if (editor.value) {
      const html = renderMd(mdText.value)
      emitHtml(html)
      editor.value.commands.setContent(html, false)
      await nextTick()
      editor.value.commands.focus('start')
    }
  }
})

watch(mdText, (v) => {
  if (mode.value === 'markdown') emitHtml(renderMd(v))
})

watch(
  () => props.modelValue,
  (val) => {
    if (!editor.value) return
    if (mode.value === 'markdown') {
      const md = turndown.turndown(val || '')
      if (md !== mdText.value) mdText.value = md
    } else if (editor.value.getHTML() !== (val || '')) {
      editor.value.commands.setContent(val || '', false)
    }
  },
)

function focusMarkdown() {
  const el = document.querySelector('.editor-shell .md-editor-textarea')
  el?.focus()
}

// ---------- 图片上传（富文本内嵌） ----------
function triggerImageUpload() {
  fileInput.value?.click()
}

async function handleImageUpload(e) {
  const file = e.target.files?.[0]
  if (!file) return
  try {
    const result = await uploadApi.upload(file)
    editor.value?.chain().focus().setImage({ src: result.url }).run()
  } catch {
    ElMessage.error('图片上传失败')
  }
  if (fileInput.value) fileInput.value.value = ''
}

onBeforeUnmount(() => {
  editor.value?.destroy()
})
</script>

<style scoped>
/* ===== 外壳：与标题等宽，满宽显示 ===== */
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

/* ===== 模式切换栏 ===== */
.editor-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px;
  padding: 8px 12px;
  background: linear-gradient(180deg, #f9fafb 0%, #f3f4f6 100%);
  border-bottom: 1px solid #e5e7eb;
}

/* ===== 文字化工具栏 ===== */
.editor-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 2px;
  padding: 6px 8px;
  border-bottom: 1px solid #f3f4f6;
  background: #fff;
}

.tb-divider {
  width: 1px;
  height: 18px;
  background: #e5e7eb;
  margin: 0 6px;
}

.editor-tb-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 30px;
  height: 28px;
  padding: 0 7px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: #4b5563;
  font-size: 13px;
  cursor: pointer;
  transition: background-color 0.12s ease, color 0.12s ease;
  line-height: 1;
  user-select: none;
}

.editor-tb-btn:hover {
  background: #f3f4f6;
  color: #111827;
}

.editor-tb-btn.is-active {
  background: #eef2ff;
  color: #3b5bdb;
  font-weight: 600;
}

.editor-tb-btn:active {
  transform: scale(0.96);
}

/* ===== 富文本编辑区：更宽更大，随内容撑高 ===== */
.editor-area {
  min-height: 420px;
  padding: 18px 20px;
}

/* ===== Markdown 左右分栏 ===== */
.md-panes {
  display: flex;
  align-items: stretch;
  padding: 12px;
  gap: 12px;
}

.md-panes:not(.is-split) .md-pane-input {
  flex: 1;
}

.md-panes.is-split .md-pane-input,
.md-panes.is-split .md-pane-preview {
  flex: 1 1 0;
}

.md-pane-input {
  width: 100%;
  min-height: 440px;
  height: auto;
  max-height: 620px;
  overflow-y: auto;
  padding: 14px 16px;
  resize: vertical;
}

.md-pane-preview {
  min-height: 440px;
  max-height: 620px;
  overflow-y: auto;
  padding: 12px 16px;
  background: #fafbfd;
  border: 1px solid #eef1f6;
  border-radius: 8px;
}
</style>
