<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">读书记录</h1>
        <p class="text-gray-500 mt-1">收藏想读的、记录读过的每一本书</p>
      </div>
      <div class="flex items-center gap-2">
        <el-button @click="manageDomainVisible = true">
          <el-icon class="mr-1"><Setting /></el-icon>管理领域
        </el-button>
        <el-button type="primary" @click="openCreate">
          <el-icon class="mr-1"><Plus /></el-icon>添加书籍
        </el-button>
      </div>
    </div>

    <!-- 筛选区 -->
    <div class="flex flex-wrap items-center gap-3">
      <el-select v-model="filterStatus" placeholder="全部进度" clearable style="width: 140px" @change="load">
        <el-option v-for="s in bookStatuses" :key="s.value" :label="s.label" :value="s.value" />
      </el-select>
      <el-select v-model="filterDomain" placeholder="全部领域" clearable filterable style="width: 150px" @change="load">
        <el-option v-for="d in domainOptions" :key="d" :label="d" :value="d" />
      </el-select>
      <el-input
        v-model="keyword"
        placeholder="搜索书名 / 作者"
        clearable
        style="width: 220px"
        @keyup.enter="load"
        @clear="load"
      >
        <template #prefix><el-icon><Search /></el-icon></template>
      </el-input>
      <el-button @click="load">搜索</el-button>
    </div>

    <!-- 书籍卡片 -->
    <div v-if="items.length" class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-3 gap-4">
      <div
        v-for="book in items"
        :key="book.id"
        class="bg-white rounded-2xl border border-gray-100 shadow-sm hover:shadow-md transition-shadow overflow-hidden cursor-pointer group"
        @click="viewBook(book)"
      >
        <!-- 顶部色条 + 状态角标 -->
        <div class="h-1.5" :style="{ background: statusColor(book.status) }" />
        <div class="p-5">
          <div class="flex items-start justify-between gap-2">
            <h3 class="text-base font-semibold text-gray-900 leading-snug group-hover:text-blue-600 transition-colors">
              {{ book.name }}
            </h3>
            <div class="flex gap-1 shrink-0" @click.stop>
              <el-button size="small" text @click="editBook(book)"><el-icon><Edit /></el-icon></el-button>
              <el-button size="small" text type="danger" @click="deleteBook(book.id)"><el-icon><Delete /></el-icon></el-button>
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-x-3 gap-y-1 mt-2 text-sm text-gray-500">
            <span v-if="book.author">{{ book.author }}</span>
            <span v-if="book.readYear">{{ book.readYear }}</span>
          </div>

          <div class="flex flex-wrap items-center gap-2 mt-3">
            <span
              class="text-xs px-2.5 py-0.5 rounded-full text-white font-medium"
              :style="{ background: statusColor(book.status) }"
            >{{ statusLabel(book.status) }}</span>
            <span v-if="book.domain" class="text-xs px-2.5 py-0.5 rounded-full bg-gray-100 text-gray-600">{{ book.domain }}</span>
          </div>

          <div class="mt-3">
            <StarRating v-if="book.rating" :model-value="book.rating" :size="16" :show-label="false" />
            <span v-else class="text-xs text-gray-300">未评分</span>
          </div>

          <p v-if="book.recommend" class="mt-3 text-sm text-gray-600 line-clamp-2 leading-relaxed">{{ book.recommend }}</p>

          <div v-if="book.tags?.length" class="flex gap-1.5 flex-wrap mt-3">
            <span v-for="tag in book.tags" :key="tag.id" class="text-xs px-2 py-0.5 rounded-full" style="background: #eef2ff; color: #4a6cf7">
              {{ tag.name }}
            </span>
          </div>
        </div>
      </div>
    </div>
    <el-empty v-else description="还没有书籍记录，添加你的第一本书吧" />

    <!-- 新建/编辑弹窗 -->
    <el-dialog v-model="showForm" width="920px" top="5vh" destroy-on-close class="editor-dialog">
      <template #header>
        <div class="flex items-center gap-2.5">
          <div class="w-9 h-9 rounded-xl flex items-center justify-center" :class="editingId ? 'bg-orange-50 text-orange-500' : 'bg-blue-50 text-blue-600'">
            <el-icon :size="18"><component :is="editingId ? 'EditPen' : 'Collection'" /></el-icon>
          </div>
          <div>
            <div class="text-base font-semibold text-gray-800">{{ editingId ? '编辑书籍' : '添加一本书' }}</div>
            <div class="text-xs text-gray-400 font-normal">书评支持富文本与 Markdown 书写</div>
          </div>
        </div>
      </template>
      <el-form label-position="top">
        <div class="grid grid-cols-2 gap-x-5">
          <el-form-item label="书名" required>
            <el-input v-model="form.name" placeholder="书籍名称" />
          </el-form-item>
          <el-form-item label="作者">
            <el-input v-model="form.author" placeholder="作者 / 译者" />
          </el-form-item>
        </div>
        <div class="grid grid-cols-3 gap-x-5">
          <el-form-item label="所属领域">
            <el-select v-model="form.domain" placeholder="选择维护的领域，或输入自定义" filterable allow-create clearable style="width: 100%">
              <el-option v-for="d in domainOptions" :key="d" :label="d" :value="d" />
            </el-select>
          </el-form-item>
          <el-form-item label="阅读年份">
            <el-date-picker v-model="form.readYear" type="year" placeholder="阅读年份" value-format="YYYY" style="width: 100%" />
          </el-form-item>
          <el-form-item label="星级评价">
            <div class="flex items-center gap-2 pt-1">
              <StarRating v-model="form.rating" editable :size="26" :show-label="false" />
              <span class="text-xs text-gray-400">{{ form.rating ? form.rating + ' 星' : '未评价' }}</span>
            </div>
          </el-form-item>
        </div>
        <el-form-item label="阅读进度" required>
          <el-radio-group v-model="form.status">
            <el-radio-button v-for="s in bookStatuses" :key="s.value" :value="s.value">{{ s.label }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="推荐语（可选，一句话推荐）">
          <el-input v-model="form.recommend" type="textarea" :rows="2" maxlength="200" show-word-limit placeholder="这本书值得推荐吗？一句话说说..." />
        </el-form-item>
        <el-form-item label="书评 / 读后感">
          <RichTextEditor v-model="form.review" placeholder="写点 Markdown 书评，例如：**喜欢**它的……" />
        </el-form-item>
        <el-form-item label="标签">
          <TagInput v-model="form.tagIds" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetForm">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveBook">保存书籍</el-button>
      </template>
    </el-dialog>

    <DomainManageDialog v-model:visible="manageDomainVisible" @changed="onDomainsChanged" />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import RichTextEditor from '../components/RichTextEditor.vue'
import TagInput from '../components/TagInput.vue'
import StarRating from '../components/StarRating.vue'
import DomainManageDialog from '../components/DomainManageDialog.vue'
import { bookApi, bookDomainApi } from '../api'

const route = useRoute()
const router = useRouter()
const showForm = ref(false)
const saving = ref(false)
const editingId = ref(null)
const items = ref([])

const filterStatus = ref('')
const filterDomain = ref('')
const keyword = ref('')

const manageDomainVisible = ref(false)
const domains = ref([])
const domainOptions = computed(() => domains.value.map((d) => d.name))

const bookStatuses = [
  { value: 'not_started', label: '未开始', color: '#9ca3af' },
  { value: 'reading', label: '进行中', color: '#4a6cf7' },
  { value: 'finished', label: '已读完', color: '#22a06b' },
  { value: 'abandoned', label: '已中止', color: '#ef4444' },
]
const statusColor = (s) => bookStatuses.find((x) => x.value === s)?.color || '#9ca3af'
const statusLabel = (s) => bookStatuses.find((x) => x.value === s)?.label || s

async function loadDomains() {
  try {
    domains.value = (await bookDomainApi.list()) || []
  } catch { /* 忽略 */ }
}

function onDomainsChanged() {
  loadDomains()
  load()
}

const form = reactive({
  name: '',
  author: '',
  domain: '',
  status: 'not_started',
  rating: 0,
  readYear: '',
  recommend: '',
  review: '',
  tagIds: [],
})

async function load() {
  try {
    const params = {}
    if (filterStatus.value) params.status = filterStatus.value
    if (filterDomain.value) params.domain = filterDomain.value
    if (keyword.value.trim()) params.keyword = keyword.value.trim()
    const data = await bookApi.list(params)
    items.value = data.items || []
  } catch { /* 忽略 */ }
}

async function handleEditQuery() {
  if (!route.query.edit) return
  const id = Number(route.query.edit)
  try {
    const item = await bookApi.get(id)
    if (item) {
      fillForm(item)
      showForm.value = true
    }
  } catch { /* 拦截器已提示 */ }
}

async function init() {
  loadDomains()
  await load()
  handleEditQuery()
}
onMounted(init)

function openCreate() {
  resetForm()
  showForm.value = true
}

function viewBook(book) {
  router.push(`/books/${book.id}`)
}

function fillForm(book) {
  editingId.value = book.id
  form.name = book.name || ''
  form.author = book.author || ''
  form.domain = book.domain || ''
  form.status = book.status || 'not_started'
  form.rating = book.rating || 0
  form.readYear = book.readYear || ''
  form.recommend = book.recommend || ''
  form.review = book.review || ''
  form.tagIds = book.tags?.map((t) => t.id) || []
}

function editBook(book) {
  fillForm(book)
  showForm.value = true
}

function resetForm() {
  editingId.value = null
  form.name = ''
  form.author = ''
  form.domain = ''
  form.status = 'not_started'
  form.rating = 0
  form.readYear = ''
  form.recommend = ''
  form.review = ''
  form.tagIds = []
  showForm.value = false
  if (route.query.edit) router.replace('/books')
}

async function saveBook() {
  if (!form.name.trim()) {
    ElMessage.warning('请填写书名')
    return
  }
  saving.value = true
  const customDomain = form.domain?.trim()
  try {
    if (editingId.value) {
      await bookApi.update(editingId.value, { ...form })
      ElMessage.success('书籍已更新')
    } else {
      await bookApi.create({ ...form })
      ElMessage.success('书籍已添加')
    }
    resetForm()
    // 使用了自定义领域则自动加入领域库
    if (customDomain && !domainOptions.value.includes(customDomain)) {
      try {
        await bookDomainApi.create(customDomain)
        loadDomains()
      } catch { /* 忽略 */ }
    }
    load()
  } catch { /* 拦截器已提示 */ } finally {
    saving.value = false
  }
}

async function deleteBook(id) {
  try {
    await ElMessageBox.confirm('确定删除这本书的记录？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await bookApi.remove(id)
    ElMessage.success('已删除')
    load()
  } catch { /* 忽略 */ }
}
</script>
