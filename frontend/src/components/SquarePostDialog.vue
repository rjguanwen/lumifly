<template>
  <el-dialog
    :model-value="visible"
    width="920px"
    top="3vh"
    :close-on-click-modal="false"
    append-to-body
    class="square-report-dialog"
    @update:model-value="$emit('update:visible', $event)"
  >
    <!-- 报告式头部 -->
    <template #header>
      <div class="report-head">
        <div>
          <div class="report-eyebrow">广场内容 · {{ post ? tMeta(post).label : '' }}</div>
          <div class="report-title">{{ post?.title || (post?.sourceType === 'record' ? '一篇随记' : '分享内容') }}</div>
        </div>
      </div>
    </template>

    <div v-if="loading" class="py-16 text-center text-gray-400">加载中…</div>
    <div v-else-if="!post" class="py-16 text-center text-gray-400">内容不存在或已撤回</div>

    <div v-else class="report-paper">
      <!-- 顶部：作者信息 -->
      <div class="paper-author">
        <UserAvatar :src="post.authorAvatar" :name="post.authorName" :size="44" />
        <div class="min-w-0">
          <div class="flex items-center gap-1.5">
            <span class="font-semibold text-gray-800">{{ post.authorName }}</span>
            <span v-if="post.userId === myId" class="text-[10px] px-1.5 py-0.5 rounded bg-blue-50 text-blue-500">我的</span>
            <span class="text-[11px] text-gray-400 ml-1">{{ fmtTime(post.createdAt) }}</span>
          </div>
          <div v-if="post.authorSignature" class="text-xs text-gray-400 mt-0.5 truncate">{{ post.authorSignature }}</div>
        </div>
        <div class="flex-1" />
        <el-button v-if="post.userId === myId" size="small" text type="danger" @click="unpublish">撤回</el-button>
      </div>

      <!-- 书籍附加信息 -->
      <template v-if="post.sourceType === 'book'">
        <div class="paper-kv">
          <span>作者：<b class="text-gray-800">{{ post.author || '—' }}</b></span>
          <span v-if="post.meta?.domain">领域：{{ post.meta.domain }}</span>
          <span v-if="post.meta?.readYear">阅读于 {{ post.meta.readYear }} 年</span>
          <span v-if="post.rating" class="text-amber-400">{{ '★'.repeat(Math.min(post.rating, 5)) }}</span>
        </div>
      </template>

      <!-- 正文（报告主体） -->
      <div class="paper-body">
        <div class="square-post-content prose prose-sm max-w-none" v-html="post.content" />
      </div>

      <!-- 互动区 -->
      <div class="paper-interact">
        <div class="flex items-center gap-4">
          <button
            type="button"
            class="inline-flex items-center gap-1.5 text-sm font-medium transition-colors"
            :class="post.liked ? 'text-rose-500' : 'text-gray-500 hover:text-rose-500'"
            @click="toggleLike"
          ><span class="text-base">{{ post.liked ? '♥' : '♡' }}</span> {{ post.likeCount || 0 }} 赞</button>
          <span class="text-sm text-gray-400">{{ comments.length }} 条评论</span>
        </div>

        <div class="flex gap-2.5 mt-3">
          <UserAvatar :src="auth.user?.avatarUrl" :name="auth.user?.displayName" :size="30" class="shrink-0" />
          <el-input
            v-model="commentText"
            type="textarea"
            :rows="2"
            maxlength="1000"
            placeholder="写下你的评论，Ctrl + Enter 快捷发送"
            @keydown.ctrl.enter="sendComment"
          />
          <el-button type="primary" :loading="sending" class="self-end" @click="sendComment">发布</el-button>
        </div>

        <div v-if="comments.length" class="mt-4 space-y-3 max-h-[24vh] overflow-y-auto">
          <div v-for="cm in comments" :key="cm.id" class="flex gap-2.5">
            <UserAvatar :src="cm.authorAvatar" :name="cm.authorName" :size="30" class="shrink-0" />
            <div class="min-w-0 flex-1 rounded-xl bg-gray-50 px-3 py-2">
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
        <div v-else class="mt-4 text-center text-sm text-gray-300 pb-1">还没有评论，来说两句吧</div>
      </div>
    </div>
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
.square-report-dialog :deep(.el-dialog) {
  border-radius: 18px;
  overflow: hidden;
  box-shadow: 0 24px 70px rgba(15, 23, 42, 0.22);
}

/* 报告式头部 */
.report-head {
  display: flex;
  align-items: center;
}

.report-eyebrow {
  font-size: 11px;
  color: #909399;
  letter-spacing: 0.5px;
  margin-bottom: 2px;
}

.report-title {
  font-size: 18px;
  font-weight: 700;
  color: #1f2937;
  max-width: 640px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 弹窗内容画布：浅色底 + 白色报告卡 */
.report-paper {
  background: linear-gradient(180deg, #f5f7fa 0%, #f9fafb 100%);
  border-radius: 12px;
  padding: 18px 20px;
  max-height: 74vh;
  overflow-y: auto;
}

.paper-author {
  display: flex;
  align-items: center;
  gap: 12px;
  background: #fff;
  border-radius: 12px;
  padding: 12px 16px;
  box-shadow: 0 1px 3px rgba(16, 24, 40, 0.06);
}

.paper-kv {
  margin-top: 12px;
  background: #fff;
  border-radius: 12px;
  padding: 10px 16px;
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  font-size: 13px;
  color: #6b7280;
  box-shadow: 0 1px 3px rgba(16, 24, 40, 0.06);
}

.paper-body {
  margin-top: 14px;
  background: #fff;
  border-radius: 12px;
  padding: 20px 22px;
  box-shadow: 0 1px 3px rgba(16, 24, 40, 0.06);
}

.paper-interact {
  margin-top: 14px;
  background: #fff;
  border-radius: 12px;
  padding: 16px 18px;
  box-shadow: 0 1px 3px rgba(16, 24, 40, 0.06);
}

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
