<template>
  <el-dialog
    :model-value="visible"
    width="720px"
    title="裁剪头像"
    :close-on-click-modal="false"
    append-to-body
    class="avatar-cropper-dialog"
    @update:model-value="onUpdate"
    @closed="teardown"
  >
    <div class="flex flex-col sm:flex-row gap-6">
      <!-- 裁剪编辑区 -->
      <div class="flex-1 min-w-0">
        <div class="avatar-crop-stage">
          <img ref="imgEl" :src="imageUrl" alt="待裁剪头像" />
        </div>
        <div class="flex items-center gap-3 mt-3 text-xs text-gray-500">
          <span>拖动调整范围，滚轮或按钮缩放</span>
          <div class="flex-1" />
          <el-button-group size="small">
            <el-button title="缩小" @click="zoomBy(-0.1)">－</el-button>
            <el-button title="放大" @click="zoomBy(0.1)">＋</el-button>
          </el-button-group>
          <el-button size="small" text type="primary" @click="resetView">重置</el-button>
        </div>
      </div>

      <!-- 效果预览 -->
      <div class="flex flex-col items-center justify-start shrink-0">
        <span class="text-xs text-gray-500 mb-3">预览效果（圆形展示）</span>
        <div class="w-32 h-32 rounded-full shadow-md overflow-hidden ring-4 ring-indigo-100 bg-gray-100 flex items-center justify-center">
          <img v-if="previewUrl" :src="previewUrl" alt="头像预览" class="w-full h-full object-cover" />
          <span v-else class="text-gray-300 text-xs">生成中…</span>
        </div>
        <p class="text-[11px] text-gray-400 mt-3 leading-relaxed text-center">确认后将按 256×256 正方形保存，<br />广场等处按圆形展示</p>
      </div>
    </div>

    <template #footer>
      <el-button @click="onUpdate(false)">取消</el-button>
      <el-button type="primary" :loading="processing" @click="confirmCrop">确认上传</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import Cropper from 'cropperjs'
import 'cropperjs/dist/cropper.css'

const props = defineProps({
  visible: { type: Boolean, default: false },
  imageUrl: { type: String, default: '' },
  fileName: { type: String, default: 'avatar.png' },
})
const emit = defineEmits(['update:visible', 'cropped'])

const imgEl = ref(null)
const cropper = ref(null)
const previewUrl = ref('')
const processing = ref(false)
let previewTimer = null

function teardown() {
  clearTimeout(previewTimer)
  if (cropper.value) {
    cropper.value.destroy()
    cropper.value = null
  }
  if (imgEl.value) imgEl.value.onload = null
  previewUrl.value = ''
}

function initCropper() {
  teardown()
  const img = imgEl.value
  if (!img) return
  if (img.complete && img.naturalWidth > 0) {
    createCropper()
  } else {
    img.onload = createCropper
  }
}

function createCropper() {
  if (!imgEl.value) return
  cropper.value = new Cropper(imgEl.value, {
    viewMode: 1,
    dragMode: 'move',
    aspectRatio: 1,
    autoCropArea: 0.9,
    background: false,
    guides: true,
    center: true,
    highlight: false,
    cropBoxMovable: true,
    cropBoxResizable: true,
    toggleDragModeOnDblclick: false,
    minCropBoxWidth: 60,
    minCropBoxHeight: 60,
    ready: () => updatePreview(),
    crop: () => {
      // 拖动/缩放过程中节流刷新右侧预览
      clearTimeout(previewTimer)
      previewTimer = setTimeout(updatePreview, 180)
    },
  })
}

function updatePreview() {
  const c = cropper.value
  if (!c) return
  try {
    const canvas = c.getCroppedCanvas({ width: 160, height: 160, imageSmoothingQuality: 'high' })
    previewUrl.value = canvas.toDataURL('image/png')
  } catch {
    /* 预览失败忽略 */
  }
}

function zoomBy(delta) {
  cropper.value?.zoom(delta)
}

function resetView() {
  cropper.value?.reset()
  updatePreview()
}

function onUpdate(val) {
  emit('update:visible', val)
}

async function confirmCrop() {
  const c = cropper.value
  if (!c) return
  processing.value = true
  try {
    const canvas = c.getCroppedCanvas({ width: 256, height: 256, imageSmoothingQuality: 'high' })
    const blob = await new Promise((resolve, reject) => {
      canvas.toBlob((b) => (b ? resolve(b) : reject(new Error('生成图片失败'))), 'image/png')
    })
    const base = (props.fileName || 'avatar').replace(/\.[^.]+$/, '')
    const file = new File([blob], `${base}-cropped.png`, { type: 'image/png' })
    emit('cropped', file)
    onUpdate(false)
  } catch {
    ElMessage.error('头像生成失败，请重试')
  } finally {
    processing.value = false
  }
}

watch(
  () => props.visible,
  (v) => {
    if (v) nextTick(initCropper)
    else teardown()
  },
)

onBeforeUnmount(teardown)
</script>

<style scoped>
.avatar-crop-stage {
  width: 100%;
  height: 340px;
  background: #111827;
  border-radius: 12px;
  overflow: hidden;
}

.avatar-crop-stage :deep(img) {
  display: block;
  max-width: 100%;
  max-height: 100%;
}
</style>
