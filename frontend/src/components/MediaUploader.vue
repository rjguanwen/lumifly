<template>
  <div
    class="w-full border-2 border-dashed border-gray-300 rounded-lg p-6 text-center transition-colors bg-white"
    :class="{ 'border-blue-500 bg-blue-50': isDragging }"
    @dragenter.prevent="isDragging = true"
    @dragover.prevent
    @dragleave.prevent="isDragging = false"
    @drop.prevent="handleDrop"
  >
    <input ref="fileInput" type="file" :accept="accept" :multiple="multiple" class="hidden" @change="handleFileSelect" />

    <div v-if="uploads.length === 0" class="cursor-pointer" @click="fileInput?.click()">
      <el-icon :size="40" color="#909399"><UploadFilled /></el-icon>
      <p class="mt-2 text-sm text-gray-600">点击或拖拽上传文件</p>
      <p class="text-xs text-gray-400 mt-1">支持 JPG, PNG, GIF, WebP, MP4, WebM</p>
    </div>

    <div v-else class="space-y-3">
      <div v-for="(upload, idx) in uploads" :key="idx" class="flex items-center gap-3 p-2 bg-gray-50 rounded">
        <img
          v-if="upload.url && upload.mimeType?.startsWith('image/')"
          :src="upload.url"
          class="w-12 h-12 object-cover rounded"
        />
        <el-icon v-else-if="upload.mimeType?.startsWith('video/')" :size="24" color="#909399"><VideoCamera /></el-icon>
        <div class="flex-1 text-left">
          <p class="text-sm font-medium text-gray-700 truncate">{{ upload.fileName }}</p>
          <p v-if="upload.uploading" class="text-xs text-blue-600">上传中...</p>
          <p v-else class="text-xs text-green-600">已上传</p>
        </div>
        <el-icon class="cursor-pointer hover:text-red-500" @click="removeUpload(idx)"><Close /></el-icon>
      </div>
      <button type="button" class="text-sm text-blue-600 hover:underline" @click="fileInput?.click()">
        继续添加
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { uploadApi } from '../api'
import { ElMessage } from 'element-plus'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
  accept: { type: String, default: '' },
  multiple: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'uploaded'])

const fileInput = ref(null)
const isDragging = ref(false)
const uploads = ref([])

const accept = computed(() => props.accept || 'image/*,video/mp4,video/webm')

async function uploadFile(file) {
  const upload = { fileName: file.name, mimeType: file.type, uploading: true }
  uploads.value.push(upload)
  const idx = uploads.value.length - 1
  try {
    const result = await uploadApi.upload(file)
    uploads.value[idx] = {
      id: result.id,
      url: result.url,
      fileName: result.fileName,
      mimeType: result.mimeType,
      uploading: false,
    }
    emitIds()
    emit('uploaded', { id: result.id, url: result.url })
  } catch {
    uploads.value.splice(idx, 1)
  }
}

function emitIds() {
  const ids = uploads.value.filter((u) => u.id).map((u) => u.id)
  emit('update:modelValue', ids)
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

function removeUpload(idx) {
  uploads.value.splice(idx, 1)
  emitIds()
}
</script>
