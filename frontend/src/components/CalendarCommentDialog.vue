<template>
  <el-dialog
    :model-value="visible"
    title="私密评价"
    width="620px"
    :close-on-click-modal="false"
    append-to-body
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="title" class="text-sm text-gray-500 mb-4 -mt-2 truncate">{{ title }}</div>

    <div v-if="loading" class="text-center py-8 text-gray-400 text-sm">加载中…</div>
    <div v-else-if="comments.length" class="space-y-3 max-h-[38vh] overflow-y-auto pr-1">
      <div v-for="cm in comments" :key="cm.id" class="flex gap-3">
        <UserAvatar :src="cm.authorAvatar" :name="cm.authorName" :size="32" />
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2">
            <span class="text-xs font-medium text-gray-700">{{ cm.authorName }}</span>
            <span v-if="cm.authorUserId === ownerId" class="text-[10px] px-1 rounded bg-indigo-50 text-indigo-500">主人</span>
            <span class="text-xs text-gray-300 flex-1">{{ fmtTime(cm.createdAt) }}</span>
            <el-button v-if="cm.authorUserId === myId" size="small" text type="danger" @click="remove(cm.id)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
          <div class="text-sm text-gray-700 whitespace-pre-line leading-relaxed mt-0.5">{{ cm.content }}</div>
        </div>
      </div>
    </div>
    <div v-else class="text-center py-8 text-gray-300 text-sm">还没有评价，写下你的想法吧</div>

    <div class="flex gap-3 mt-4 pt-4 border-t border-gray-100">
      <UserAvatar :src="auth.user?.avatarUrl" :name="auth.user?.displayName" :size="32" />
      <div class="flex-1">
        <el-input
          v-model="content"
          type="textarea"
          :rows="3"
          maxlength="500"
          show-word-limit
          placeholder="写一条双方可见的评价..."
        />
      </div>
    </div>
    <template #footer>
      <el-button @click="$emit('update:visible', false)">关闭</el-button>
      <el-button type="primary" :loading="posting" :disabled="!content.trim()" @click="submit">发表评价</el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import UserAvatar from './UserAvatar.vue'
import { useAuthStore } from '../stores/auth'
import { calendarCommentApi } from '../api'

const props = defineProps({
  visible: Boolean,
  ownerId: { type: Number, required: true },
  targetType: { type: String, required: true }, // record / milestone
  targetId: { type: Number, required: true },
  title: { type: String, default: '' },
})
const emit = defineEmits(['update:visible'])

const auth = useAuthStore()
const myId = auth.user?.id

const comments = ref([])
const loading = ref(false)
const posting = ref(false)
const content = ref('')

function fmtTime(s) {
  return s ? s.slice(0, 16).replace('T', ' ') : ''
}

async function load() {
  if (!props.visible) return
  loading.value = true
  try {
    const data = await calendarCommentApi.list(props.ownerId, props.targetType, props.targetId)
    comments.value = data.items || []
  } catch {
    /* 拦截器已提示 */
  } finally {
    loading.value = false
  }
}

async function submit() {
  const c = content.value.trim()
  if (!c) return
  posting.value = true
  try {
    await calendarCommentApi.post({ ownerId: props.ownerId, targetType: props.targetType, targetId: props.targetId, content: c })
    content.value = ''
    ElMessage.success('评价已发表')
    load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    posting.value = false
  }
}

async function remove(id) {
  try {
    await calendarCommentApi.remove(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* 拦截器已提示 */
  }
}

watch(() => props.visible, (v) => {
  if (v) {
    content.value = ''
    load()
  }
})
</script>
