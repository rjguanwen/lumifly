<template>
  <el-dialog
    :model-value="visible"
    width="820px"
    top="4vh"
    :close-on-click-modal="false"
    append-to-body
    class="square-post-dialog"
    @update:model-value="$emit('update:visible', $event)"
  >
    <div v-if="loading" class="py-16 text-center text-gray-400">加载中…</div>
    <div v-else-if="!post" class="py-16 text-center text-gray-400">内容不存在或已撤回</div>

    <template v-else>
      <div class="flex flex-wrap items-center gap-2.5">
        <UserAvatar :src="post.authorAvatar" :name="post.authorName" :size="40" />
        <div class="min-w-0">
          <div class="flex items-center gap-1.5">
            <span class="text-sm font-medium text-gray-800">{{ post.authorName }}</span>
            <span v-if="post.userId === myId" class="text-[10px] px-1.5 py-0.5 rounded bg-blue-50 text-blue-500">我的</span>
          </div>
          <div class="text-xs text-gray-400">{{ fmtTime(post.createdAt) }}</div>
        </div>
        <div class="flex-1" />
        <span
          class="text-xs px-2 py-0.5 rounded-full font-medium"
          :style="{ background: tMeta(post).bg, color: tMeta(post).color }"
        >{{ tMeta(post).label }}</span>
        <el-button v-if="post.userId === myId" size="small" text type="danger" @click="unpublish">
          撤回
        </el-button>
      </div>

      <div v-if="post.authorSignature" class="text-sm text-gray-500 italic mt-2">{{ post.authorSignature }}</div>

      <!-- 书籍附加信息 -->
      <template v-if="post.sourceType === 'book'">
        <div class="mt-3 rounded-xl bg-gray-50 px-4 py-3 flex flex-wrap gap-x-4 gap-y-1 text-sm text-gray-600">
          <span>作者：<b class="text-gray-800">{{ post.author || '—' }}</b></span>
          <span v-if="post.meta?.domain">领域：{{ post.meta.domain }}</span>
          <span v-if="post.meta?.readYear">阅读于 {{ post.meta.readYear }} 年</span>
          <span class="text-amber-400">{{ post.rating ? '★'.repeat(post.rating) : '' }}</span>
        </div>
      </template>

      <h1 v-if="post.title" class="text-xl font-bold text-gray-900 leading-snug mt-4">{{ post.title }}</h1>
      <div class="square-post-content prose prose-sm max-w-none mt-2 max-h-[34vh] overflow-y-auto" v-html="post.content" />

      <!-- 操作与评论 -->
      <div class="mt-4 pt-3 border-t border-gray-100">
        <button
          type="button"
          class="inline-flex items-center gap-1.5 text-sm transition-colors"
          :class="post.liked ? 'text-rose-500' : 'text-gray-500 hover:text-rose-500'"
          @click="toggleLike"
        >{{ post.liked ? '♥' : '♡' }} {{ post.likeCount || 0 }}</button>
        <span class="text-sm text-gray-400 ml-4">评论 {{ comments.length }}</span>
      </div>

      <div class="flex gap-3 mt-3">
        <UserAvatar :src="auth.user?.avatarUrl" :name="auth.user?.displayName" :size="30" />
        <el-input
          v-model="commentText"
          type="textarea"
          :rows="2"
          maxlength="1000"
          placeholder="友善评论…"
          @keydown.ctrl.enter="sendComment"
        />
        <el-button type="primary" :loading="sending" class="self-end" @click="sendComment">发布</el-button>
      </div>

      <div v-if="comments.length" class="mt-3 space-y-3 max-h-[26vh] overflow-y-auto">
        <div v-for="cm in comments" :key="cm.id" class="flex gap-2.5">
          <UserAvatar :src="cm.authorAvatar" :name="cm.authorName" :size="30" class="shrink-0" />
          <div class="min-w-0 flex-1 bg-gray-50 rounded-xl px-3 py-2">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-gray-800">{{ cm.authorName }}</span>
              <span class="text-xs text-gray-400">{{ fmtTime(cm.createdAt) }}</span>
              <div class="flex-1" />
              <button v-if="cm.canDelete" type="button" class="text-xs text-gray-400 hover:text-red-500" @click="removeComment(cm)">删除</button>
            </div>
            <p class="mt-1 text-sm text-gray-700 whitespace-pre-wrap">{{ cm.content }}</p>
          </div>
        </div>
      </div>
      <div v-else class="mt-3 text-center text-sm text-gray-300 pb-1">还没有评论</div>
    </template>
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import UserAvatar from './UserAvatar.vue'
import { useAuthStore } from '../stores/auth'
import { squareApi } from '../api'
import { squareTypeOf } from '../utils/squareMeta'

const props = defineProps({
  visible: Boolean,
  postId: { type: [Number, String], default: 0 },
})
const emit = defineEmits(['update:visible', 'changed'])

const auth = useAuthStore()
const myId = auth.user?.id
const tMeta = (p) => squareTypeOf(p.sourceType)

const loading = ref(false)
const post = ref(null)
const comments = ref([])
const commentText = ref('')
const sending = ref(false)

function fmtTime(s) {
  if (!s) return ''
  const d = new Date(s)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

async function load() {
  if (!props.visible || !props.postId) return
  loading.value = true
  try {
    post.value = await squareApi.detail(props.postId)
    const data = await squareApi.comments(props.postId)
    comments.value = (data.items || []).map((cm) => ({ ...cm, canDelete: cm.authorUserId === myId }))
  } catch {
    post.value = null
  } finally {
    loading.value = false
  }
}

async function toggleLike() {
  try {
    await squareApi.like(post.value.id)
    post.value = await squareApi.detail(post.value.id)
  } catch { /* 拦截器已提示 */ }
}

async function sendComment() {
  const text = commentText.value.trim()
  if (!text) {
    ElMessage.warning('请输入评论内容')
    return
  }
  sending.value = true
  try {
    const item = await squareApi.addComment(post.value.id, text)
    comments.value.push({ ...item, canDelete: true })
    commentText.value = ''
  } catch { /* 拦截器已提示 */ } finally {
    sending.value = false
  }
}

async function removeComment(cm) {
  try {
    await ElMessageBox.confirm('删除这条评论？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await squareApi.deleteComment(cm.id)
    comments.value = comments.value.filter((x) => x.id !== cm.id)
  } catch { /* 拦截器已提示 */ }
}

async function unpublish() {
  try {
    await ElMessageBox.confirm('撤回该内容？他人点赞与评论将一并清除。', '撤回发布', { type: 'warning' })
  } catch {
    return
  }
  try {
    await squareApi.unpublish(post.value.id)
    ElMessage.success('已撤回')
    emit('changed')
    emit('update:visible', false)
  } catch { /* 拦截器已提示 */ }
}

watch(() => props.visible, (v) => {
  if (v) {
    commentText.value = ''
    load()
  } else {
    post.value = null
    comments.value = []
  }
})
</script>

<style scoped>
.square-post-content :deep(img) {
  max-width: 100%;
  border-radius: 12px;
  margin: 10px 0;
}

.square-post-content :deep(video) {
  max-width: 100%;
  border-radius: 12px;
  margin: 10px 0;
}
</style>
