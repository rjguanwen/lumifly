<template>
  <div
    class="ua-round"
    :style="{ width: size + 'px', height: size + 'px', fontSize: Math.round(size * 0.42) + 'px' }"
  >
    <img v-if="src" :src="imgSrc" :alt="name" :width="size" :height="size" decoding="async" class="ua-img" />
    <span v-else class="ua-char" :style="{ background: bg }">{{ (name || '?').charAt(0) }}</span>
  </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  name: { type: String, default: '' },
  src: { type: String, default: '' },
  size: { type: Number, default: 32 },
})

const palettes = [
  'linear-gradient(135deg, #5b7cff, #9333ea)',
  'linear-gradient(135deg, #22a06b, #4a6cf7)',
  'linear-gradient(135deg, #f5a623, #ef4444)',
  'linear-gradient(135deg, #0ea5e9, #6366f1)',
  'linear-gradient(135deg, #14b8a6, #8b5cf6)',
  'linear-gradient(135deg, #f97316, #ec4899)',
]

// 头像按显示尺寸取缩略图：18px 的头像没必要下载 200KB 的原图。
// 后端 /api/uploads 只接受白名单档位（32/64/128/256/512），这里就近向上取档，
// 非 /api/uploads/ 来源（如外链、data URI）原样返回。
const THUMB_STEPS = [32, 64, 128, 256, 512]
const imgSrc = computed(() => {
  const s = props.src
  if (!s || !s.startsWith('/api/uploads/')) return s
  // 2x DPR：显示 18px 需要 36 物理像素 → 取 64 档
  const need = props.size * 2
  const step = THUMB_STEPS.find((v) => v >= need) || THUMB_STEPS[THUMB_STEPS.length - 1]
  const joined = s.includes('?') ? `${s}&w=${step}` : `${s}?w=${step}`
  return joined
})

const bg = computed(() => {
  const s = props.name || '?'
  let h = 0
  for (let i = 0; i < s.length; i++) h = (h * 31 + s.charCodeAt(i)) % 997
  return palettes[h % palettes.length]
})
</script>

<style scoped>
.ua-round {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  background: #e5e7eb;
}

.ua-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.ua-char {
  color: #fff;
  font-weight: 600;
  line-height: 1;
  user-select: none;
}
</style>
