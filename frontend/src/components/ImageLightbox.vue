<template>
  <el-dialog
    :model-value="visible"
    width="auto"
    align-center
    :show-close="false"
    append-to-body
    class="gallery-dialog"
    @update:model-value="emit('update:visible', $event)"
  >
    <div class="lf-gallery">
      <!-- 左切换 -->
      <button v-if="images.length > 1" type="button" class="lf-g-nav lf-g-prev" @click.stop="step(-1)">
        <el-icon :size="26"><ArrowLeft /></el-icon>
      </button>

      <div class="lf-g-stage" @click="emit('update:visible', false)">
        <img v-if="current" :src="current.url" :alt="current.alt || ''" class="lf-g-img" @click.stop />
      </div>

      <!-- 右切换 -->
      <button v-if="images.length > 1" type="button" class="lf-g-nav lf-g-next" @click.stop="step(1)">
        <el-icon :size="26"><ArrowRight /></el-icon>
      </button>

      <!-- 计数 -->
      <div v-if="images.length > 1" class="lf-g-count">{{ index + 1 }} / {{ images.length }}</div>
    </div>
  </el-dialog>
</template>

<script setup>
import { computed, watch } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  images: { type: Array, default: () => [] }, // [{ url, alt }]
  imageIndex: { type: Number, default: 0 },
})
const emit = defineEmits(['update:visible', 'update:imageIndex'])

const images = computed(() => props.images || [])
const index = computed(() => {
  const i = Math.round(props.imageIndex)
  if (!images.value.length) return 0
  return Math.max(0, Math.min(i, images.value.length - 1))
})
const current = computed(() => images.value[index.value])

function step(delta) {
  if (images.value.length <= 1) return
  const next = (index.value + delta + images.value.length) % images.value.length
  emit('update:imageIndex', next)
}

// 键盘切换
function onKeydown(e) {
  if (!props.visible) return
  if (e.key === 'ArrowLeft') step(-1)
  else if (e.key === 'ArrowRight') step(1)
  else if (e.key === 'Escape') emit('update:visible', false)
}

watch(
  () => props.visible,
  (v) => {
    if (v) window.addEventListener('keydown', onKeydown)
    else window.removeEventListener('keydown', onKeydown)
  },
  { immediate: true },
)
</script>

<style scoped>
.lf-gallery {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  user-select: none;
}

.lf-g-stage {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  max-width: 86vw;
  max-height: 84vh;
  cursor: zoom-out;
}

.lf-g-img {
  display: block;
  max-width: 86vw;
  max-height: 84vh;
  border-radius: 10px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.35);
}

.lf-g-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 42px;
  height: 42px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  border: none;
  cursor: pointer;
  background: rgba(255, 255, 255, 0.16);
  color: #fff;
  backdrop-filter: blur(4px);
  transition: background-color 0.15s ease, transform 0.15s ease;
  z-index: 2;
}

.lf-g-nav:hover {
  background: rgba(255, 255, 255, 0.32);
  transform: translateY(-50%) scale(1.06);
}

.lf-g-prev {
  left: 12px;
}

.lf-g-next {
  right: 12px;
}

.lf-g-count {
  position: absolute;
  bottom: -38px;
  left: 50%;
  transform: translateX(-50%);
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
  letter-spacing: 1px;
  background: rgba(0, 0, 0, 0.35);
  padding: 3px 14px;
  border-radius: 999px;
}
</style>

<style>
/* 弹窗容器：透明暗底，无内边距 */
.gallery-dialog.el-dialog {
  background: rgba(10, 12, 20, 0.88);
  box-shadow: none;
  padding: 0;
  border-radius: 12px;
  overflow: hidden;
}

.gallery-dialog .el-dialog__header {
  display: none;
}

.gallery-dialog .el-dialog__body {
  padding: 0;
}
</style>
