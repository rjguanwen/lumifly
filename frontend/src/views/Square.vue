<template>
  <div class="space-y-6">
    <!-- 标题 -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 flex items-center gap-2">
          <el-icon color="#4a6cf7"><Compass /></el-icon>广场
        </h1>
        <p class="text-gray-500 mt-1">分享你的人生片段，看看大家在记录什么</p>
      </div>
    </div>

    <!-- 工具行：分类 + 搜索 + 排序 -->
    <div class="flex flex-wrap items-center gap-3">
      <div class="flex flex-wrap items-center gap-2">
        <el-button
          size="small"
          :type="typeFilter === '' ? 'primary' : ''"
          round
          @click="setType('')"
        >全部</el-button>
        <el-button
          v-for="t in squareTypes"
          :key="t.value"
          size="small"
          round
          :style="typeFilter === t.value ? { background: t.color, borderColor: t.color, color: '#fff' } : {}"
          @click="setType(t.value)"
        >{{ t.label }}</el-button>
      </div>
      <div class="flex-1" />
      <el-input
        v-model="keyword"
        placeholder="搜索广场内容"
        clearable
        style="width: 200px"
        @keyup.enter="onSearch"
        @clear="onSearch"
      >
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
      <el-select v-model="sortBy" style="width: 110px" @change="onSearch">
        <el-option label="最新发布" value="latest" />
        <el-option label="最热" value="hot" />
      </el-select>
      <el-select
        v-model="authorSel"
        clearable
        filterable
        placeholder="只看某位作者"
        style="width: 150px"
        @change="onAuthorChange"
      >
        <el-option v-for="f in authorOptions" :key="f.id" :label="f.name" :value="f.id" />
      </el-select>
    </div>

    <!-- 当前作者筛选提示 -->
    <div v-if="authorFilter" class="flex items-center gap-2 -mt-2">
      <el-tag type="primary" effect="plain" closable @close="clearAuthorFilter">
        正在查看 <b>{{ authorFilter.name }}</b> 发布的内容
      </el-tag>
    </div>

    <!-- 空态 -->
    <el-empty v-if="loaded && !items.length" description="广场暂时没有内容" />

    <!-- 帖子瀑布流：行优先填充（行内与行间均新→旧） -->
    <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4 items-start">
      <div
        v-for="p in items"
        :key="p.id"
        class="bg-white rounded-2xl border border-gray-100 shadow-sm hover:shadow-md transition-shadow overflow-hidden cursor-pointer"
        @click="openDetail(p)"
      >
        <!-- 头部：作者 + 类型 -->
        <div class="flex items-center gap-2.5 px-5 pt-4">
          <UserAvatar :src="p.authorAvatar" :name="p.authorName" :size="32" />
          <div class="min-w-0">
            <div class="flex items-center gap-1.5">
              <span
                class="text-sm font-medium text-gray-800 truncate cursor-pointer hover:text-indigo-600"
                :class="{ 'text-indigo-600': authorFilter?.id === p.userId }"
                :title="authorFilter?.id === p.userId ? '正在筛选该作者' : '只看 TA 发布的内容'"
                @click.stop="filterByAuthor(p)"
              >{{ p.authorName }}</span>
              <span v-if="p.userId === myId" class="text-[10px] px-1.5 py-0.5 rounded bg-blue-50 text-blue-500">我的</span>
            </div>
            <div class="text-xs text-gray-400">{{ fmtTime(p.createdAt) }}</div>
            <p v-if="p.authorSignature" class="text-xs text-gray-400 italic truncate mt-0.5">{{ p.authorSignature }}</p>
          </div>
          <div class="flex-1" />
          <span v-if="p.isNew" class="flex items-center gap-1 text-[11px] text-rose-500 shrink-0">
            <span class="w-2 h-2 rounded-full bg-rose-500 shadow-[0_0_0_3px_rgba(244,63,94,0.15)]" />新
          </span>
          <span
            class="text-xs px-2 py-0.5 rounded-full font-medium flex items-center gap-1 shrink-0"
            :style="{ background: tMeta(p).bg, color: tMeta(p).color }"
          ><el-icon :size="12"><component :is="tMeta(p).icon" /></el-icon>{{ tMeta(p).label }}</span>
        </div>

        <!-- 内容 -->
        <div class="px-5 pt-3 pb-2">
          <h3 class="text-base font-semibold text-gray-900 leading-snug">{{ p.title }}</h3>

          <!-- 书籍附加信息 -->
          <template v-if="p.sourceType === 'book'">
            <div class="flex flex-wrap items-center gap-2 mt-2">
              <span v-if="p.author" class="text-sm text-gray-500">{{ p.author }}</span>
              <span v-if="p.rating" class="text-amber-400 text-sm">{{ '★'.repeat(p.rating) }}</span>
            </div>
            <div v-if="p.meta?.domain || p.meta?.readYear" class="flex gap-1.5 flex-wrap mt-1.5">
              <span v-if="p.meta.domain" class="text-xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-600">{{ p.meta.domain }}</span>
              <span v-if="p.meta.readYear" class="text-xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-600">{{ p.meta.readYear }}</span>
            </div>
          </template>

          <p v-if="p.preview" class="mt-2 text-sm text-gray-600 leading-relaxed line-clamp-4">{{ p.preview }}</p>
        </div>

        <!-- 底部操作 -->
        <div class="flex items-center gap-1 px-4 py-2 border-t border-gray-50 bg-gray-50/60">
          <button
            type="button"
            class="btn-act"
            :class="p.liked ? 'is-active' : ''"
            @click.stop="toggleLike(p)"
          >{{ p.liked ? '♥' : '♡' }} {{ p.likeCount || 0 }}</button>
          <button type="button" class="btn-act" @click.stop="openDetail(p)">
            <el-icon :size="13"><ChatDotRound /></el-icon> {{ p.commentCount || 0 }}
          </button>
          <div class="flex-1" />
          <span class="text-xs text-blue-500">查看全文 →</span>
        </div>
      </div>
    </div>

    <!-- 滚动加载哨兵：接近底部自动加载下一页 -->
    <div v-if="items.length < total" ref="sentinel" class="py-6 text-center text-sm text-gray-400">
      {{ loading ? '加载中…' : '下拉加载更多' }}
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { friendApi, squareApi } from '../api'
import { useAuthStore } from '../stores/auth'
import UserAvatar from '../components/UserAvatar.vue'
import { squareTypes, squareTypeOf } from '../utils/squareMeta'

