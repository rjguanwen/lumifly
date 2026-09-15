<template>
  <div class="space-y-6">
    <!-- 标题 -->
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">速记语录</h1>
        <p class="text-gray-500 mt-1">记下那些突发奇想的话与书中的名言警句</p>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon class="mr-1"><ChatDotSquare /></el-icon>记一句
      </el-button>
    </div>

    <!-- 首屏加载骨架（与单行列表同形） -->
    <div v-if="loading && !items.length" class="space-y-3" aria-hidden="true">
      <div v-for="i in 5" :key="'sk' + i" class="bg-white rounded-2xl border border-gray-100 shadow-sm px-5 py-4 flex items-start gap-3">
        <div class="skl rounded-full w-5 h-5 shrink-0" />
        <div class="flex-1 min-w-0 space-y-2.5">
          <div class="skl h-4 w-3/4" />
          <div class="skl h-3 w-1/3" />
        </div>
        <div class="skl h-6 w-28 rounded-full shrink-0" />
      </div>
    </div>

    <!-- 每条语录一行，由新到旧（列表数据本身按 id 倒序返回） -->
    <div v-else-if="items.length" class="space-y-3">
      <div
        v-for="n in items"
        :key="n.id"
        class="bg-white rounded-2xl border border-gray-100 shadow-sm hover:shadow-md transition-shadow px-5 py-4 flex flex-col sm:flex-row sm:items-start gap-3"
      >
        <div class="flex items-start gap-2.5 flex-1 min-w-0">
          <el-icon class="text-cyan-500 mt-1 shrink-0" :size="18"><ChatDotSquare /></el-icon>
          <div class="min-w-0 flex-1">
            <p class="text-gray-800 leading-relaxed whitespace-pre-line break-words">{{ n.content }}</p>
            <div v-if="n.tags?.length" class="flex flex-wrap gap-1 mt-2">
              <span
                v-for="tag in n.tags"
                :key="tag.id"
                class="px-2 py-0.5 text-xs rounded-full bg-gray-100 text-gray-600"
              >{{ tag.name }}</span>
            </div>
          </div>
        </div>

        <div class="flex items-center gap-1 shrink-0 self-end sm:self-start">
          <span class="text-xs text-gray-400 mr-1 whitespace-nowrap">{{ fmtDate(n.createdAt) }}</span>
          <el-button size="small" text @click="editNote(n)">
            <el-icon><Edit /></el-icon><span class="ml-0.5">编辑</span>
          </el-button>
          <el-button size="small" text type="danger" @click="deleteNote(n.id)">
            <el-icon><Delete /></el-icon>
          </el-button>
          <el-button
            v-if="!pubMap[n.id]"
            size="small"
            type="primary"
            plain
            :loading="publishingId === n.id"
            @click="publishNote(n)"
          >
            <el-icon class="mr-0.5"><Share /></el-icon>发布
          </el-button>
          <el-button
            v-else-if="pubMap[n.id].status === 'published'"
            size="small"
            type="success"
            :loading="publishingId === n.id"
            @click="unpublishNote(n)"
          >
            <el-icon class="mr-0.5"><Check /></el-icon>已发布
          </el-button>
          <el-button
            v-else-if="pubMap[n.id].status === 'pending'"
            size="small"
            type="warning"
            plain
            :loading="publishingId === n.id"
            @click="pendingNote(n)"
          >
            <el-icon class="mr-0.5"><Clock /></el-icon>审核中
          </el-button>
          <el-button
            v-else
            size="small"
            type="danger"
            plain
            :loading="publishingId === n.id"
            @click="publishNote(n)"
          >
            <el-icon class="mr-0.5"><RefreshRight /></el-icon>未通过
          </el-button>
        </div>
      </div>
    </div>
    <el-empty v-else-if="loaded && !items.length" description="还没有语录，把脑海中的话随手记下来吧" />

    <!-- 滚动加载：接近底部自动追加下一页 -->
    <div v-if="items.length && items.length < total" ref="sentinel" class="py-4 text-center text-sm text-gray-400">
      {{ loading ? '加载中…' : '继续下滑加载更多' }}
    </div>
    <div v-if="items.length && items.length >= total && total > 0" class="py-4 text-center text-xs text-gray-300">
      — 已经到底啦 —
    </div>

    <!-- 新建 / 编辑弹窗 -->
    <el-dialog v-model="showForm" width="620px" top="8vh" destroy-on-close>
      <template #header>
        <div class="flex items-center gap-2.5">
          <div class="w-9 h-9 rounded-xl flex items-center justify-center" :class="editingId ? 'bg-cyan-50 text-cyan-600' : 'bg-cyan-50 text-cyan-600'">
            <el-icon :size="18"><component :is="editingId ? 'EditPen' : 'ChatDotSquare'" /></el-icon>
          </div>
          <div>
            <div class="text-base font-semibold text-gray-800">{{ editingId ? '编辑语录' : '记一句语录' }}</div>
            <div class="text-xs text-gray-400 font-normal">突发奇想的话、书中的名言警句都可以记在这里</div>
          </div>
        </div>
      </template>
      <el-form label-position="top">
        <el-form-item label="内容" required>
          <el-input
            v-model="form.content"
            type="textarea"
            :rows="5"
            maxlength="360"
            show-word-limit
            resize="none"
            placeholder="随手记下一句话，最多 360 字..."
          />
        </el-form-item>
        <el-form-item label="标签">
          <TagInput v-model="form.tagIds" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="closeForm">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveNote">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import TagInput from '../components/TagInput.vue'
