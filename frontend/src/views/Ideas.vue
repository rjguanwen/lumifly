<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">想法灵感</h1>
        <p class="text-gray-500 mt-1">捕捉每一个灵光一闪</p>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon class="mr-1"><Plus /></el-icon>记录灵感
      </el-button>
    </div>

    <!-- 瀑布流卡片 -->
    <div v-if="items.length" class="columns-1 sm:columns-2 xl:columns-3 gap-4">
      <el-card
        v-for="idea in items"
        :key="idea.id"
        shadow="hover"
        class="!rounded-2xl break-inside-avoid mb-4 cursor-pointer group"
        @click="viewIdea(idea)"
      >
        <div class="space-y-2">
          <div class="flex items-start justify-between gap-2" @click.stop>
            <h3 v-if="idea.title" class="font-medium text-gray-900 text-lg">{{ idea.title }}</h3>
            <div class="flex gap-1 shrink-0">
              <el-button size="small" text @click="editIdea(idea)"><el-icon><Edit /></el-icon></el-button>
              <el-button size="small" text type="danger" @click="deleteIdea(idea.id)"><el-icon><Delete /></el-icon></el-button>
            </div>
          </div>

          <div v-if="idea.content" class="text-sm text-gray-600 prose prose-sm" v-html="idea.content" />
          <div v-if="idea.media?.length" class="flex gap-2 flex-wrap mt-2">
            <template v-for="m in idea.media" :key="m.id">
              <img v-if="m.mimeType?.startsWith('image/')" :src="m.url" class="w-full rounded-lg object-cover max-h-48" />
              <video v-else-if="m.mimeType?.startsWith('video/')" :src="m.url" controls class="w-full rounded-lg max-h-48" @click.stop />
            </template>
          </div>
          <div v-if="idea.tags?.length" class="flex gap-1 flex-wrap mt-2">
            <span v-for="tag in idea.tags" :key="tag.id" class="px-2 py-0.5 text-xs rounded-full bg-gray-100 text-gray-600">
              {{ tag.name }}
            </span>
          </div>
          <div class="text-xs text-gray-400 pt-1">
            <span>{{ idea.createdAt?.slice(0, 10) }}</span>
          </div>
        </div>
      </el-card>
    </div>
    <el-empty v-else description="还没有记录灵感，把脑海中的想法写下来吧" />

    <!-- 新建/编辑弹窗 -->
    <el-dialog v-model="showForm" width="920px" top="5vh" destroy-on-close class="editor-dialog">
      <template #header>
        <div class="flex items-center gap-2.5">
          <div class="w-9 h-9 rounded-xl flex items-center justify-center" :class="editingId ? 'bg-orange-50 text-orange-500' : 'bg-yellow-50 text-yellow-500'">
            <el-icon :size="18"><component :is="editingId ? 'EditPen' : 'Lightning'" /></el-icon>
          </div>
          <div>
            <div class="text-base font-semibold text-gray-800">{{ editingId ? '编辑灵感' : '记录一个灵感' }}</div>
            <div class="text-xs text-gray-400 font-normal">内容支持富文本与 Markdown 两种编辑方式</div>
          </div>
        </div>
      </template>
      <el-form label-position="top">
        <el-form-item label="标题（可选）">
          <el-input v-model="form.title" placeholder="给灵感起个名字..." />
        </el-form-item>
        <el-form-item label="内容">
          <RichTextEditor v-model="form.content" placeholder="记录你的想法，可以用 Markdown 书写..." />
        </el-form-item>
        <el-form-item label="附件">
          <MediaUploader v-model="form.mediaIds" multiple />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="resetForm">取消</el-button>
        <el-button type="primary" :loading="saving" @click="saveIdea">保存灵感</el-button>
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
import { ideaApi } from '../api'

const route = useRoute()
const router = useRouter()
const showForm = ref(false)
const saving = ref(false)
const editingId = ref(null)
const items = ref([])

const form = reactive({
  title: '',
  content: '',
  mediaIds: [],
})

async function load() {
  try {
    const data = await ideaApi.list()
    items.value = data.items || []
  } catch {
    /* 忽略 */
  }
}

async function handleEditQuery() {
  if (!route.query.edit) return
  const id = Number(route.query.edit)
  try {
    const item = await ideaApi.get(id)
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

function viewIdea(idea) {
  router.push(`/ideas/${idea.id}`)
}

function fillForm(idea) {
  editingId.value = idea.id
  form.title = idea.title || ''
  form.content = idea.content || ''
  form.mediaIds = idea.media?.map((m) => m.id) || []
}

function editIdea(idea) {
  fillForm(idea)
  showForm.value = true
}

function resetForm() {
  editingId.value = null
  form.title = ''
  form.content = ''
  form.mediaIds = []
  showForm.value = false
  if (route.query.edit) router.replace('/ideas')
}

async function saveIdea() {
  if (!form.title && !form.content) {
    ElMessage.warning('请输入标题或内容')
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      await ideaApi.update(editingId.value, { ...form })
      ElMessage.success('灵感已更新')
    } else {
      await ideaApi.create({ ...form })
      ElMessage.success('灵感已记录')
    }
    resetForm()
    load()
  } catch {
    /* 错误已由拦截器提示 */
  } finally {
    saving.value = false
  }
}

async function deleteIdea(id) {
  try {
    await ElMessageBox.confirm('确定删除这条灵感？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await ideaApi.remove(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* 忽略 */
  }
}
</script>