const router = useRouter()
const auth = useAuthStore()
const myId = auth.user?.id

const items = ref([])
const total = ref(0)
const page = ref(1)
const loading = ref(false)
const loaded = ref(false)
const sentinel = ref(null)

const typeFilter = ref('')
const keyword = ref('')
const sortBy = ref('latest')

// 作者筛选
const friends = ref([])
const authorSel = ref(null)
const authorFilter = ref(null) // { id, name }
const authorOptions = computed(() => {
  const opts = []
  if (auth.user) opts.push({ id: auth.user.id, name: '我（自己）' })
  for (const f of friends.value) {
    if (f.id !== auth.user?.id) opts.push({ id: f.id, name: f.displayName })
  }
  if (authorFilter.value && !opts.some((o) => o.id === authorFilter.value.id)) {
    opts.push({ id: authorFilter.value.id, name: authorFilter.value.name })
  }
  return opts
})

const tMeta = (p) => squareTypeOf(p.sourceType)

async function loadFriends() {
  try {
    const data = await friendApi.list()
    friends.value = data.items || []
  } catch { /* 忽略 */ }
}

function applyAuthorFilter(user) {
  if (!user) {
    authorFilter.value = null
    authorSel.value = null
  } else {
    authorFilter.value = { id: user.id, name: user.name }
    authorSel.value = user.id
  }
  load()
}

function filterByAuthor(p) {
  applyAuthorFilter({ id: p.userId, name: p.authorName })
}

function onAuthorChange(v) {
  if (!v) {
    authorFilter.value = null
    load()
    return
  }
  const o = authorOptions.value.find((x) => x.id === v)
  if (o) applyAuthorFilter(o)
}

function clearAuthorFilter() {
  authorFilter.value = null
  authorSel.value = null
  load()
}

function fmtTime(s) {
  if (!s) return ''
  const d = new Date(s)
  const now = new Date()
  const sameYear = d.getFullYear() === now.getFullYear()
  const pad = (n) => String(n).padStart(2, '0')
  return sameYear
    ? `${d.getMonth() + 1}月${d.getDate()}日`
    : `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

async function load(reset = true) {
  if (loading.value) return
  loading.value = true
  try {
    const params = { sort: sortBy.value, page: reset ? 1 : page.value + 1, limit: 24 }
    if (typeFilter.value) params.type = typeFilter.value
    if (keyword.value.trim()) params.keyword = keyword.value.trim()
    if (authorFilter.value?.id) params.userId = authorFilter.value.id
    const data = await squareApi.list(params)
    items.value = reset ? data.items || [] : [...items.value, ...(data.items || [])]
    total.value = data.total || items.value.length
    if (!reset) page.value += 1
    loaded.value = true
  } catch { /* 拦截器已提示 */ } finally {
    loading.value = false
  }
}

function setType(v) {
  typeFilter.value = v
  load()
}
function onSearch() {
  load()
}
function loadMore() {
  load(false)
}

async function toggleLike(p) {
  try {
    const r = await squareApi.like(p.id)
    p.liked = r.liked
    p.likeCount = r.likeCount
  } catch { /* 忽略 */ }
}

function openDetail(p) {
  router.push(`/square/${p.id}`)
}

// 滚动加载：当底部哨兵接近视口（剩余不足 320px）时自动加载下一页。
// 滚动容器取自哨兵最近的 overflow 祖先（Element Plus 的 el-main），无则监听 window。
function findScrollParent(el) {
  while (el && el !== document.body) {
    const s = window.getComputedStyle(el)
    if (s.overflowY === 'auto' || s.overflowY === 'scroll') return el
    el = el.parentElement
  }
  return window
}

let scrollEl = null
function onNearBottom() {
  if (loading.value || items.value.length >= total.value) return
  const el = scrollEl === window ? document.documentElement : scrollEl
  if (el.scrollTop + el.clientHeight >= el.scrollHeight - 320) loadMore()
}

watch(
  total,
  async () => {
    await nextTick()
    if (items.value.length >= total.value || !sentinel.value) return
    if (!scrollEl) {
      scrollEl = findScrollParent(sentinel.value)
      scrollEl.addEventListener('scroll', onNearBottom, { passive: true })
    }
    // 列表不足一屏时也自动补页，保证能继续滚动加载
    onNearBottom()
  },
  { flush: 'post' },
)

onBeforeUnmount(() => scrollEl?.removeEventListener('scroll', onNearBottom))

onMounted(() => {
  load()
  loadFriends()
})
</script>

<style scoped>
.btn-act {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: none;
  background: transparent;
  border-radius: 8px;
  font-size: 13px;
  color: #6b7280;
  cursor: pointer;
  transition: background-color 0.12s ease, color 0.12s ease;
}

.btn-act:hover {
  background: #fff;
  color: #d97706;
}

.btn-act.is-active {
  color: #ef4444;
}
</style>
