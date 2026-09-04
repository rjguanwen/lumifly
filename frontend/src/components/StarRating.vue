<template>
  <span class="inline-flex items-center star-rating" :class="{ 'is-editable': editable }">
    <template v-for="i in 5" :key="i">
      <span
        class="star"
        :class="{
          'filled': i <= display,
          'editable': editable,
        }"
        :style="editable ? { fontSize: size + 'px' } : {}"
        @click="editable && emit('update:modelValue', i === props.modelValue ? 0 : i)"
      >★</span>
    </template>
    <span v-if="modelValue && showLabel" class="star-label">{{ modelValue }} 星</span>
  </span>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: { type: Number, default: 0 },
  editable: { type: Boolean, default: false },
  size: { type: Number, default: 18 },
  showLabel: { type: Boolean, default: true },
})
const emit = defineEmits(['update:modelValue'])

const display = computed(() => props.modelValue || 0)
</script>

<style scoped>
.star {
  color: #e5e7eb;
  line-height: 1;
  display: inline-block;
}

.star.filled {
  color: #f5a623;
}

.star.editable {
  cursor: pointer;
  transition: transform 0.1s ease, color 0.1s ease;
  font-size: inherit;
}

.star.editable:hover {
  transform: scale(1.2);
  color: #f0a51f;
}

.star-label {
  margin-left: 6px;
  font-size: 12px;
  color: #9ca3af;
}
</style>
