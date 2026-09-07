<template>
  <div class="space-y-6">
    <!-- 标题 -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900 flex items-center gap-2">
          <el-icon color="#0e9f6e"><UserFilled /></el-icon>好友
        </h1>
        <p class="text-gray-500 mt-1">添加好友，随时在广场专注查看彼此的生活</p>
      </div>
    </div>

    <!-- 添加好友 -->
    <div class="bg-white rounded-2xl border border-gray-100 shadow-sm p-5">
      <div class="flex items-center gap-3">
        <el-input
          v-model="searchKeyword"
          placeholder="输入好友昵称搜索（如：老关）"
          clearable
          style="max-width: 320px"
          @keyup.enter="doSearch"
        >
          <template #prefix><el-icon><Search /></el-icon></template>
        </el-input>
        <el-button type="primary" :loading="searching" @click="doSearch">搜索用户</el-button>
      </div>

      <!-- 搜索结果 -->
      <div v-if="searchResults.length" class="mt-4 divide-y divide-gray-50 border-t border-gray-100">
        <div v-for="u in searchResults" :key="u.id" class="flex items-center gap-3 py-3">
          <UserAvatar :src="u.avatarUrl" :name="u.displayName" :size="40" />
          <div class="min-w-0 flex-1">
            <div class="text-sm font-medium text-gray-800">{{ u.displayName }}</div>
            <div v-if="u.signature" class="text-xs text-gray-400 truncate">{{ u.signature }}</div>
          </div>
          <el-button
            v-if="u.relation === 'none'"
            size="small"
            type="primary"
            plain
            @click="sendRequest(u)"
          >加好友</el-button>
          <el-tag v-else-if="u.relation === 'requested'" size="small" type="info" effect="plain">已发送申请</el-tag>
          <template v-else-if="u.relation === 'requested_me'">
            <el-tag size="small" type="warning" effect="plain">对方已申请</el-tag>
            <el-button size="small" type="success" plain @click="acceptFromSearch(u)">接受</el-button>
          </template>
          <el-tag v-else size="small" type="success" effect="plain">已是好友</el-tag>
        </div>
      </div>
      <p v-if="searched && !searchResults.length" class="text-sm text-gray-400 mt-3">没有找到匹配的用户</p>
    </div>

    <!-- 收到的好友申请 -->
    <div v-if="received.length">
      <h2 class="text-base font-semibold text-gray-700 mb-3 flex items-center gap-1.5">
        新的好友申请
        <span class="text-xs px-1.5 py-0.5 rounded-full bg-amber-50 text-amber-600">{{ received.length }}</span>
      </h2>
      <div class="space-y-2.5">
        <div v-for="r in received" :key="r.requestId" class="flex items-center gap-3 bg-white rounded-2xl border border-gray-100 shadow-sm p-4">
          <UserAvatar :src="r.avatarUrl" :name="r.displayName" :size="44" />
          <div class="min-w-0 flex-1">
            <div class="text-sm font-semibold text-gray-800">{{ r.displayName }}</div>
            <div v-if="r.signature" class="text-xs text-gray-400 truncate">{{ r.signature }}</div>
            <div v-else class="text-xs text-gray-300">{{ fmtTime(r.createdAt) }}</div>
          </div>
          <el-button size="small" type="danger" plain @click="reject(r.requestId)">忽略</el-button>
          <el-button size="small" type="primary" @click="accept(r.requestId)">接受</el-button>
        </div>
      </div>
    </div>

    <!-- 我的好友 -->
    <div>
      <h2 class="text-base font-semibold text-gray-700 mb-3">我的好友（{{ friends.length }}）</h2>
      <div v-if="friends.length" class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
        <div
          v-for="f in friends"
          :key="f.id"
          class="flex items-center gap-3 bg-white rounded-2xl border border-gray-100 shadow-sm hover:shadow-md transition-shadow p-4"
        >
          <UserAvatar :src="f.avatarUrl" :name="f.displayName" :size="48" />
          <div class="min-w-0 flex-1">
            <div class="text-sm font-semibold text-gray-800 truncate">{{ f.displayName }}</div>
            <div v-if="f.signature" class="text-xs text-gray-400 truncate">{{ f.signature }}</div>
            <div class="text-[11px] text-gray-300 mt-0.5">{{ f.friendSince ? '成为好友于 ' + fmtTime(f.friendSince) : '' }}</div>
          </div>
          <el-button size="small" text type="danger" @click="removeFriend(f)">解除</el-button>
        </div>
      </div>
      <el-empty v-else-if="!received.length" description="还没有好友，先搜索并添加一位吧" />
    </div>

    <!-- 我发出的申请 -->
    <div v-if="sent.length">
      <h2 class="text-base font-semibold text-gray-700 mb-3">等待对方处理</h2>
      <div class="flex flex-wrap gap-2.5">
        <div v-for="s in sent" :key="s.requestId" class="flex items-center gap-2.5 bg-white rounded-xl border border-gray-100 px-3 py-2">
          <UserAvatar :src="s.avatarUrl" :name="s.displayName" :size="32" />
          <span class="text-sm text-gray-700">{{ s.displayName }}</span>
          <el-button size="small" text @click="cancelSent(s.requestId)">撤回</el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import UserAvatar from '../components/UserAvatar.vue'
