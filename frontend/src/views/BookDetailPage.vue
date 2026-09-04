<template>
  <div class="max-w-3xl mx-auto">
    <div v-if="item" class="bg-white rounded-2xl shadow-sm overflow-hidden border border-gray-100">
      <!-- 顶部工具条 -->
      <div class="px-5 sm:px-8 pt-4 pb-3 flex items-center justify-between flex-wrap gap-2 border-b border-gray-100 bg-gray-50/50">
        <el-button text type="primary" @click="$router.push('/books')">
          <el-icon class="mr-1"><ArrowLeft /></el-icon>返回书库
        </el-button>
        <div class="flex items-center gap-2">
          <PublishButton source-type="book" :source-id="route.params.id" />
          <el-button size="small" @click="edit"><el-icon class="mr-1"><Edit /></el-icon>编辑</el-button>
          <el-button size="small" type="danger" plain @click="remove"><el-icon class="mr-1"><Delete /></el-icon>删除</el-button>
        </div>
      </div>

      <article class="px-5 sm:px-10 py-7">
        <!-- 元信息行 -->
        <div class="flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-gray-500 mb-3">
          <span
            class="text-xs px-2.5 py-0.5 rounded-full text-white font-medium"
            :style="{ background: statusColor(item.status) }"
          >{{ statusLabel(item.status) }}</span>
          <span v-if="item.author">作者：{{ item.author }}</span>
          <span v-if="item.readYear">阅读于 {{ item.readYear }} 年</span>
        </div>

        <div class="flex items-start justify-between flex-wrap gap-4">
          <div class="min-w-[220px]">
            <h1 class="text-2xl sm:text-3xl font-bold text-gray-900 leading-snug">{{ item.name }}</h1>
            <div class="flex flex-wrap items-center gap-2 mt-3">
              <span v-if="item.domain" class="text-xs px-2.5 py-0.5 rounded-full bg-gray-100 text-gray-600">{{ item.domain }}</span>
              <span v-for="tag in item.tags" :key="tag.id" class="text-xs px-2.5 py-0.5 rounded-full" style="background: #eef2ff; color: #4a6cf7">
                {{ tag.name }}
              </span>
            </div>
            <div class="mt-4">
              <span v-if="item.rating" class="flex items-center">
                <StarRating :model-value="item.rating" :size="24" :show-label="false" />
                <span class="text-sm text-gray-500 ml-2">{{ item.rating }} 星</span>
              </span>
              <span v-else class="text-sm text-gray-300">尚未评分</span>
            </div>
          </div>
        </div>

        <!-- 推荐语 -->
        <div v-if="item.recommend" class="mt-6 rounded-xl bg-blue-50 border border-blue-100 px-4 py-3 flex gap-3">
          <el-icon :size="18" color="#4a6cf7" class="mt-0.5 shrink-0"><ChatLineSquare /></el-icon>
          <div class="text-gray-700 leading-relaxed">{{ item.recommend }}</div>
        </div>

        <!-- 书评 -->
        <div class="mt-6">
          <div class="flex items-center gap-2 text-sm font-semibold text-gray-700 mb-3">
            <el-icon color="#f5a623"><Notebook /></el-icon>书评 / 读后感
          </div>
          <div v-if="item.review" class="prose" v-html="item.review" />
          <div v-else class="text-gray-300 text-sm py-6 text-center">还没有写书评，点击右上角「编辑」记录你的读后感吧</div>
        </div>
      </article>

      <footer class="px-5 sm:px-10 py-3.5 border-t border-gray-100 flex flex-wrap gap-4 text-xs text-gray-400">
        <span>创建于 {{ (item.createdAt || '').replace('T', ' ').slice(0, 16) }}</span>
        <span v-if="item.updatedAt && item.updatedAt !== item.createdAt">更新于 {{ (item.updatedAt || '').replace('T', ' ').slice(0, 16) }}</span>
      </footer>
    </div>

    <div v-else class="py-20 text-center text-gray-400">
      <el-empty :description="loadFailed ? '书籍不存在或已被删除' : '加载中...'" />
      <el-button v-if="loadFailed" class="mt-2" @click="$router.push('/books')">返回书库</el-button>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import StarRating from '../components/StarRating.vue'
import PublishButton from '../components/PublishButton.vue'
import { bookApi } from '../api'

const route = useRoute()
const router = useRouter()
const item = ref(null)
const loadFailed = ref(false)

const bookStatuses = [
  { value: 'not_started', label: '未开始', color: '#9ca3af' },
  { value: 'reading', label: '进行中', color: '#4a6cf7' },
  { value: 'finished', label: '已读完', color: '#22a06b' },
  { value: 'abandoned', label: '已中止', color: '#ef4444' },
]
const statusColor = (s) => bookStatuses.find((x) => x.value === s)?.color || '#9ca3af'
const statusLabel = (s) => bookStatuses.find((x) => x.value === s)?.label || s

async function load() {
  try {
    item.value = await bookApi.get(route.params.id)
  } catch {
    loadFailed.value = true
  }
}
onMounted(load)

function edit() {
  router.push({ path: '/books', query: { edit: route.params.id } })
}

async function remove() {
  try {
    await ElMessageBox.confirm('确定删除这本书的记录？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await bookApi.remove(route.params.id)
    ElMessage.success('已删除')
    router.push('/books')
  } catch { /* 拦截器已提示 */ }
}
</script>
