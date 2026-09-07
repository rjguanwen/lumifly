<template>
  <el-dialog
    :model-value="visible"
    title="分享人生日历"
    width="560px"
    :close-on-click-modal="false"
    append-to-body
    @update:model-value="$emit('update:visible', $event)"
  >
    <p class="text-sm text-gray-500 -mt-2 mb-4">
      选择好友后，对方可以在「好友日历」中查看你的人生日历以及其中的大事记与日常记录，并可留下仅你们双方可见的评价。可随时取消分享。
    </p>

    <div v-if="loading" class="text-center py-10 text-gray-400">加载中…</div>
    <div v-else-if="!friends.length" class="text-center py-10">
      <el-empty description="还没有好友，先去「好友」页添加吧" :image-size="90" />
    </div>
    <div v-else class="space-y-2">
      <div
        v-for="f in friends"
        :key="f.id"
        class="flex items-center gap-3 rounded-xl border border-gray-100 px-4 py-3"
        :class="sharedIds.has(f.id) ? 'bg-indigo-50/60 border-indigo-100' : ''"
      >
        <UserAvatar :src="f.avatarUrl" :name="f.displayName" :size="38" />
        <div class="flex-1 min-w-0">
          <div class="text-sm font-medium text-gray-800">{{ f.displayName }}</div>
          <div v-if="f.signature" class="text-xs text-gray-400 truncate">{{ f.signature }}</div>
          <div v-else class="text-xs text-gray-300">好友</div>
        </div>
        <el-badge :value="unreadMap[f.id] || 0" :hidden="!unreadMap[f.id]" :max="99" class="mr-1">
          <el-button size="small" text type="primary" @click="openCenter(f)">
            <el-icon class="mr-0.5"><ChatDotSquare /></el-icon>私密评论
          </el-button>
        </el-badge>
        <el-switch
          :model-value="sharedIds.has(f.id)"
          :loading="busyId === f.id"
          @change="(v) => toggle(f, v)"
        />
      </div>
    </div>
    <template #footer>
      <el-button @click="$emit('update:visible', false)">完成</el-button>
    </template>
  </el-dialog>

  <!-- 与该好友的私密评论中心 -->
  <CalendarCommentCenterDialog
    v-model:visible="centerVisible"
    :friend="centerFriend"
    @read="loadUnread"
  />
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import UserAvatar from './UserAvatar.vue'
import CalendarCommentCenterDialog from './CalendarCommentCenterDialog.vue'
import { friendApi, calendarShareApi, calendarCenterApi } from '../api'

defineProps({ visible: Boolean })
const emit = defineEmits(['update:visible'])

const friends = ref([])
const sharedIds = ref(new Set())
const loading = ref(false)
const busyId = ref(null)

const centerVisible = ref(false)
const centerFriend = ref(null)
const unreadMap = ref({})

async function load() {
  loading.value = true
  try {
    const [f, s, u] = await Promise.all([friendApi.list(), calendarShareApi.mine(), calendarCenterApi.unread()])
    friends.value = f.items || []
    sharedIds.value = new Set((s.items || []).map((x) => x.id))
    const map = {}
    for (const it of u.items || []) map[it.id] = it.unread
    unreadMap.value = map
  } catch {
    /* 拦截器已提示 */
  } finally {
    loading.value = false
  }
}

async function loadUnread() {
  try {
    const u = await calendarCenterApi.unread()
    const map = {}
    for (const it of u.items || []) map[it.id] = it.unread
    unreadMap.value = map
  } catch { /* 忽略 */ }
}

function openCenter(f) {
  centerFriend.value = f
  centerVisible.value = true
}

async function toggle(f, on) {
  busyId.value = f.id
  try {
    if (on) {
      await calendarShareApi.add(f.id)
      ElMessage.success(`已将人生日历分享给 ${f.displayName}`)
    } else {
      await calendarShareApi.remove(f.id)
      ElMessage.success(`已停止向 ${f.displayName} 分享`)
    }
    await load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    busyId.value = null
  }
}

onMounted(load)
</script>
