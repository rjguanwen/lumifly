<template>
  <div class="space-y-6">
    <!-- 列表模式 -->
    <template v-if="!ownerId">
      <div class="flex items-center justify-between">
        <div>
          <h1 class="text-2xl font-bold text-gray-900">好友日历</h1>
          <p class="text-gray-500 mt-1">好友与你分享的人生日历，可以查看其中的大事记与日常记录</p>
        </div>
      </div>

      <div v-if="items.length" class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
        <div
          v-for="it in items"
          :key="it.id"
          class="flex items-center gap-3 bg-white rounded-2xl border border-gray-100 shadow-sm hover:shadow-md transition-shadow p-4"
        >
          <UserAvatar :src="it.avatarUrl" :name="it.displayName" :size="48" />
          <div class="flex-1 min-w-0">
            <div class="text-sm font-semibold text-gray-800 truncate">{{ it.displayName }}</div>
            <div v-if="it.signature" class="text-xs text-gray-400 truncate">{{ it.signature }}</div>
            <div class="text-[11px] text-gray-300 mt-0.5">{{ it.sharedAt ? '分享于 ' + fmt(it.sharedAt) : '' }}</div>
          </div>
          <el-badge :value="unreadMap[it.id] || 0" :hidden="!unreadMap[it.id]" :max="99" class="mr-1">
            <el-button size="small" @click="openCenter(it)">
              <el-icon class="mr-0.5"><ChatDotSquare /></el-icon>评论
            </el-button>
          </el-badge>
          <el-button size="small" type="primary" plain @click="$router.push(`/shared-calendar/${it.id}`)">
            查看日历
          </el-button>
        </div>
      </div>
      <el-empty v-else description="暂时还没有好友向你分享人生日历" />
    </template>

    <!-- 详情模式 -->
    <template v-else>
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-3">
          <el-button text @click="$router.push('/shared-calendar')">
            <el-icon><ArrowLeft /></el-icon>
          </el-button>
          <UserAvatar :src="ownerInfo?.avatarUrl" :name="ownerInfo?.displayName" :size="40" />
          <div>
            <h1 class="text-xl font-bold text-gray-900 flex items-center gap-2">
              {{ ownerInfo?.displayName }} 的人生日历
              <span class="text-xs font-normal px-2 py-0.5 rounded-full bg-indigo-50 text-indigo-500">好友分享</span>
            </h1>
            <p class="text-gray-400 text-xs mt-0.5">仅可见双方；评价仅你与 TA 可见</p>
          </div>
        </div>
      </div>

      <el-card shadow="never">
        <SharedLifeCalendar
          v-if="summaryData"
          :owner-id="Number(ownerId)"
          :birth-date="summaryData.birthDate"
          :lifespan="summaryData.expectedLifespan || 80"
          :summary-data="summaryData"
        />
        <div v-else class="text-center py-8 text-gray-400">加载中…</div>
      </el-card>
    </template>

    <!-- 与该好友的私密评论中心 -->
    <CalendarCommentCenterDialog
      v-model:visible="centerVisible"
      :friend="centerFriend"
      @read="loadUnread"
    />
  </div>
</template>

<script setup>
import { onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import UserAvatar from '../components/UserAvatar.vue'
import SharedLifeCalendar from '../components/SharedLifeCalendar.vue'
import CalendarCommentCenterDialog from '../components/CalendarCommentCenterDialog.vue'
import { calendarShareApi, sharedCalendarApi, calendarCenterApi } from '../api'

const route = useRoute()
const ownerId = ref(route.params.userId || '')
const items = ref([])
const ownerInfo = ref(null)
const summaryData = ref(null)

const centerVisible = ref(false)
const centerFriend = ref(null)
const unreadMap = ref({})

function fmt(s) {
  return s ? s.slice(0, 10) : ''
}

function openCenter(it) {
  centerFriend.value = it
  centerVisible.value = true
}

async function loadUnread() {
  try {
    const u = await calendarCenterApi.unread()
    const map = {}
    for (const x of u.items || []) map[x.id] = x.unread
    unreadMap.value = map
  } catch { /* 忽略 */ }
}

async function loadList() {
  try {
    const data = await calendarShareApi.received()
    items.value = data.items || []
  } catch {
    /* 拦截器已提示 */
  }
  loadUnread()
}

async function loadDetail() {
  if (!ownerId.value) return
  summaryData.value = null
  ownerInfo.value = null
  try {
    const s = await sharedCalendarApi.summary(ownerId.value)
    summaryData.value = s
    ownerInfo.value = { displayName: s.ownerName, avatarUrl: s.ownerAvatar }
  } catch {
    /* 拦截器已提示：未被分享等 */
  }
}

watch(() => route.params.userId, (v) => {
  ownerId.value = v || ''
  if (ownerId.value) loadDetail()
  else loadList()
})

onMounted(() => {
  if (ownerId.value) loadDetail()
  else loadList()
})
</script>
