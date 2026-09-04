<template>
  <div
    class="ua-round"
    :style="{ width: size + 'px', height: size + 'px', fontSize: Math.round(size * 0.42) + 'px' }"
  >
    <img v-if="src" :src="src" :alt="name" class="ua-img" />
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
