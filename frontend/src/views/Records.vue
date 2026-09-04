<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">日常记录</h1>
        <p class="text-gray-500 mt-1">记录你的每一天</p>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon class="mr-1"><Plus /></el-icon>写日记
      </el-button>
    </div>

    <!-- 列表 -->
    <div v-if="items.length" class="space-y-4">
      <el-card v-for="record in items" :key="record.id" shadow="hover" class="!rounded-2xl group">
        <div class="flex items-start justify-between">
          <div class="flex-1 min-w-0 cursor-pointer" @click="viewRecord(record)">
            <div class="flex items-center gap-2 mb-2">
              <span class="text-sm text-gray-500">{{ record.recordDate }}</span>
              <span v-if="record.mood" class="text-base leading-none">{{ moodEmoji(record.mood) }}</span>
              <el-tag v-for="tag in record.tags" :key="tag.id" size="small" type="info" effect="plain">
                {{ tag.name }}
              </el-tag>
            </div>
            <h3 v-if="record.title" class="text-lg font-medium text-gray-900 mb-1">{{ record.title }}</h3>
            <div v-if="record.content" class="text-gray-600 text-sm line-clamp-3 prose prose-sm" v-html="record.content" />
            <div v-else class="text-gray-400 text-sm">（无正文内容）</div>
            <div v-if="record.media?.length" class="flex gap-2 mt-3">
              <img
                v-for="m in record.media.filter((x) => x.mimeType?.startsWith('image/'))"
                :key="m.id"
                :src="m.url"
                class="w-16 h-16 object-cover rounded-lg"
              />
            </div>
          </div>
          <div class="flex gap-1 ml-4 shrink-0">
            <el-button size="small" text @click="editRecord(record)"><el-icon><Edit /></el-icon></el-button>
            <el-button size="small" text type="danger" @click="deleteRecord(record.id)"><el-icon><Delete /></el-icon></el-button>
          </div>
        </div>
      </el-card>
    </div>
    <el-empty v-else description="还没有日记，开始记录今天的生活吧" />

    <!-- 新建/编辑弹窗 -->
    <el-dialog v-model="showForm" width="920px" top="5vh" destroy-on-close class="editor-dialog">
      <template #header>
        <div class="flex items-center gap-2.5">
          <div class="w-9 h-9 rounded-xl flex items-center justify-center" :class="editingId ? 'bg-orange-50 text-orange-500' : 'bg-blue-50 text-blue-600'">
            <el-icon :size="18"><component :is="editingId ? 'EditPen' : 'Notebook'" /></el-icon>
          </div>
          <div>
            <div class="text-base font-semibold text-gray-800">{{ editingId ? '编辑日记' : '写一篇新日记' }}</div>
            <div class="text-xs text-gray-400 font-normal">内容支持富文本与 Markdown 两种编辑方式</div>
          </div>
        </div>
      </template>
      <el-form label-position="top">
        <div class="grid grid-cols-2 gap-4">
          <el-form-item label="日期" required>
            <el-date-picker v-model="form.recordDate" type="date" class="w-full" value-format="YYYY-MM-DD" />
          </el-form-item>
          <el-form-item label="心情">
            <MoodPicker v-model="form.mood" />
          </el-form-item>
        </div>
        <el-form-item label="标题（可选）">
          <el-input v-model="form.title" placeholder="给今天起个标题..." />
        </el-form-item>
        <el-form-item label="内容">
          <RichTextEditor v-model="form.content" placeholder="记录今天的所见所闻，可以用 Markdown 书写..." />
        </el-form-item>
        <el-form-item label="标签">
          <TagInput v-model="form.tagIds" />
        </el-form-item>
        <el-form-item label="附件">
          <MediaUploader v-model="form.mediaIds" :initial-items="form.mediaObjects" multiple />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetForm">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveRecord">保存日记</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import RichTextEditor from '../components/RichTextEditor.vue'
import MediaUploader from '../components/MediaUploader.vue'
import TagInput from '../components/TagInput.vue'
import MoodPicker from '../components/MoodPicker.vue'
import { recordApi } from '../api'
import { moodEmoji } from '../utils/helpers'

const route = useRoute()
const router = useRouter()
const showForm = ref(false)
const saving = ref(false)
const editingId = ref(null)
const items = ref([])

const form = reactive({
  title: '',
  content: '',
  recordDate: new Date().toISOString().slice(0, 10),
  mood: null,
  weather: '',
  tagIds: [],
  mediaIds: [],
  mediaObjects: [],
})

async function load() {
  try {
    const params = { limit: 100 }
    if (route.query.from) params.from = route.query.from
    if (route.query.to) params.to = route.query.to
    const data = await recordApi.list(params)
    items.value = data.items || []
  } catch {
    /* 忽略 */
  }
}

async function handleEditQuery() {
  if (!route.query.edit) return
  const id = Number(route.query.edit)
  try {
    const item = await recordApi.get(id)
    if (item) {
      fillForm(item)
      showForm.value = true
    }
  } catch {
    /* 拦截器已提示 */
  }
}

async function init() {
  await load()
  handleEditQuery()
}
onMounted(init)

function openCreate() {
  resetForm()
  showForm.value = true
}

function viewRecord(record) {
  const q = {}
  if (route.query.from) q.from = route.query.from
  if (route.query.to) q.to = route.query.to
  router.push({ path: `/records/${record.id}`, query: q })
}

function fillForm(record) {
  editingId.value = record.id
  form.title = record.title || ''
  form.content = record.content || ''
  form.recordDate = record.recordDate
  form.mood = record.mood
  form.weather = record.weather || ''
  form.tagIds = record.tags?.map((t) => t.id) || []
  form.mediaIds = record.media?.map((m) => m.id) || []
  form.mediaObjects = (record.media || []).map((m) => ({ id: m.id, url: m.url, fileName: m.fileName, mimeType: m.mimeType }))
}

function editRecord(record) {
  fillForm(record)
  showForm.value = true
}

function resetForm() {
  editingId.value = null
  form.title = ''
  form.content = ''
  form.recordDate = new Date().toISOString().slice(0, 10)
  form.mood = null
  form.tagIds = []
  form.mediaIds = []
  form.mediaObjects = []
  showForm.value = false
  if (route.query.edit) {
    router.replace({ path: '/records', query: { from: route.query.from, to: route.query.to } })
  }
}

async function saveRecord() {
  if (!form.recordDate) {
    ElMessage.warning('请选择日期')
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      await recordApi.update(editingId.value, { ...form })
      ElMessage.success('日记已更新')
    } else {
      await recordApi.create({ ...form })
      ElMessage.success('日记已保存')
    }
    resetForm()
    load()
  } catch {
    /* 错误已由拦截器提示 */
  } finally {
    saving.value = false
  }
}

async function deleteRecord(id) {
  try {
    await ElMessageBox.confirm('确定删除这条日记？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await recordApi.remove(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* 忽略 */
  }
}
</script>
