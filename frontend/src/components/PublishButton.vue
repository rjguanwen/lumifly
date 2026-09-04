<template>
  <el-button
    size="small"
    :type="publishedId ? 'success' : 'primary'"
    :plain="!publishedId"
    :loading="busy"
    @click="toggle"
  >
    <el-icon v-if="!publishedId" class="mr-1"><Share /></el-icon>
    <el-icon v-else class="mr-1"><Check /></el-icon>
    {{ publishedId ? '已发布到广场' : '发布到广场' }}
  </el-button>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { squareApi } from '../api'

const props = defineProps({
  sourceType: { type: String, required: true },
  sourceId: { type: [Number, String], required: true },
})

const publishedId = ref(0)
const busy = ref(false)

async function load() {
  try {
    const data = await squareApi.mine()
    const match = (data.items || []).find(
      (p) => p.sourceType === props.sourceType && Number(p.sourceId) === Number(props.sourceId),
    )
    publishedId.value = match ? match.id : 0
  } catch { /* 忽略 */ }
}
onMounted(load)

async function toggle() {
  if (publishedId.value) {
    try {
      await ElMessageBox.confirm('撤回后将不再在广场展示（他人已点赞/评论也会清除），确定撤回？', '撤回发布', {
        type: 'warning',
      })
    } catch {
      return
    }
    busy.value = true
    try {
      await squareApi.unpublish(publishedId.value)
      publishedId.value = 0
      ElMessage.success('已从广场撤回')
    } catch { /* 拦截器已提示 */ } finally {
      busy.value = false
    }
  } else {
    busy.value = true
    try {
      const item = await squareApi.publish(props.sourceType, Number(props.sourceId))
      publishedId.value = item.id
      ElMessage.success('已发布到广场')
    } catch { /* 拦截器已提示 */ } finally {
      busy.value = false
    }
  }
}
</script>
