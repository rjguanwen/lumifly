<template>
  <div class="max-w-3xl mx-auto">
    <div v-if="post" class="bg-white rounded-2xl shadow-sm overflow-hidden border border-gray-100">
      <!-- 工具条 -->
      <div class="px-5 sm:px-8 pt-4 pb-3 flex items-center justify-between flex-wrap gap-2 border-b border-gray-100 bg-gray-50/50">
        <el-button text type="primary" @click="$router.push('/square')">
          <el-icon class="mr-1"><ArrowLeft /></el-icon>返回广场
        </el-button>
        <div class="flex items-center gap-2">
          <el-button v-if="isMine" size="small" type="danger" plain :loading="unpublishing" @click="unpublish">
            <el-icon class="mr-1"><Delete /></el-icon>撤回
          </el-button>
          <el-button
            size="small"
            :type="post.liked ? 'danger' : 'primary'"
            :plain="!post.liked"
            @click="toggleLike"
          >{{ post.liked ? '♥ 已赞' : '♡ 点赞' }} {{ post.likeCount || '' }}</el-button>
        </div>
      </div>

      <article class="px-5 sm:px-10 py-7">
        <!-- 元信息 -->
        <div class="flex flex-wrap items-center gap-2.5 mb-4">
          <UserAvatar :src="post.authorAvatar" :name="post.authorName" :size="36" />
          <div class="min-w-0">
            <div class="text-sm font-medium text-gray-800 flex items-center gap-2">
              {{ post.authorName }}
              <span v-if="isMine" class="text-[10px] px-1.5 py-0.5 rounded bg-blue-50 text-blue-500">我的</span>
            </div>
            <div class="text-xs text-gray-400">{{ fmtTime(post.createdAt) }}</div>
            <p v-if="post.authorSignature" class="text-sm text-gray-500 mt-1 truncate max-w-[60vw]">{{ post.authorSignature }}</p>
          </div>
          <div class="flex-1" />
          <span
            class="text-xs px-2.5 py-1 rounded-full font-medium flex items-center gap-1"
            :style="{ background: tMeta.bg, color: tMeta.color }"
          ><el-icon :size="12"><component :is="tMeta.icon" /></el-icon>{{ tMeta.label }}</span>
        </div>

        <!-- 书籍附加信息 -->
        <template v-if="post.sourceType === 'book'">
          <div class="rounded-xl bg-gray-50 px-4 py-3 mb-4 flex flex-wrap items-center gap-x-4 gap-y-1.5">
            <span class="text-sm text-gray-600">作者：<b class="text-gray-800">{{ post.author || '—' }}</b></span>
            <span v-if="post.meta?.domain" class="text-sm text-gray-600">领域：<b class="text-gray-800">{{ post.meta.domain }}</b></span>
            <span v-if="post.meta?.readYear" class="text-sm text-gray-600">阅读于：{{ post.meta.readYear }} 年</span>
            <span class="text-amber-400 text-base" :title="post.rating + '星'">{{ post.rating ? '★'.repeat(post.rating) : '未评分' }}</span>
          </div>
        </template>

        <h1 v-if="post.title" class="text-2xl sm:text-3xl font-bold text-gray-900 leading-snug mb-5">{{ post.title }}</h1>

        <!-- 正文 -->
        <div v-if="post.content" class="prose" v-html="post.content" />
        <div v-else class="py-8 text-center text-gray-300">（该内容没有正文）</div>
      </article>
    </div>

    <div v-else class="py-20 text-center text-gray-400">
      <el-empty :description="loadFailed ? '内容不存在或已撤回' : '加载中...'" />
      <el-button v-if="loadFailed" class="mt-2" @click="$router.push('/square')">返回广场</el-button>
    </div>

    <!-- 评论区 -->
    <div v-if="post" class="mt-5 bg-white rounded-2xl shadow-sm border border-gray-100 overflow-hidden">
      <div class="px-5 sm:px-7 py-4 border-b border-gray-100 flex items-center justify-between">
        <span class="font-semibold text-gray-800 flex items-center gap-1.5">
          <el-icon color="#4a6cf7"><ChatDotRound /></el-icon>评论（{{ comments.length }}）
        </span>
      </div>

      <div class="px-5 sm:px-7 py-4 flex gap-3">
        <UserAvatar :src="auth.user?.avatarUrl" :name="auth.user?.displayName" :size="32" />
        <el-input
          v-model="commentText"
          type="textarea"
          :rows="2"
          maxlength="1000"
          placeholder="友善评论，理性发言…"
          @keydown.ctrl.enter="sendComment"
        />
        <el-button type="primary" :loading="sending" class="self-end" @click="sendComment">发布</el-button>
      </div>

      <div v-if="comments.length" class="px-5 sm:px-7 pb-5 space-y-4">
        <div v-for="cm in comments" :key="cm.id" class="flex gap-3">
          <UserAvatar :src="cm.authorAvatar" :name="cm.authorName" :size="32" />
          <div class="flex-1 min-w-0 bg-gray-50 rounded-xl px-4 py-2.5">
            <div class="flex items-center gap-2">
              <span class="text-sm font-medium text-gray-800">{{ cm.authorName }}</span>
              <span class="text-xs text-gray-400">{{ fmtTime(cm.createdAt) }}</span>
              <div class="flex-1" />
              <button v-if="cm.canDelete" type="button" class="text-xs text-gray-400 hover:text-red-500" @click="removeComment(cm)">删除</button>
            </div>
            <p class="mt-1 text-sm text-gray-700 leading-relaxed whitespace-pre-wrap">{{ cm.content }}</p>
          </div>
        </div>
      </div>
      <div v-else class="px-7 pb-6 text-center text-sm text-gray-300">还没有评论，来抢沙发～</div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { squareApi } from '../api'
import { useAuthStore } from '../stores/auth'
import UserAvatar from '../components/UserAvatar.vue'
import { squareTypeOf } from '../utils/squareMeta'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const post = ref(null)
const loadFailed = ref(false)
const comments = ref([])
const commentText = ref('')
const sending = ref(false)
const unpublishing = ref(false)

const tMeta = computed(() => squareTypeOf(post.value?.sourceType))
const isMine = computed(() => post.value && auth.user && post.value.userId === auth.user.id)

function fmtTime(s) {
  if (!s) return ''
  const d = new Date(s)
  const pad = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

async function load() {
  try {
    post.value = await squareApi.detail(route.params.id)
    const data = await squareApi.comments(route.params.id)
    comments.value = data.items || []
  } catch {
    loadFailed.value = true
  }
}
onMounted(load)

async function toggleLike() {
  try {
    const r = await squareApi.like(post.value.id)
    post.value.liked = r.liked
    post.value.likeCount = r.likeCount
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
  unpublishing.value = true
  try {
    await squareApi.unpublish(post.value.id)
    ElMessage.success('已撤回')
    router.push('/square')
  } catch { /* 拦截器已提示 */ } finally {
    unpublishing.value = false
  }
}
</script>
