<template>
  <div>
    <!-- 图例 -->
    <div class="flex items-center justify-between gap-4 flex-wrap mb-3">
      <div class="flex items-center gap-3 text-xs text-gray-500 flex-wrap">
        <span class="flex items-center gap-1.5"><i class="w-3 h-3 rounded-sm bg-gray-200 inline-block" />已度过</span>
        <span class="flex items-center gap-1.5"><i class="w-3 h-3 rounded-sm bg-blue-300 inline-block" />有记录</span>
        <span class="flex items-center gap-1.5"><i class="w-3 h-3 rounded-sm bg-amber-400 inline-block" />有大事记</span>
        <span class="flex items-center gap-1.5"><i class="w-3 h-3 rounded-sm bg-green-500 inline-block" />当前</span>
        <span class="flex items-center gap-1.5"><i class="w-3 h-3 rounded-sm bg-gray-100 inline-block" />未来</span>
      </div>
      <span class="text-xs text-gray-400">每格 = 1 周 · 每行 = 1 年（点击格子查看详情）</span>
    </div>

    <!-- 网格 -->
    <div class="bg-gray-50 rounded-xl p-3 overflow-x-auto">
      <div class="life-calendar-grid" :style="gridStyle">
        <template v-for="row in metaGrid" :key="row[0].yi">
          <div
            v-for="cell in row"
            :key="cell.key"
            class="life-calendar-cell"
            :class="cell.cls"
            :title="cell.title"
            @click="openCell(cell)"
          />
        </template>
      </div>
    </div>

    <!-- 时段详情 -->
    <el-drawer v-model="showDetail" size="430px" :title="'时段详情 · 第' + (selectedCell?.year ?? '') + '年'" class="life-drawer">
      <div v-if="selectedCell" class="px-1 space-y-5">
        <div class="bg-gray-50 rounded-lg p-4">
          <div class="text-lg font-semibold text-gray-900">
            第 {{ selectedCell.year }} 年 · {{ selectedCell.weekLabel }}
          </div>
          <div class="text-sm text-gray-500 mt-1">{{ selectedCell.startDate }} ~ {{ selectedCell.endDate }}</div>
        </div>

        <div v-if="selectedCell.data.milestones.length">
          <div class="text-sm font-medium text-gray-500 mb-2 flex items-center gap-1.5">
            <i class="w-2 h-2 rounded-full bg-amber-400 inline-block" />大事记
          </div>
          <div v-for="m in selectedCell.data.milestones" :key="m.id" class="p-3 bg-amber-50 rounded-xl mb-2 border border-amber-100">
            <div class="flex items-center justify-between mb-1">
              <span class="text-sm font-medium text-amber-800">{{ m.title }}</span>
              <span class="text-xs text-amber-600">{{ categoryLabel(m.category) }}</span>
            </div>
            <div class="text-xs text-gray-500 mb-1">{{ m.eventDate }}<span v-if="m.importance"> · {{ '★'.repeat(m.importance) }}</span></div>
            <div v-if="m.description" class="text-sm text-gray-700 prose prose-sm mt-1" v-html="m.description" />
            <div v-if="m.media?.length" class="mt-2 space-y-2">
              <template v-for="med in m.media" :key="med.id">
                <img v-if="med.mimeType?.startsWith('image/')" :src="med.url" :alt="med.fileName" class="w-full rounded-lg object-cover max-h-60 cursor-pointer hover:opacity-90 transition-opacity" @click.stop="openGallery(m.media, med.url)" />
                <video v-else-if="med.mimeType?.startsWith('video/')" :src="med.url" controls class="w-full rounded-lg max-h-60" />
              </template>
            </div>
          </div>
        </div>

        <div v-if="selectedCell.data.recordCount > 0">
          <div class="text-sm font-medium text-gray-500 mb-2 flex items-center gap-1.5">
            <i class="w-2 h-2 rounded-full bg-blue-300 inline-block" />日常记录
          </div>
          <div class="bg-blue-50 border border-blue-100 rounded-xl p-3 flex items-center justify-between">
            <span class="text-gray-700">{{ selectedCell.data.recordCount }} 条记录</span>
            <el-button size="small" type="primary" plain @click="goRecords">查看记录</el-button>
          </div>
        </div>

        <div v-if="!selectedCell.data.milestones.length && selectedCell.data.recordCount === 0" class="text-center py-6">
          <el-empty description="该时段暂无记录" :image-size="70" />
          <div class="flex gap-3 justify-center mt-2">
            <el-button size="small" @click="$router.push('/records')">写日记</el-button>
            <el-button size="small" @click="$router.push('/milestones')">记大事</el-button>
          </div>
        </div>
      </div>
    </el-drawer>

    <ImageLightbox
      v-model:visible="lightboxVisible"
      :images="lightboxImages"
      v-model:image-index="lightboxIndex"
    />
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import dayjs from 'dayjs'
import ImageLightbox from './ImageLightbox.vue'
import { categoryLabel } from '../utils/helpers'

