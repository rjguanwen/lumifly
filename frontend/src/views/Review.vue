<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">内容审核</h1>
        <p class="text-gray-500 mt-1">自动审查拦截的内容等待人工复核，通过后才会在广场公开展示</p>
      </div>
      <el-button v-if="isAdmin" @click="openTerms">
        <el-icon class="mr-1"><Setting /></el-icon>敏感词管理
      </el-button>
    </div>

    <!-- 审核队列 -->
    <div v-if="items.length" class="space-y-4">
      <div v-for="p in items" :key="p.id" class="bg-white rounded-2xl border border-gray-100 shadow-sm overflow-hidden">
        <div class="flex items-center gap-3 px-5 py-3 border-b border-gray-50 bg-amber-50/40">
          <UserAvatar :src="p.authorAvatar" :name="p.authorName" :size="34" />
          <span class="text-sm font-medium text-gray-800">{{ p.authorName }}</span>
          <span class="text-xs text-gray-400">发布于 {{ fmtTime(p.createdAt) }}</span>
          <span
            class="px-2 py-0.5 rounded-full text-xs"
            :style="{ background: typeMeta(p).bg, color: typeMeta(p).color }"
          >{{ typeMeta(p).label || p.sourceType }}</span>
          <div class="flex-1" />
          <el-tag v-for="cat in catList(p.reviewCategories)" :key="cat" size="small" type="danger" effect="dark" class="!mr-1">
            {{ catLabel(cat) }}
          </el-tag>
        </div>
        <div class="px-5 py-4">
          <h3 class="font-semibold text-gray-900 mb-1.5">{{ p.title }}</h3>
          <div class="prose prose-sm max-w-none text-gray-700 max-h-64 overflow-y-auto" v-html="p.content" />
        </div>
        <div class="px-5 py-3 border-t border-gray-50 flex justify-end gap-2 bg-gray-50/60">
          <el-button size="small" @click="reject(p)">驳回</el-button>
          <el-button size="small" type="primary" @click="approve(p)">通过并发布</el-button>
        </div>
      </div>
    </div>
    <el-empty v-else-if="!loading" description="暂无待审核内容" />

    <!-- 敏感词库管理（仅管理员） -->
    <el-dialog v-model="showTerms" width="720px" top="6vh" title="敏感词库管理" destroy-on-close>
      <p class="text-xs text-gray-400 mb-4">
        命中内置或自定义敏感词的内容将进入人工审核。请在对应类别输入自定义词，每行一个；内置词库不可编辑。
      </p>
      <div v-if="termsReady" class="space-y-5 max-h-[60vh] overflow-y-auto pr-1">
        <div v-for="(label, cat) in labels" :key="cat" class="rounded-xl border border-gray-100 p-4">
          <div class="text-sm font-medium text-gray-700 mb-1.5 flex items-center gap-2">
            {{ label }}
            <span class="text-xs font-normal text-gray-400">（类别 {{ cat }}）</span>
          </div>
          <el-input
            v-model="customForm[cat]"
            type="textarea"
            :rows="3"
            placeholder="自定义敏感词，每行一个；为空则仅使用内置词库"
          />
          <div v-if="builtin[cat]?.length" class="mt-2 flex flex-wrap gap-1">
            <span v-for="w in builtin[cat]" :key="w" class="px-1.5 py-0.5 text-[11px] rounded bg-gray-100 text-gray-400">{{ w }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="showTerms = false">取消</el-button>
        <el-button type="primary" :loading="savingTerms" @click="saveTerms">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import UserAvatar from '../components/UserAvatar.vue'
import { adminApi, moderateApi } from '../api'
import { useAuthStore } from '../stores/auth'
import { squareTypeOf } from '../utils/squareMeta'

const auth = useAuthStore()
const isAdmin = computed(() => auth.user?.role === 'admin')

const items = ref([])
const loading = ref(false)

const fmtTime = (s) => (s ? s.slice(0, 16).replace('T', ' ') : '')
const catLabel = (cat) => (labels.value[cat] || cat)

// 类型徽标
const typeMeta = (p) => squareTypeOf(p.sourceType)

function catList(s) {
  return String(s || '').split(',').filter(Boolean)
}

async function load() {
  loading.value = true
  try {
    const data = await moderateApi.queue()
    items.value = data.items || []
  } catch {
    /* 拦截器已提示 */
  } finally {
    loading.value = false
  }
}

async function approve(p) {
  try {
    await ElMessageBox.confirm('确认通过该内容并发布到广场？', '审核通过', { type: 'info' })
  } catch {
    return
  }
  try {
    await moderateApi.approve(p.id)
    ElMessage.success('已通过并发布')
    load()
  } catch {
    /* 拦截器已提示 */
  }
}

async function reject(p) {
  try {
    await ElMessageBox.confirm('驳回后该内容仅作者可见，作者可修改后重新提交。确定驳回？', '驳回内容', { type: 'warning' })
  } catch {
    return
  }
  try {
    await moderateApi.reject(p.id)
    ElMessage.success('已驳回')
    load()
  } catch {
    /* 拦截器已提示 */
  }
}

// ---------- 敏感词库（管理员） ----------
const showTerms = ref(false)
const termsReady = ref(false)
const labels = ref({})
const builtin = ref({})
const customForm = reactive({})

async function openTerms() {
  showTerms.value = true
  termsReady.value = false
  try {
    const data = await moderateApi.terms()
    labels.value = data.labels || {}
    builtin.value = data.builtin || {}
    const cats = Object.keys(data.labels || {})
    for (const cat of cats) {
      customForm[cat] = (data.custom?.[cat] || []).join('\n')
    }
    termsReady.value = true
  } catch {
    /* 拦截器已提示 */
  }
}

async function saveTerms() {
  const custom = {}
  for (const cat of Object.keys(customForm)) {
    const list = customForm[cat].split('\n').map((s) => s.trim()).filter(Boolean)
    if (list.length) custom[cat] = list
  }
  try {
    await adminApi.saveModerationTerms(custom)
    ElMessage.success('词库已保存，新内容将按更新后的词库审查')
    showTerms.value = false
  } catch {
    /* 拦截器已提示 */
  }
}

onMounted(load)
</script>
