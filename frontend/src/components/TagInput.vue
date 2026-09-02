<template>
  <div class="flex flex-wrap gap-2 items-center w-full bg-white border border-gray-200 rounded-lg px-2 py-1.5">
    <span
      v-for="tag in selectedTags"
      :key="tag.id"
      class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
      :style="{ backgroundColor: (tag.color || '#e5e7eb') + '20', color: tag.color || '#374151' }"
    >
      {{ tag.name }}
      <el-icon :size="12" class="cursor-pointer hover:opacity-70" @click="removeTag(tag.id)"><Close /></el-icon>
    </span>
    <div class="relative">
      <input
        v-model="inputValue"
        class="text-sm border-0 outline-none bg-transparent flex-1 min-w-[90px]"
        placeholder="添加标签..."
        @keydown.enter.prevent="createOrSelectTag"
        @focus="showDropdown = true"
      />
      <div
        v-if="showDropdown && filteredTags.length > 0"
        class="absolute top-full left-0 mt-1 w-48 bg-white border border-gray-200 rounded-lg shadow-lg z-10 max-h-40 overflow-auto"
      >
        <button
          v-for="tag in filteredTags"
          :key="tag.id"
          type="button"
          class="w-full text-left px-3 py-2 text-sm hover:bg-gray-50"
          @mousedown.prevent="selectTag(tag)"
        >
          {{ tag.name }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { tagApi } from '../api'

const props = defineProps({
  modelValue: { type: Array, default: () => [] },
})

const emit = defineEmits(['update:modelValue'])

const inputValue = ref('')
const showDropdown = ref(false)
const allTags = ref([])

const selectedTagIds = computed(() => props.modelValue || [])
const selectedTags = computed(() => allTags.value.filter((t) => selectedTagIds.value.includes(t.id)))
const filteredTags = computed(() =>
  allTags.value
    .filter((t) => !selectedTagIds.value.includes(t.id))
    .filter((t) => !inputValue.value || t.name.toLowerCase().includes(inputValue.value.toLowerCase())),
)

onMounted(async () => {
  try {
    allTags.value = await tagApi.list()
  } catch {
    /* 忽略 */
  }
})

function selectTag(tag) {
  emit('update:modelValue', [...selectedTagIds.value, tag.id])
  inputValue.value = ''
  showDropdown.value = false
}

function removeTag(tagId) {
  emit('update:modelValue', selectedTagIds.value.filter((id) => id !== tagId))
}

async function createOrSelectTag() {
  const name = inputValue.value.trim()
  if (!name) return
  const existing = allTags.value.find((t) => t.name.toLowerCase() === name.toLowerCase())
  if (existing) {
    selectTag(existing)
    return
  }
  try {
    const tag = await tagApi.create({ name })
    allTags.value.push(tag)
    selectTag(tag)
  } catch {
    /* 忽略 */
  }
}
</script>
