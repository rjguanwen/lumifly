<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">人生大事记</h1>
        <p class="text-gray-500 mt-1">记录人生中的重要时刻</p>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon class="mr-1"><Plus /></el-icon>记录大事
      </el-button>
    </div>

    <!-- 分类筛选 -->
    <div class="flex gap-2 flex-wrap">
      <el-button
        v-for="cat in milestoneCategories"
        :key="cat.value"
        :type="selectedCategory === cat.value ? 'primary' : ''"
        size="small"
        @click="selectedCategory = selectedCategory === cat.value ? '' : cat.value"
      >
        {{ cat.label }}
      </el-button>
    </div>

    <!-- 时间线 -->
    <div v-if="filteredItems.length" class="timeline">
      <div v-for="milestone in filteredItems" :key="milestone.id" class="timeline-item">
        <span class="timeline-dot" :style="{ background: categoryColor(milestone.category) }" />
        <el-card shadow="hover" class="!rounded-2xl group">
          <div class="flex items-start justify-between">
            <div class="flex-1 min-w-0 cursor-pointer" @click="viewMilestone(milestone)">
              <div class="flex items-center gap-2 mb-2">
                <span class="text-sm text-gray-500">{{ milestone.eventDate }}</span>
                <el-tag size="small" :type="categoryBadge(milestone.category)">{{ categoryLabel(milestone.category) }}</el-tag>
                <span v-for="i in milestone.importance" :key="i" class="text-amber-400 text-xs">★</span>
              </div>
              <h3 class="text-lg font-semibold text-gray-900 mb-1">{{ milestone.title }}</h3>
              <div v-if="milestone.description" class="text-gray-600 text-sm line-clamp-3 prose prose-sm" v-html="milestone.description" />
              <div v-if="milestone.media?.length" class="flex gap-2 mt-3">
                <img
                  v-for="m in milestone.media.filter((x) => x.mimeType?.startsWith('image/'))"
                  :key="m.id"
                  :src="m.url"
                  loading="lazy"
                  class="w-20 h-20 object-cover rounded-lg"
                />
              </div>
            </div>
            <div class="flex gap-1 ml-4 shrink-0">
              <el-button size="small" text @click="editMilestone(milestone)"><el-icon><Edit /></el-icon></el-button>
              <el-button size="small" text type="danger" @click="deleteMilestone(milestone.id)"><el-icon><Delete /></el-icon></el-button>
            </div>
          </div>
        </el-card>
      </div>
    </div>
    <el-empty v-else description="还没有大事记，记录你人生中的重要时刻吧" />

    <!-- 新建/编辑弹窗 -->
    <el-dialog v-model="showForm" width="920px" top="5vh" destroy-on-close class="editor-dialog">
      <template #header>
        <div class="flex items-center gap-2.5">
          <div class="w-9 h-9 rounded-xl flex items-center justify-center" :class="editingId ? 'bg-orange-50 text-orange-500' : 'bg-blue-50 text-blue-600'">
            <el-icon :size="18"><component :is="editingId ? 'EditPen' : 'Trophy'" /></el-icon>
          </div>
          <div>
            <div class="text-base font-semibold text-gray-800">{{ editingId ? '编辑大事记' : '记录一件人生大事' }}</div>
            <div class="text-xs text-gray-400 font-normal">所见即所得编辑，支持标题、加粗、图片与链接</div>
          </div>
        </div>
      </template>
      <el-form label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="form.title" placeholder="事件名称" />
        </el-form-item>
        <div class="grid grid-cols-2 gap-4">
          <el-form-item label="日期" required>
            <el-date-picker v-model="form.eventDate" type="date" class="w-full" value-format="YYYY-MM-DD" />
          </el-form-item>
          <el-form-item label="分类">
            <el-select v-model="form.category" class="w-full">
              <el-option v-for="cat in milestoneCategories" :key="cat.value" :label="cat.label" :value="cat.value" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item label="重要程度">
          <div class="flex gap-1 items-center">
            <button v-for="i in 5" :key="i" type="button" class="text-2xl transition-transform hover:scale-110" @click="form.importance = i">
              <span :class="i <= form.importance ? 'text-amber-400' : 'text-gray-200'">★</span>
            </button>
            <span class="text-xs text-gray-400 ml-2">{{ importanceLabel }}</span>
          </div>
        </el-form-item>
        <el-form-item label="描述">
          <RichTextEditor v-model="form.description" placeholder="详细描述这个重要事件，支持标题、加粗、列表等格式..." />
        </el-form-item>
        <el-form-item label="附件">
          <MediaUploader v-model="form.mediaIds" :initial-items="form.mediaObjects" multiple />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetForm">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveMilestone">保存大事记</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import RichTextEditor from '../components/RichTextEditor.vue'
