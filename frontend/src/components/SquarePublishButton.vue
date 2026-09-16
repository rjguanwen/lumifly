<script setup>
import { Check, Clock, RefreshRight, Share } from '@element-plus/icons-vue'

// 列表页「发布到广场」状态按钮：未发布 / 已发布 / 审核中 / 未通过
defineProps({
  state: { type: Object, default: null }, // { id, status }
  loading: { type: Boolean, default: false },
})
defineEmits(['publish', 'unpublish', 'pending'])
</script>

<template>
  <el-button
    v-if="!state"
    size="small"
    type="primary"
    plain
    :loading="loading"
    @click="$emit('publish')"
  >
    <el-icon class="mr-0.5"><Share /></el-icon>发布
  </el-button>
  <el-button
    v-else-if="state.status === 'published'"
    size="small"
    type="success"
    :loading="loading"
    @click="$emit('unpublish')"
  >
    <el-icon class="mr-0.5"><Check /></el-icon>已发布
  </el-button>
  <el-button
    v-else-if="state.status === 'pending'"
    size="small"
    type="warning"
    plain
    :loading="loading"
    @click="$emit('pending')"
  >
    <el-icon class="mr-0.5"><Clock /></el-icon>审核中
  </el-button>
  <el-button
    v-else
    size="small"
    type="danger"
    plain
    :loading="loading"
    @click="$emit('publish')"
  >
    <el-icon class="mr-0.5"><RefreshRight /></el-icon>未通过
  </el-button>
</template>
