<template>
  <div class="space-y-4">
    <div class="flex items-center gap-2">
      <el-tag size="small" :type="priorityType(item.priority)">{{ priorityLabel(item.priority) }}</el-tag>
      <el-tag size="small" type="info">{{ statusLabel(item.status) }}</el-tag>
    </div>
    <h3 class="text-lg font-semibold text-gray-900">{{ item.title }}</h3>
    <div v-if="item.description" class="prose prose-sm" v-html="item.description" />
    <div class="text-sm text-gray-500 space-y-1">
      <div v-if="item.startDate">开始：{{ item.startDate }}</div>
      <div v-if="item.targetDate">目标：{{ item.targetDate }}</div>
    </div>
  </div>
</template>

<script setup>
import { priorityType, priorityLabel, statusLabel } from '../../utils/helpers'

defineProps({ item: { type: Object, required: true } })
</script>