import MediaUploader from '../components/MediaUploader.vue'
import { milestoneApi } from '../api'
import {
  milestoneCategories,
  categoryColor,
  categoryBadge,
  categoryLabel,
} from '../utils/helpers'

const route = useRoute()
const router = useRouter()
const showForm = ref(false)
const saving = ref(false)
const editingId = ref(null)
const selectedCategory = ref('')
const items = ref([])

const form = reactive({
  title: '',
  description: '',
  eventDate: new Date().toISOString().slice(0, 10),
  category: 'other',
  importance: 3,
  mediaIds: [],
  mediaObjects: [],
})

const filteredItems = computed(() => {
  if (!selectedCategory.value) return items.value
  return items.value.filter((m) => m.category === selectedCategory.value)
})

const importanceLabel = computed(() => ['', '普通', '值得记住', '重要', '非常重要', '人生大事'][form.importance] || '')

async function load() {
  try {
    const data = await milestoneApi.list({ limit: 100 })
    items.value = data.items || []
  } catch {
    /* 忽略 */
  }
}

async function handleEditQuery() {
  if (!route.query.edit) return
  const id = Number(route.query.edit)
  try {
    const item = await milestoneApi.get(id)
    if (item) {
      fillForm(item)
      showForm.value = true
    }
  } catch { /* 拦截器已提示 */ }
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

function viewMilestone(milestone) {
  router.push(`/milestones/${milestone.id}`)
}

function fillForm(m) {
  editingId.value = m.id
  form.title = m.title
  form.description = m.description || ''
  form.eventDate = m.eventDate
  form.category = m.category
  form.importance = m.importance
  form.mediaIds = m.media?.map((x) => x.id) || []
  form.mediaObjects = (m.media || []).map((x) => ({ id: x.id, url: x.url, fileName: x.fileName, mimeType: x.mimeType }))
}

function editMilestone(m) {
  fillForm(m)
  showForm.value = true
}

function resetForm() {
  editingId.value = null
  form.title = ''
  form.description = ''
  form.eventDate = new Date().toISOString().slice(0, 10)
  form.category = 'other'
  form.importance = 3
  form.mediaIds = []
  showForm.value = false
  if (route.query.edit) router.replace('/milestones')
}

async function saveMilestone() {
  if (!form.title || !form.eventDate) {
    ElMessage.warning('请填写标题和日期')
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      await milestoneApi.update(editingId.value, { ...form })
      ElMessage.success('大事记已更新')
    } else {
      await milestoneApi.create({ ...form })
      ElMessage.success('大事记已保存')
    }
    resetForm()
    load()
  } catch {
    /* 错误已由拦截器提示 */
  } finally {
    saving.value = false
  }
}

async function deleteMilestone(id) {
  try {
    await ElMessageBox.confirm('确定删除这条大事记？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await milestoneApi.remove(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* 忽略 */
  }
}
</script>

<style scoped>
/* ===== 时间线：竖线与圆点严格对齐 ===== */
.timeline {
  position: relative;
}

.timeline-item {
  position: relative;
  padding-left: 52px;
  padding-bottom: 26px;
}

.timeline-item:last-child {
  padding-bottom: 0;
}

/* 竖向连接线：与圆点共用同一中心轴（left: 15px） */
.timeline-item::before {
  content: '';
  position: absolute;
  left: 15px;
  top: 4px;
  bottom: 0;
  width: 2px;
  margin-left: -1px;
  border-radius: 2px;
  background: #e5e7eb;
}

.timeline-item:last-child::before {
  bottom: auto;
  height: 42px;
}

/* 圆形分类标记：圆心精确落在轴线上 */
.timeline-dot {
  position: absolute;
  left: 15px;
  top: 31px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  transform: translate(-50%, -50%);
  border: 3px solid #fff;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.05), 0 2px 6px rgba(0, 0, 0, 0.12);
  z-index: 1;
  box-sizing: content-box;
  transition: transform 0.15s ease;
}

.timeline-item:hover .timeline-dot {
  transform: translate(-50%, -50%) scale(1.18);
}
</style>
