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

    <!-- 帖子流：行序网格。整个页面新帖在上方行、旧帖在下方行；同一行内左→右从新到旧。
         列表数据（items）按发布时间新→旧排列，grid 默认按行填充，因此天然满足上述行序。
         严禁改回 columns 瀑布流（CSS columns 按列填充会打乱横向新旧次序）。 -->
    <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 items-start">
      <div
        v-for="p in items"
        :key="p.id"
        class="group bg-white rounded-xl border border-gray-100 shadow-sm hover:shadow-lg transition-shadow overflow-hidden cursor-pointer"
        @click="openDetail(p)"
      >
        <!-- 有图卡片：缩小版封面（约为原高 2/3）+ 角标 -->
        <template v-if="postImgs(p).length">
          <div class="relative overflow-hidden">
            <img
              :src="postImgs(p)[0]"
              loading="lazy"
              class="w-full aspect-[5/4] object-cover transition-transform duration-300 group-hover:scale-[1.03]"
              alt=""
            />
            <span v-if="p.isNew" class="absolute left-2 top-2 px-1.5 py-0.5 rounded-md bg-rose-500 text-white text-[11px]">新</span>
            <span
              class="absolute right-2 top-2 px-1.5 py-0.5 rounded-md text-[11px] font-medium backdrop-blur"
              :style="{ background: tMeta(p).color, color: '#fff' }"
            >{{ tMeta(p).label }}</span>
            <template v-if="p.sourceType === 'book'">
              <span v-if="p.author" class="absolute left-2 bottom-2 text-[11px] text-white drop-shadow px-1.5 py-0.5 rounded bg-black/35 backdrop-blur">{{ p.author }}</span>
              <span v-if="p.rating" class="absolute right-2 bottom-2 text-[12px] text-amber-300 drop-shadow">{{ '★'.repeat(Math.min(p.rating, 5)) }}</span>
            </template>
          </div>
        </template>

        <!-- 纯文字卡片：紧凑头部条（类型 + 新标记），正文撑内容 -->
        <div v-else class="px-3 pt-2.5 flex items-center gap-2">
          <el-icon :size="15" :color="tMeta(p).color"><component :is="tMeta(p).icon" /></el-icon>
          <span
            class="text-[11px] px-1.5 py-0.5 rounded"
            :style="{ background: tMeta(p).bg, color: tMeta(p).color }"
          >{{ tMeta(p).label }}</span>
          <span v-if="p.isNew" class="ml-auto flex items-center gap-1 text-[10px] text-rose-500">
            <i class="w-1.5 h-1.5 rounded-full bg-rose-500 inline-block" />新
          </span>
        </div>

        <!-- 文案区 -->
        <div class="px-3 py-2.5">
          <h3 class="text-[15px] font-semibold text-gray-900 leading-snug line-clamp-2">{{ p.title || p.preview }}</h3>
          <p v-if="!postImgs(p).length && p.preview" class="mt-1 text-[13px] text-gray-500 leading-relaxed line-clamp-3">{{ p.preview }}</p>
          <div class="mt-2 flex items-center gap-1.5">
            <UserAvatar :src="p.authorAvatar" :name="p.authorName" :size="18" />
            <span
              class="text-xs text-gray-500 truncate cursor-pointer hover:text-indigo-600"
              :class="{ 'text-indigo-600': authorFilter?.id === p.userId }"
              :title="authorFilter?.id === p.userId ? '正在筛选该作者' : '只看 TA 发布的内容'"
              @click.stop="filterByAuthor(p)"
            >{{ p.authorName }}</span>
            <span v-if="p.userId === myId" class="text-[10px] px-1 rounded bg-blue-50 text-blue-500">我的</span>
            <span class="flex-1" />
            <span class="text-xs text-gray-400">{{ fmtTime(p.createdAt) }}</span>
          </div>
          <div class="mt-0.5 flex items-center text-xs text-gray-400">
            <button
              type="button"
              class="btn-act like-btn"
              :class="{ 'is-liked': p.liked }"
              @click.stop="toggleLike(p)"
            ><i class="heart" aria-hidden="true">{{ p.liked ? '♥' : '♡' }}</i> {{ p.likeCount || 0 }}</button>
            <button type="button" class="btn-act comment-btn" @click.stop="openDetail(p)">
              <el-icon :size="14"><ChatDotRound /></el-icon> {{ p.commentCount || 0 }}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- 滚动加载哨兵：接近底部自动加载下一页 -->
    <div v-if="items.length < total" ref="sentinel" class="py-6 text-center text-sm text-gray-400">
      {{ loading ? '加载中…' : '下拉加载更多' }}
    </div>

    <!-- 帖子内容弹窗（支持 ESC 关闭） -->
    <SquarePostDialog
      v-model:visible="postDialogVisible"
      :post-id="postDialogId"
      @changed="load(true)"
    />
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { friendApi, squareApi } from '../api'
import { useAuthStore } from '../stores/auth'
import UserAvatar from '../components/UserAvatar.vue'
import SquarePostDialog from '../components/SquarePostDialog.vue'
import { squareTypes, squareTypeOf } from '../utils/squareMeta'

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

// 帖子弹窗状态
const postDialogVisible = ref(false)
const postDialogId = ref(0)
function openDetail(p) {
  postDialogId.value = p.id
  postDialogVisible.value = true
}

// 从快照正文提取图片，用于卡片缩略（最多前 3 张，带缓存）
const imgCache = {}
function extractContentImgs(html) {
  if (!html) return []
  const out = []
  const re = /<img[^>]+src="([^"]+)"/g
  let m
  while ((m = re.exec(html)) && out.length < 3) out.push(m[1])
  return out
}
function postImgs(p) {
  if (!(p.id in imgCache)) imgCache[p.id] = extractContentImgs(p.content)
  return imgCache[p.id]
}

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
/* 点赞 / 评论小按钮：无边框浅灰文字，悬浮时浅色圆底 + 变色 */
.btn-act {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 9px;
  margin: -4px -4px;
  border: 0;
  background: transparent;
  border-radius: 999px;
  font-size: 12px;
  line-height: 1;
  color: #9ca3af;
  cursor: pointer;
  user-select: none;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.btn-act:hover {
  background-color: #f3f4f6;
}

.btn-act:active {
  transform: scale(0.92);
}

.like-btn .heart {
  font-style: normal;
  font-size: 13px;
  transition: transform 0.15s ease;
}

.like-btn:hover {
  color: #f43f5e;
}

.like-btn.is-liked {
  color: #f43f5e;
}

.like-btn.is-liked .heart {
  transform: scale(1.08);
}

.comment-btn:hover {
  color: #4b5563;
}
</style>