const props = defineProps({
  birthDate: { type: String, required: true },
  lifespan: { type: Number, default: 80 },
  summaryData: { type: Object, default: null },
  compact: { type: Boolean, default: false },
})

// ---------- 静态色阶（字符串须保持字面量，便于 Tailwind 扫描） ----------
const recordShades = ['bg-blue-200', 'bg-blue-300', 'bg-blue-400', 'bg-blue-500', 'bg-blue-600']
const emptyCls = 'bg-transparent'
const pastCls = 'bg-gray-200'
const milestoneCls = 'bg-amber-400'
const currentCls = 'bg-green-500'
const futureCls = 'bg-gray-100'

const cellSize = computed(() => (props.compact ? 6 : 11))
const gridGap = computed(() => (props.compact ? 1 : 2))
const gridStyle = computed(() => ({
  display: 'grid',
  gridTemplateColumns: `repeat(52, ${cellSize.value}px)`,
  gap: `${gridGap.value}px`,
}))

// 预解析一次汇总数据，避免每格重复解析
function parseSummary() {
  const s = props.summaryData || {}
  const records = []
  for (const r of s.records || []) {
    if (!r) continue
    const d = dayjs(r.record_date || r.recordDate)
    if (d.isValid()) records.push({ d, n: Number(r.count) || 1 })
  }
  const milestones = []
  for (const m of s.milestones || []) {
    if (!m) continue
    const d = dayjs(m.eventDate)
    if (d.isValid()) milestones.push({ d, item: m })
  }
  return { records, milestones }
}

// ---------- 生成带标注的网格 ----------
const metaGrid = computed(() => {
  const birth = dayjs(props.birthDate)
  const now = dayjs()
  const lifespan = Math.max(1, Math.min(120, props.lifespan || 80))
  const { records, milestones } = parseSummary()

  const rows = []
  for (let yi = 0; yi < lifespan; yi++) {
    const row = []
    for (let wi = 0; wi < 52; wi++) {
      const weekStart = birth.add(yi * 52 + wi, 'week')
      const weekEnd = weekStart.add(6, 'day')
      const start = weekStart.format('YYYY-MM-DD')
      const end = weekEnd.format('YYYY-MM-DD')

      const inWeek = (d) => d && d.isValid() && !d.isBefore(weekStart, 'day') && !d.isAfter(weekEnd, 'day')

      let recordCount = 0
      const cellMilestones = []
      if (!weekEnd.isBefore(birth, 'day')) {
        for (const r of records) if (inWeek(r.d)) recordCount += r.n
        for (const m of milestones) if (inWeek(m.d)) cellMilestones.push(m.item)
      }

      const isCurrent = !weekStart.isAfter(now, 'day') && !weekEnd.isBefore(now, 'day')
      const isPast = !isCurrent && weekEnd.isBefore(now, 'day')
      const born = !weekEnd.isBefore(birth, 'day')

      let cls
      if (!born) cls = [emptyCls]
      else if (isCurrent) cls = [currentCls]
      else if (isPast) {
        if (cellMilestones.length) cls = [milestoneCls]
        else if (recordCount > 0) cls = [recordShades[Math.min(recordCount, 5) - 1]]
        else cls = [pastCls]
      } else {
        cls = [futureCls]
      }

      const bits = [`出生第 ${yi + 1} 年 · 第 ${wi + 1} 周`, `${start} ~ ${end}`]
      if (recordCount > 0) bits.push(`日常记录 ${recordCount} 条`)
      if (cellMilestones.length) bits.push(`大事记 ${cellMilestones.length} 件`)

      row.push({
        key: `${yi}-${wi}`,
        yi,
        wi,
        year: yi + 1,
        weekLabel: `第 ${wi + 1} 周`,
        startDate: start,
        endDate: end,
        cls,
        title: bits.join('\n'),
        data: { milestones: cellMilestones, recordCount },
      })
    }
    rows.push(row)
  }
  return rows
})

const showDetail = ref(false)
const selectedCell = ref(null)
const lightboxVisible = ref(false)
const lightboxIndex = ref(0)
const lightboxImages = ref([])

function openCell(cell) {
  selectedCell.value = cell
  showDetail.value = true
}

function openGallery(mediaList, url) {
  const imgs = (mediaList || [])
    .filter((med) => med.mimeType?.startsWith('image/'))
    .map((med) => ({ url: med.url, alt: med.fileName }))
  if (!imgs.length) return
  lightboxImages.value = imgs
  const i = imgs.findIndex((x) => x.url === url)
  lightboxIndex.value = i >= 0 ? i : 0
  lightboxVisible.value = true
}

function goRecords() {
  showDetail.value = false
  if (selectedCell.value) {
    const { startDate, endDate } = selectedCell.value
    window.location.href = `/records?from=${startDate}&to=${endDate}`
  }
}
</script>
