<template>
  <div class="flex flex-wrap gap-2">
    <button
      v-for="opt in moodOptions"
      :key="opt.value"
      type="button"
      class="px-3 py-1.5 rounded-full text-sm border transition-all"
      :class="
        props.modelValue === opt.value
          ? 'border-blue-500 bg-blue-50 text-blue-600 font-medium'
          : 'border-gray-200 bg-white text-gray-600 hover:border-blue-300'
      "
      @click="emit('update:modelValue', props.modelValue === opt.value ? null : opt.value)"
    >
      {{ opt.label }} {{ moodMap[opt.value] }}
    </button>
  </div>
</template>

<script setup>
import { moodOptions, moodMap } from '../utils/helpers'

const props = defineProps({
  modelValue: { type: String, default: null },
})
const emit = defineEmits(['update:modelValue'])
</script>