import { quickNoteApi, squareApi } from '../api'
import { useInfiniteScroll } from '../composables/useInfiniteScroll'

const route = useRoute()
const router = useRouter()
const showForm = ref(false)
const saving = ref(false)
const editingId = ref(null)
const form = reactive({ content: '', tagIds: [] })

// 发布到广场的状态映射 id -> { id: publicationId, status }
const pubMap = ref({})
const publishingId = ref(null)

function fmtDate(s) {
  if (!s) return ''
  return s.slice(0, 10)
}

// 分页滚动加载：整页刷新调用 load()，滚动到底由内部自动追加下一页
const { items, total, loading, loaded, sentinel, load } = useInfiniteScroll(
  (params) => quickNoteApi.list(params),
  { pageSize: 24 },
)

// 列表（含滚动追加）变化后刷新每条语录的广场发布状态
watch(items, () => { loadPubState() })

async function loadPubState() {
  try {
    const data = await squareApi.mine()
    const map = {}
    for (const p of data.items || []) {
      if (p.sourceType === 'quick') map[Number(p.sourceId)] = { id: p.id, status: p.status || 'published' }
    }
    pubMap.value = map
  } catch {
    /* 忽略 */
  }
}

async function init() {
  await load()
  if (route.query.edit) {
    const id = Number(route.query.edit)
    const item = items.value.find((x) => x.id === id)
    if (item) {
      fillForm(item)
      showForm.value = true
    }
  }
}
onMounted(init)

function openCreate() {
  editingId.value = null
  form.content = ''
  form.tagIds = []
  showForm.value = true
}

function fillForm(note) {
  editingId.value = note.id
  form.content = note.content || ''
  form.tagIds = note.tags?.map((t) => t.id) || []
}

function editNote(note) {
  fillForm(note)
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  if (route.query.edit) router.replace('/quick-notes')
}

async function saveNote() {
  const content = form.content.trim()
  if (!content) {
    ElMessage.warning('请输入内容')
    return
  }
  saving.value = true
  try {
    const payload = { content, tagIds: form.tagIds }
    if (editingId.value) {
      await quickNoteApi.update(editingId.value, payload)
      ElMessage.success('语录已更新')
    } else {
      await quickNoteApi.create(payload)
      ElMessage.success('已记下这句语录')
    }
    closeForm()
    load()
  } catch {
    /* 拦截器已提示 */
  } finally {
    saving.value = false
  }
}

async function deleteNote(id) {
  try {
    await ElMessageBox.confirm('确定删除这条语录？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await quickNoteApi.remove(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* 忽略 */
  }
}

async function afterPublish(note, item) {
  pubMap.value[note.id] = { id: item.id, status: item.status || 'published' }
  if (pubMap.value[note.id].status === 'pending') {
    ElMessage.info('内容可能包含敏感信息，已转入人工审核，通过后将在广场展示')
  } else if (pubMap.value[note.id].status === 'rejected') {
    ElMessage.warning('该内容未能通过审核，请修改后重新提交')
  } else {
    ElMessage.success('已发布到广场')
  }
}

async function publishNote(note) {
  publishingId.value = note.id
  try {
    const item = await squareApi.publish('quick', note.id)
    afterPublish(note, item)
  } catch {
    /* 拦截器已提示 */
  } finally {
    publishingId.value = null
  }
}

async function unpublishNote(note) {
  try {
    await ElMessageBox.confirm('撤回后将不再在广场展示，确定撤回？', '撤回发布', { type: 'warning' })
  } catch {
    return
  }
  publishingId.value = note.id
  try {
    await squareApi.unpublish(pubMap.value[note.id].id)
    delete pubMap.value[note.id]
    ElMessage.success('已从广场撤回')
  } catch {
    /* 拦截器已提示 */
  } finally {
    publishingId.value = null
  }
}

async function pendingNote(note) {
  try {
    await ElMessageBox.confirm(
      '该语录正在人工审核中。如需修改后重新发布，请先撤回；不操作则继续等待审核结果。',
      '内容审核中',
      { type: 'info', confirmButtonText: '撤回', cancelButtonText: '继续等待' },
    )
  } catch {
    return
  }
  publishingId.value = note.id
  try {
    await squareApi.unpublish(pubMap.value[note.id].id)
    delete pubMap.value[note.id]
    ElMessage.success('已撤回，修改后重新点击发布即可')
  } catch {
    /* 拦截器已提示 */
  } finally {
    publishingId.value = null
  }
}
</script>

<style scoped>
/* 加载骨架占位块（shimmer 动效由 base.css 全局 @keyframes skl-shimmer 提供）
   不要依赖 base.css 的全局 .skl（Tailwind 处理后该顶层规则会丢失），必须在 scoped 内定义 */
.skl {
  border-radius: 6px;
  background-color: #e2e8f0;
  background-image: linear-gradient(
    90deg,
    rgba(255, 255, 255, 0) 0,
    rgba(255, 255, 255, 0.5) 50%,
    rgba(255, 255, 255, 0) 100%
  );
  background-size: 220% 100%;
  animation: skl-shimmer 1.4s ease-in-out infinite;
}
</style>
