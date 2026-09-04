<template>
  <div class="max-w-3xl mx-auto">
    <div v-if="item" class="bg-white rounded-2xl shadow-sm overflow-hidden border border-gray-100">
      <!-- 顶部工具条 -->
      <div class="px-5 sm:px-8 pt-4 pb-3 flex items-center justify-between flex-wrap gap-2 border-b border-gray-100 bg-gray-50/50">
        <el-button text type="primary" @click="goBack">
          <el-icon class="mr-1"><ArrowLeft /></el-icon>返回列表
        </el-button>
        <div class="flex items-center gap-2">
          <PublishButton source-type="milestone" :source-id="route.params.id" />
          <el-button size="small" @click="edit"><el-icon class="mr-1"><Edit /></el-icon>编辑</el-button>
          <el-button size="small" type="danger" plain @click="remove"><el-icon class="mr-1"><Delete /></el-icon>删除</el-button>
        </div>
      </div>

      <article class="px-5 sm:px-10 py-7">
        <!-- 元信息 -->
        <div class="flex flex-wrap items-center gap-x-3 gap-y-2 text-sm mb-4">
          <el-tag :type="categoryBadge(item.category)" size="small">{{ categoryLabel(item.category) }}</el-tag>
          <span v-for="i in item.importance" :key="i" class="text-amber-400 text-xs">★</span>
          <span class="text-gray-500">{{ item.eventDate }}<template v-if="item.endDate"> ~ {{ item.endDate }}</template></span>
          <el-tag v-for="tag in item.tags || []" :key="tag.id" size="small" type="info" effect="plain">{{ tag.name }}</el-tag>
        </div>

        <h1 class="text-2xl sm:text-3xl font-bold text-gray-900 leading-snug mb-6">{{ item.title }}</h1>

        <!-- 正文 -->
        <div v-if="item.description" class="prose" v-html="item.description" />
        <div v-else class="py-10 text-center text-gray-300">（本条大事记无描述内容）</div>

        <!-- 媒体 -->
        <div v-if="item.media?.length" class="mt-8 grid grid-cols-2 sm:grid-cols-3 gap-3">
          <template v-for="m in item.media" :key="m.id">
            <img
              v-if="m.mimeType?.startsWith('image/')"
              :src="m.url"
              :alt="m.fileName"
              class="w-full aspect-square object-cover rounded-xl cursor-zoom-in hover:opacity-90 transition-opacity"
              @click="openMediaImage(m)"
            />
            <video v-else-if="m.mimeType?.startsWith('video/')" :src="m.url" controls class="w-full rounded-xl max-h-72" />
          </template>
        </div>
      </article>

      <footer class="px-5 sm:px-10 py-3.5 border-t border-gray-100 flex flex-wrap gap-4 text-xs text-gray-400">
        <span>创建于 {{ (item.createdAt || '').replace('T', ' ').slice(0, 16) }}</span>
        <span v-if="item.updatedAt && item.updatedAt !== item.createdAt">更新于 {{ (item.updatedAt || '').replace('T', ' ').slice(0, 16) }}</span>
      </footer>
    </div>

    <div v-else class="py-20 text-center text-gray-400">
      <el-empty :description="loadFailed ? '内容不存在或已被删除' : '加载中...'" />
      <el-button v-if="loadFailed" class="mt-2" @click="goBack">返回列表</el-button>
    </div>

    <ImageLightbox
      v-model:visible="lightboxVisible"
      :images="lightboxImages"
      v-model:image-index="lightboxIndex"
    />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import ImageLightbox from '../components/ImageLightbox.vue'
import PublishButton from '../components/PublishButton.vue'
import { milestoneApi } from '../api'
import { categoryBadge, categoryLabel } from '../utils/helpers'

const route = useRoute()
const router = useRouter()
const item = ref(null)
const loadFailed = ref(false)
const lightboxVisible = ref(false)
const lightboxIndex = ref(0)
const lightboxImages = computed(() =>
  (item.value?.media || [])
    .filter((m) => m.mimeType?.startsWith('image/'))
    .map((m) => ({ url: m.url, alt: m.fileName })),
)

async function load() {
  try {
    item.value = await milestoneApi.get(route.params.id)
  } catch {
    loadFailed.value = true
  }
}
onMounted(load)

function goBack() {
  router.push({ path: '/milestones', query: { ...route.query } })
}

function edit() {
  router.push({ path: '/milestones', query: { edit: route.params.id } })
}

async function remove() {
  try {
    await ElMessageBox.confirm('确定删除这条大事记？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await milestoneApi.remove(route.params.id)
    ElMessage.success('已删除')
    router.push('/milestones')
  } catch { /* 拦截器已提示 */ }
}

function openMediaImage(m) {
  const i = lightboxImages.value.findIndex((x) => x.url === m.url)
  lightboxIndex.value = i >= 0 ? i : 0
  lightboxVisible.value = true
}
</script>
