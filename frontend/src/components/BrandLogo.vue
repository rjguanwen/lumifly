<template>
  <svg :width="size" :height="size" viewBox="0 0 128 128" :aria-label="'飞光 Lumifly'">
    <defs>
      <radialGradient :id="uid + 'bg'" cx="50%" cy="38%" r="78%">
        <stop offset="0%" stop-color="#8FB0FF" />
        <stop offset="38%" stop-color="#5B7CFF" />
        <stop offset="100%" stop-color="#21409F" />
      </radialGradient>
      <radialGradient :id="uid + 'sun'" cx="50%" cy="45%" r="62%">
        <stop offset="0%" stop-color="#FFF8D6" />
        <stop offset="45%" stop-color="#FFD766" />
        <stop offset="100%" stop-color="#F0990E" />
      </radialGradient>
      <linearGradient :id="uid + 'wing'" x1="0%" y1="0%" x2="0%" y2="100%">
        <stop offset="0%" stop-color="#FFFFFF" />
        <stop offset="100%" stop-color="#D9E6FF" />
      </linearGradient>
    </defs>

    <!-- 深空圆底 -->
    <circle cx="64" cy="64" r="62" :fill="'url(#' + uid + 'bg)'" />
    <circle cx="64" cy="64" r="62" fill="none" stroke="#FFFFFF" stroke-opacity="0.15" stroke-width="1.5" />

    <!-- 顶部环境光 -->
    <circle cx="44" cy="30" r="40" fill="#FFFFFF" opacity="0.07" />
    <circle cx="44" cy="30" r="20" fill="#FFFFFF" opacity="0.08" />

    <!-- 光羽双翼（展翅上扬） -->
    <path d="M64 66 C46 58 26 56 12 44" fill="none" :stroke="'url(#' + uid + 'wing)'" stroke-width="11" stroke-linecap="round" />
    <path d="M64 66 C82 58 102 56 116 44" fill="none" :stroke="'url(#' + uid + 'wing)'" stroke-width="11" stroke-linecap="round" />
    <!-- 翼下淡影 -->
    <path d="M64 66 C46 58 26 56 12 44" fill="none" stroke="#1B3180" stroke-opacity="0.18" stroke-width="3" stroke-linecap="round" transform="translate(0,4)" />
    <path d="M64 66 C82 58 102 56 116 44" fill="none" stroke="#1B3180" stroke-opacity="0.18" stroke-width="3" stroke-linecap="round" transform="translate(0,4)" />

    <!-- 中心光核 -->
    <circle cx="64" cy="76" r="21" fill="#FFFFFF" opacity="0.16" />
    <circle cx="64" cy="76" r="14.5" :fill="'url(#' + uid + 'sun)'" />
    <circle cx="58.5" cy="70.5" r="4.5" fill="#FFFFFF" opacity="0.9" />
  </svg>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  size: { type: [Number, String], default: 32 },
})

// 每个实例生成唯一 id，避免多实例渐变冲突
const uid = computed(() => `lf-${Math.random().toString(36).slice(2, 9)}-`)
</script>
