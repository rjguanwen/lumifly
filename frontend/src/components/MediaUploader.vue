<template>
  <div
    class="w-full border-2 border-dashed border-gray-300 rounded-xl px-4 py-4 text-center transition-colors bg-white"
    :class="{ 'border-blue-500 bg-blue-50': isDragging }"
    @dragenter.prevent="isDragging = true"
    @dragover.prevent
    @dragleave.prevent="isDragging = false"
    @drop.prevent="handleDrop"
  >
    <input
      ref="fileInput"
      type="file"
      :accept="accept"
      :multiple="multiple"
      class="hidden"
      @change="handleFileSelect"
    />

    <!-- 已上传附件缩略列表（含删除） -->
    <div v-if="uploads.length" class="space-y-2 mb-3 text-left">
      <div
        v-for="(upload, idx) in uploads"
        :key="upload.key || upload.id"
        class="media-item flex items-center gap-3 p-2 bg-gray-50 rounded-lg border border-gray-100"
      >
        <img
          v-if="upload.mimeType?.startsWith('image/') && upload.url"
          :src="upload.thumbUrl || thumbOf(upload)"
          width="44"
          height="44"
          loading="lazy"
          decoding="async"
          class="w-11 h-11 rounded object-cover shrink-0 bg-gray-100"
        />
        <div
          v-else-if="upload.mimeType?.startsWith('video/')"
          class="w-11 h-11 rounded bg-blue-50 text-blue-500 flex items-center justify-center shrink-0"
        >
          <el-icon :size="20"><VideoCamera /></el-icon>
        </div>
        <div class="flex-1 min-w-0">
          <p class="text-sm font-medium text-gray-700 truncate">{{ upload.fileName }}</p>
          <p class="text-xs mt-0.5" :class="upload.uploading ? 'text-blue-500' : 'text-green-600'">
            {{ upload.uploading ? '上传中...' : (upload.initial ? '已上传' : '已上传') }}
          </p>
        </div>
        <button
          type="button"
          class="media-remove w-7 h-7 flex items-center justify-center rounded-full text-gray-400 hover:text-red-500 hover:bg-red-50 transition-colors"
          :disabled="upload.uploading"
          @click="removeUpload(upload, idx)"
        >
          <el-icon><Close /></el-icon>
        </button>
      </div>
    </div>

    <!-- 空态引导 -->
    <div v-if="!uploads.length" class="cursor-pointer py-2" @click="fileInput?.click()">
      <el-icon :size="34" color="#9ca3af"><UploadFilled /></el-icon>
      <p class="mt-2 text-sm text-gray-500">点击或拖拽上传附件</p>
      <p class="text-xs text-gray-400 mt-1">图片（jpg/png/gif/webp）· 视频（mp4/webm）</p>
    </div>

    <!-- 继续添加 -->
    <div v-if="uploads.length" class="flex items-center justify-center gap-2">
      <el-button size="small" plain @click="fileInput?.click()">
        <el-icon class="mr-1"><Plus /></el-icon>继续上传
      </el-button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { uploadApi } from '../api'

// 附件允许的图片类型。与后端 upload.go 的 allowedImageTypes 逐项对齐：
// 后端只接受这四种图片，多写会让用户选中后才被拒，少写会挡住本来支持的格式。
// 用「扩展名 + MIME」两种写法：Windows 文件选择框按扩展名过滤更可靠，
// 只写 MIME 时部分系统会把 .jpg 也置灰。
const DEFAULT_ACCEPT = [
  '.jpg',
  '.jpeg',
  '.png',
  '.gif',
  '.webp',
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
  '.mp4',
  '.webm',
  'video/mp4',
  'video/webm',
].join(',')

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  initialItems: { type: Array, default: () => [] }, // 编辑时已有关联附件 [{id,url,fileName,mimeType}]
  // 默认空串 → 回落到 DEFAULT_ACCEPT。注意这里必须用 computed 得到**字符串**：
  // 曾经写成 `const accept = () => props.accept || '...'` 并按 `:accept="accept"` 绑定，
  // 结果传给 <input> 的是函数源码 "() => props.accept || ..."，
  // 浏览器无法识别这个 accept 值，就回退成它自己的兜底类型（表现为
  // 文件选择框默认显示「自定义文件(*.m4v;*.mp4)」）。
  accept: { type: String, default: '' },
  multiple: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue'])

const fileInput = ref(null)
const isDragging = ref(false)
const uploads = ref([])

const accept = computed(() => {
  const custom = (props.accept || '').trim()
  return custom || DEFAULT_ACCEPT
})

// 44px 的格子不该拉原图（单张可达数 MB）。接口没给 thumbUrl 时按同规则兜底拼一个，
// 后端 /api/uploads 支持 ?w=64；万一生成失败会自动回落原图，不会裂图。
function thumbOf(upload) {
  if (!upload.url || !upload.mimeType?.startsWith('image/')) return upload.url
  if (!upload.url.startsWith('/api/uploads/')) return upload.url
  return `${upload.url}?w=64`
}

let keySeq = 0

// 依据父传入的已有关联附件初始化列表
watch(
  () => props.initialItems,
  (items) => {
    uploads.value = (items || []).map((it) => ({
      id: it.id,
      url: it.url,
      thumbUrl: it.thumbUrl || null,
      fileName: it.fileName,
      mimeType: it.mimeType,
      initial: true,
    }))
  },
  { immediate: true, deep: true },
)

function emitIds() {
  emit('update:modelValue', uploads.value.filter((u) => u.id).map((u) => u.id))
}

async function uploadFile(file) {
  const upload = { key: `new-${keySeq++}`, fileName: file.name, mimeType: file.type, uploading: true }
  uploads.value.push(upload)
  try {
    const result = await uploadApi.upload(file)
    upload.id = result.id
    upload.url = result.url
    upload.thumbUrl = result.thumbUrl || null
    upload.fileName = result.fileName || file.name
    upload.mimeType = result.mimeType || file.type
    upload.uploading = false
    emitIds()
  } catch {
    uploads.value = uploads.value.filter((u) => u !== upload)
  }
}

function handleFileSelect(e) {
  const files = e.target.files
  if (!files) return
  for (const file of files) uploadFile(file)
  if (fileInput.value) fileInput.value.value = ''
}

function handleDrop(e) {
  isDragging.value = false
  const files = e.dataTransfer?.files
  if (!files) return
  for (const file of files) uploadFile(file)
}

async function removeUpload(upload, idx) {
  if (upload.uploading) return
  if (upload.id) {
    try {
      await ElMessageBox.confirm('删除该附件？文件将被永久删除，且从当前内容中移除。', '删除附件', {
        type: 'warning',
        confirmButtonText: '删除',
      })
    } catch {
      return
    }
    try {
      await uploadApi.remove(upload.id)
    } catch {
      return
    }
  }
  uploads.value.splice(idx, 1)
  emitIds()
}
</script>
