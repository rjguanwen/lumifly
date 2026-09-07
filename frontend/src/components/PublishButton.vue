<template>
  <el-button
    v-if="!pub"
    size="small"
    type="primary"
    plain
    :loading="busy"
    @click="doPublish"
  >
    <el-icon class="mr-1"><Share /></el-icon>发布到广场
  </el-button>
  <el-button
    v-else-if="pub.status === 'published'"
    size="small"
    type="success"
    :loading="busy"
    @click="doUnpublish"
  >
    <el-icon class="mr-1"><Check /></el-icon>已发布到广场
  </el-button>
  <el-button
    v-else-if="pub.status === 'pending'"
    size="small"
    type="warning"
    plain
    @click="onPendingClick"
  >
    <el-icon class="mr-1"><Clock /></el-icon>审核中
  </el-button>
  <el-button
    v-else
    size="small"
    type="danger"
    plain
    :loading="busy"
    @click="doPublish"
  >
    <el-icon class="mr-1"><RefreshRight /></el-icon>未通过 · 重新提交
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

// pub: null=未发布 / {id,status}；status: published|pending|rejected
const pub = ref(null)
const busy = ref(false)

async function load() {
  try {
    const data = await squareApi.mine()
    const match = (data.items || []).find(
      (p) => p.sourceType === props.sourceType && Number(p.sourceId) === Number(props.sourceId),
    )
    pub.value = match ? { id: match.id, status: match.status || 'published' } : null
  } catch { /* 忽略 */ }
}
onMounted(load)

function afterPublish(item) {
  pub.value = { id: item.id, status: item.status || 'published' }
  if (pub.value.status === 'pending') {
    ElMessage.info('内容可能包含敏感信息，已转入人工审核，通过后将在广场展示')
  } else if (pub.value.status === 'rejected') {
    ElMessage.warning('该内容未能通过审核，请修改后重新提交')
  } else {
    ElMessage.success('已发布到广场')
  }
}

async function doPublish() {
  busy.value = true
  try {
    const item = await squareApi.publish(props.sourceType, Number(props.sourceId))
    afterPublish(item)
  } catch { /* 拦截器已提示 */ } finally {
    busy.value = false
  }
}

async function doUnpublish() {
  try {
    await ElMessageBox.confirm('撤回后将不再在广场展示（他人已点赞/评论也会清除），确定撤回？', '撤回发布', {
      type: 'warning',
    })
  } catch {
    return
  }
  busy.value = true
  try {
    await squareApi.unpublish(pub.value.id)
    pub.value = null
    ElMessage.success('已从广场撤回')
  } catch { /* 拦截器已提示 */ } finally {
    busy.value = false
  }
}

async function onPendingClick() {
  try {
    await ElMessageBox.confirm(
      '该内容正在人工审核中。如需修改后重新发布，请先撤回，修改源内容后再点击发布；不操作则继续等待审核结果。',
      '内容审核中',
      { type: 'info', confirmButtonText: '撤回', cancelButtonText: '继续等待' },
    )
  } catch {
    return
  }
  busy.value = true
  try {
    await squareApi.unpublish(pub.value.id)
    pub.value = null
    ElMessage.success('已撤回，修改后重新点击发布即可')
  } catch { /* 拦截器已提示 */ } finally {
    busy.value = false
  }
}
</script>
