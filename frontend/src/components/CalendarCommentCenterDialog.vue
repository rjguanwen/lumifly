<template>
  <el-dialog
    :model-value="visible"
    width="700px"
    :title="'与 ' + (friend?.displayName || '好友') + ' 的日历私密评论'"
    :close-on-click-modal="false"
    append-to-body
    top="5vh"
    @update:model-value="$emit('update:visible', $event)"
  >
    <p class="text-sm text-gray-500 -mt-2 mb-3">
      这些评论仅你们双方可见，按随记 / 大事分组展示。点某条内容下的「回复」即可继续交流。
    </p>

    <div v-if="loading" class="text-center py-10 text-gray-400">加载中…</div>
    <div v-else-if="!groups.length" class="text-center py-10 text-gray-300 text-sm">还没有往来评论</div>
    <div v-else class="space-y-4 max-h-[60vh] overflow-y-auto pr-1">
      <div v-for="(g, i) in groups" :key="i" class="rounded-2xl border border-gray-100 overflow-hidden">
        <!-- 内容标题条 -->
        <div class="flex items-center gap-2 px-4 py-2.5 bg-gray-50/80">
          <span
            class="px-1.5 py-0.5 rounded text-[11px] shrink-0"
            :style="g.targetType === 'milestone' ? { background: '#fef3c7', color: '#d97706' } : { background: '#eef2ff', color: '#4a6cf7' }"
          >{{ g.targetType === 'milestone' ? '大事' : '随记' }}</span>
          <span class="text-xs text-gray-400 shrink-0">{{ g.isMine ? '我的' : g.ownerName + ' 的' }}</span>
          <span class="text-sm font-medium text-gray-800 truncate flex-1">{{ g.title }}</span>
          <span class="text-[11px] text-gray-300">{{ g.date }}</span>
        </div>

        <!-- 评论列表 -->
        <div class="px-4 py-2 space-y-3">
          <div v-for="cm in g.comments" :key="cm.id" class="flex gap-2.5">
            <UserAvatar :src="cm.authorAvatar" :name="cm.authorName" :size="30" class="shrink-0" />
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2">
                <span class="text-xs font-medium text-gray-700">{{ cm.authorName }}</span>
                <span v-if="cm.isMine" class="text-[10px] px-1 rounded bg-indigo-50 text-indigo-500">我</span>
                <span class="text-[11px] text-gray-300 flex-1">{{ fmt(cm.createdAt) }}</span>
                <el-button v-if="cm.isMine" size="small" text type="danger" @click="removeComment(g, cm)">
                  <el-icon><Delete /></el-icon>
                </el-button>
              </div>
              <div class="text-sm text-gray-700 whitespace-pre-line leading-relaxed mt-0.5">{{ cm.content }}</div>
            </div>
          </div>
        </div>

        <!-- 回复 -->
        <div class="px-4 pb-3 flex justify-end">
          <el-button size="small" text type="primary" @click="replyTo(g)">
            <el-icon class="mr-0.5"><ChatLineSquare /></el-icon>回复
          </el-button>
        </div>
      </div>
    </div>

    <CalendarCommentDialog
      v-model:visible="replyVisible"
      :owner-id="replyOwner"
      :target-type="replyType"
      :target-id="replyTarget"
      :title="replyTitle"
      @update:visible="(v) => !v && load()"
    />
  </el-dialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import UserAvatar from './UserAvatar.vue'
import CalendarCommentDialog from './CalendarCommentDialog.vue'
import { calendarCenterApi, calendarCommentApi } from '../api'

const props = defineProps({
  visible: Boolean,
  friend: { type: Object, default: null },
})
const emit = defineEmits(['update:visible', 'read'])

const groups = ref([])
const loading = ref(false)

function fmt(s) {
  return s ? s.slice(0, 16).replace('T', ' ') : ''
}

async function load() {
  if (!props.visible || !props.friend) return
  loading.value = true
  try {
    const data = await calendarCenterApi.center(props.friend.id)
    groups.value = data.groups || []
  } catch {
    /* 拦截器已提示 */
  } finally {
    loading.value = false
  }
  // 打开即标记已读，并通知父级刷新角标
  try {
    await calendarCenterApi.read(props.friend.id)
    emit('read')
  } catch { /* 忽略 */ }
}

async function removeComment(g, cm) {
  try {
    await ElMessageBox.confirm('确定删除这条评论？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await calendarCommentApi.remove(cm.id)
    g.comments = g.comments.filter((x) => x.id !== cm.id)
    if (!g.comments.length) groups.value = groups.value.filter((x) => x !== g)
    ElMessage.success('已删除')
  } catch {
    /* 拦截器已提示 */
  }
}

const replyVisible = ref(false)
const replyOwner = ref(0)
const replyType = ref('record')
const replyTarget = ref(0)
const replyTitle = ref('')
function replyTo(g) {
  replyOwner.value = g.ownerUserId
  replyType.value = g.targetType
  replyTarget.value = g.targetId
  replyTitle.value = g.title
  replyVisible.value = true
}

watch(() => props.visible, (v) => {
  if (v) load()
  else groups.value = []
})
</script>