import { friendApi } from '../api'

const searchKeyword = ref('')
const searching = ref(false)
const searched = ref(false)
const searchResults = ref([])
const friends = ref([])
const received = ref([])
const sent = ref([])

function fmtTime(s) {
  if (!s) return ''
  return s.slice(0, 10)
}

async function loadAll() {
  try {
    const [f, r] = await Promise.all([friendApi.list(), friendApi.requests()])
    friends.value = f.items || []
    received.value = r.received || []
    sent.value = r.sent || []
  } catch { /* 拦截器已提示 */ }
}

async function doSearch() {
  const kw = searchKeyword.value.trim()
  if (!kw) {
    ElMessage.warning('请输入要搜索的昵称')
    return
  }
  searching.value = true
  searched.value = true
  try {
    const data = await friendApi.searchUsers(kw)
    searchResults.value = data.items || []
  } catch { /* 拦截器已提示 */ } finally {
    searching.value = false
  }
}

async function sendRequest(u) {
  try {
    const r = await friendApi.sendRequest(u.id)
    ElMessage.success(r.accepted ? '你们已经成为好友了！' : '好友申请已发送')
    searchResults.value = searchResults.value.map((x) =>
      x.id === u.id ? { ...x, relation: r.accepted ? 'friend' : 'requested' } : x,
    )
    loadAll()
  } catch { /* 拦截器已提示 */ }
}

async function acceptFromSearch(u) {
  try {
    // 对方已申请我，直接发起申请即可互相成为好友
    await friendApi.sendRequest(u.id)
    ElMessage.success('你们已经成为好友了！')
    loadAll()
    doSearch()
  } catch { /* 拦截器已提示 */ }
}

async function accept(requestId) {
  try {
    await friendApi.acceptRequest(requestId)
    ElMessage.success('已接受好友申请')
    loadAll()
  } catch { /* 拦截器已提示 */ }
}

async function reject(requestId) {
  try {
    await friendApi.cancelOrReject(requestId)
    ElMessage.success('已忽略该申请')
    loadAll()
  } catch { /* 拦截器已提示 */ }
}

async function cancelSent(requestId) {
  try {
    await friendApi.cancelOrReject(requestId)
    ElMessage.success('已撤回申请')
    loadAll()
  } catch { /* 拦截器已提示 */ }
}

async function removeFriend(f) {
  try {
    await ElMessageBox.confirm(`确定与「${f.displayName}」解除好友关系？`, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await friendApi.remove(f.id)
    ElMessage.success('已解除好友关系')
    loadAll()
  } catch { /* 拦截器已提示 */ }
}

onMounted(loadAll)
</script>
