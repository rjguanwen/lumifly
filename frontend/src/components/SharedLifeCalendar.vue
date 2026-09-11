<template>
  <div>
    <div class="flex items-center justify-between gap-4 flex-wrap mb-3">
      <div class="flex items-center gap-3 text-xs text-gray-500 flex-wrap">
        <span class="flex items-center gap-1.5"><i class="w-3 h-3 rounded-sm bg-gray-200 inline-block" />已度过</span>
        <span class="flex items-center gap-1.5"><i class="w-3 h-3 rounded-sm bg-blue-300 inline-block" />有记录</span>
        <span class="flex items-center gap-1.5"><i class="w-3 h-3 rounded-sm bg-amber-400 inline-block" />有大事记</span>
        <span class="flex items-center gap-1.5"><i class="w-3 h-3 rounded-sm bg-gray-100 inline-block" />未来</span>
      </div>
      <span class="text-xs text-gray-400">每格 = 1 周 · 每列 = 1 年（点击格子查看大事记与日常记录）</span>
    </div>

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
      <div v-if="yearTicks.length" class="life-calendar-axis mt-1.5" :style="tickAxisStyle">
        <span
          v-for="tk in yearTicks"
          :key="tk.key"
          class="life-calendar-year"
          :style="{ gridColumn: `${tk.startCol} / span ${tk.span}` }"
        >{{ tk.text }}</span>
      </div>
    </div>

    <!-- 周详情抽屉 -->
    <el-drawer v-model="showDrawer" size="480px" :title="'TA 的日历 · 第' + (current?.year ?? '') + '年'" class="life-drawer">
      <div v-if="current" class="px-1 space-y-5">
        <div class="bg-gray-50 rounded-lg p-4">
          <div class="text-lg font-semibold text-gray-900">第 {{ current.year }} 年 · {{ current.weekLabel }}</div>
          <div class="text-sm text-gray-500 mt-1">{{ current.startDate }} ~ {{ current.endDate }}</div>
          <div v-if="loadingPeriod" class="text-xs text-gray-400 mt-1">加载中…</div>
        </div>

        <div v-if="!loadingPeriod">
          <!-- 大事记 -->
          <div v-if="periodMilestones.length">
            <div class="text-sm font-medium text-gray-500 mb-2">大事记（{{ periodMilestones.length }}）</div>
            <div v-for="m in periodMilestones" :key="m.id" class="p-3 bg-amber-50 rounded-xl mb-2 border border-amber-100">
              <div class="flex items-center justify-between mb-1">
                <span class="text-sm font-medium text-amber-800">{{ m.title }}</span>
                <el-tag size="small" type="warning" effect="plain">{{ categoryLabel(m.category) }}</el-tag>
              </div>
              <div class="text-xs text-gray-500 mb-1">{{ m.eventDate }}<span v-if="m.importance"> · {{ '★'.repeat(m.importance) }}</span></div>
              <div v-if="m.description" class="text-sm text-gray-700 prose prose-sm mt-1" v-html="m.description" />
              <div v-if="m.media?.length" class="mt-2 space-y-2">
                <template v-for="med in m.media" :key="med.id">
                  <img v-if="med.mimeType?.startsWith('image/')" :src="med.url" alt="" loading="lazy" class="w-full rounded-lg object-cover max-h-60 cursor-pointer" @click.stop="openGallery(m.media, med.url)" />
                  <video v-else-if="med.mimeType?.startsWith('video/')" :src="med.url" controls class="w-full rounded-lg max-h-60" />
                </template>
              </div>
              <div class="flex justify-end mt-2">
                <el-button size="small" text type="primary" @click="openComments(m.title, 'milestone', m.id)">
                  <el-icon class="mr-0.5"><ChatLineSquare /></el-icon>私密评价
                </el-button>
              </div>
            </div>
          </div>

          <!-- 日常记录 -->
          <div>
            <div class="text-sm font-medium text-gray-500 mb-2">日常记录（{{ periodRecords.length }}）</div>
            <div v-if="periodRecords.length" class="space-y-2">
              <div v-for="r in periodRecords" :key="r.id" class="bg-blue-50 border border-blue-100 rounded-xl px-3 py-2.5">
                <div class="flex items-center gap-2">
                  <span v-if="moodEmoji(r.mood)" class="text-lg leading-none">{{ moodEmoji(r.mood) }}</span>
                  <div class="flex-1 min-w-0">
                    <div class="text-sm font-medium text-gray-800 truncate">{{ r.title || '（无标题随记）' }}</div>
                    <div class="text-xs text-gray-400">{{ r.recordDate }}</div>
                  </div>
                  <el-button size="small" text type="primary" @click="openComments(r.title || '随记', 'record', r.id)">
                    <el-icon class="mr-0.5"><ChatLineSquare /></el-icon>评价
                  </el-button>
                  <el-button size="small" type="primary" plain @click="viewRecord(r)">查看</el-button>
                </div>
              </div>
            </div>
            <div v-else class="bg-gray-50 rounded-xl p-5 text-center text-sm text-gray-400">该周暂无随记</div>
          </div>
        </div>
      </div>
    </el-drawer>

    <!-- 记录查看 -->
    <el-dialog v-model="showRecordDetail" width="760px" top="6vh" destroy-on-close title="随记内容">
      <div v-if="detailRecord" class="space-y-4">
        <div class="flex items-center gap-2 text-sm text-gray-500">
          <span class="font-semibold text-gray-700">{{ detailRecord.recordDate }}</span>
          <span v-if="detailRecord.mood" class="text-base">{{ moodEmoji(detailRecord.mood) }}</span>
        </div>
        <h3 v-if="detailRecord.title" class="text-xl font-bold text-gray-900">{{ detailRecord.title }}</h3>
        <div v-if="detailRecord.content" class="prose max-w-none" v-html="detailRecord.content" />
        <div v-if="detailRecord.media?.length" class="grid grid-cols-2 gap-3">
          <template v-for="med in detailRecord.media" :key="med.id">
            <img v-if="med.mimeType?.startsWith('image/')" :src="med.url" alt="" loading="lazy" class="w-full rounded-xl object-cover aspect-square cursor-pointer" @click="openGallery(detailRecord.media, med.url)" />
            <video v-else-if="med.mimeType?.startsWith('video/')" :src="med.url" controls class="w-full rounded-xl max-h-56" />
          </template>
        </div>
        <div class="flex justify-end">
          <el-button type="primary" plain @click="openComments(detailRecord.title || '随记', 'record', detailRecord.id)">私密评价</el-button>
        </div>
      </div>
    </el-dialog>

    <!-- 评论 -->
    <CalendarCommentDialog
      v-model:visible="commentVisible"
      :owner-id="ownerId"
      :target-type="commentType"
      :target-id="commentTarget"
      :title="commentTitle"
    />

    <ImageLightbox v-model:visible="lightboxVisible" :images="lightboxImages" v-model:image-index="lightboxIndex" />
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import dayjs from 'dayjs'
import ImageLightbox from './ImageLightbox.vue'
import CalendarCommentDialog from './CalendarCommentDialog.vue'
import { sharedCalendarApi } from '../api'
import { categoryLabel, moodEmoji } from '../utils/helpers'
import {
  buildWeekBounds, firstCellReachable, lastCellStartingBefore,
} from '../utils/lifeCalendarCells'

const props = defineProps({
  ownerId: { type: Number, required: true },
  birthDate: { type: String, required: true },
  lifespan: { type: Number, default: 80 },
  summaryData: { type: Object, default: null },
})

const cellSize = 11
const gridGap = 2
const gridStyle = {
  display: 'grid',
  gridAutoFlow: 'column',
  gridAutoColumns: `${cellSize}px`,
  gridTemplateRows: `repeat(52, ${cellSize}px)`,
  gap: `${gridGap}px`,
}

const lifeYears = computed(() => Math.max(1, Math.min(120, props.lifespan || 80)))
const birthYearVal = computed(() => {
  const b = dayjs(props.birthDate)
  return b.isValid() ? b.year() : new Date().getFullYear()
})
const yearTicks = computed(() => {
  const cols = lifeYears.value
  const ticks = []
  for (let yi = 0; yi < cols; yi += 10) {
    ticks.push({ key: yi, startCol: yi + 1, span: Math.min(10, cols - yi), text: String(birthYearVal.value + yi) })
  }
  return ticks
})
const tickAxisStyle = { display: 'grid', gridTemplateColumns: `repeat(${lifeYears.value}, ${cellSize}px)`, gap: `${gridGap}px` }

const recordShades = ['bg-blue-200', 'bg-blue-300', 'bg-blue-400', 'bg-blue-500', 'bg-blue-600']

// 性能：同 LifeCalendar，把每格重扫全部数据改为「边界预算一次 + 二分归格」，
// 命中口径与颜色/文案优先级与改动前逐格一致（见 utils/lifeCalendarCells.js）。
const metaGrid = computed(() => {
  const birth = dayjs(props.birthDate)
  const now = dayjs()
  const s = props.summaryData || {}
  const records = []
  for (const r of s.records || []) {
    const d = dayjs(r.record_date)
    if (d.isValid()) records.push({ d, n: Number(r.count) || 1 })
  }
  const milestones = []
  for (const m of s.milestones || []) {
    const d = dayjs(m.event_date)
    if (d.isValid()) milestones.push({ d, item: m })
  }

  const cellCount = lifeYears.value * 52
  const bound = buildWeekBounds(birth, cellCount)
  const birthRaw = birth.valueOf()
  const nowRaw = now.valueOf()
  const recTotals = new Float64Array(cellCount)
  const hasMsFlags = new Uint8Array(cellCount)

  if (birth.isValid()) {
    for (const r of records) {
      const lo = firstCellReachable(bound, r.d.startOf('day').valueOf())
      const hi = lastCellStartingBefore(bound, r.d.endOf('day').valueOf())
      for (let p = lo; p <= hi; p++) recTotals[p] += r.n
    }
    for (const m of milestones) {
      const lo = firstCellReachable(bound, m.d.startOf('day').valueOf())
      const hi = lastCellStartingBefore(bound, m.d.endOf('day').valueOf())
      for (let p = lo; p <= hi; p++) hasMsFlags[p] = 1
    }
  } else {
    // birthDate 非法时，原始的取反比较式会让每条数据都命中每一格，这里保持同样结果
    let total = 0
    for (const r of records) total += r.n
    recTotals.fill(total)
    if (milestones.length) hasMsFlags.fill(1)
  }

  const rows = []
  for (let yi = 0; yi < lifeYears.value; yi++) {
    const row = []
    for (let wi = 0; wi < 52; wi++) {
      const p = yi * 52 + wi
      const recordCount = recTotals[p]
      const hasMilestone = hasMsFlags[p] === 1

      // 保持 !(a < b) 的取反形式，对应原来的 !weekEnd.isBefore(birth, 'day')
      const born = !(bound.weEndDay[p] < birthRaw)
      // 这一条原本就没有取反（weekEnd.isBefore(now, 'day')），所以直接用 <
      const isPast = bound.weEndDay[p] < nowRaw

      let cls
      if (!born) cls = 'bg-transparent'
      else if (isPast) {
        if (hasMilestone) cls = 'bg-amber-400'
        else if (recordCount > 0) cls = recordShades[Math.min(recordCount, 5) - 1]
        else cls = 'bg-gray-200'
      } else cls = 'bg-gray-100'

      const bits = [`出生第 ${yi + 1} 年 · 第 ${wi + 1} 周`, `${bound.startStr[p]} ~ ${bound.endStr[p]}`]
      if (recordCount > 0) bits.push(`日常记录 ${recordCount} 条`)
      if (hasMilestone) bits.push('有大事记')
      row.push({
        key: `${yi}-${wi}`, year: yi + 1, weekLabel: `第 ${wi + 1} 周`,
        startDate: bound.startStr[p], endDate: bound.endStr[p],
        cls, title: bits.join('\n'),
      })
    }
    rows.push(row)
  }
  return rows
})

const showDrawer = ref(false)
const current = ref(null)
const loadingPeriod = ref(false)
const periodRecords = ref([])
const periodMilestones = ref([])

async function openCell(cell) {
  current.value = cell
  showDrawer.value = true
  loadingPeriod.value = true
  try {
    const data = await sharedCalendarApi.period(props.ownerId, cell.startDate, cell.endDate)
    periodRecords.value = data.records || []
    periodMilestones.value = data.milestones || []
  } catch {
    periodRecords.value = []
    periodMilestones.value = []
  } finally {
    loadingPeriod.value = false
  }
}

const showRecordDetail = ref(false)
const detailRecord = ref(null)
function viewRecord(r) {
  detailRecord.value = r
  showRecordDetail.value = true
}

const commentVisible = ref(false)
const commentType = ref('record')
const commentTarget = ref(0)
const commentTitle = ref('')
function openComments(title, type, id) {
  commentTitle.value = title || ''
  commentType.value = type
  commentTarget.value = id
  commentVisible.value = true
}

const lightboxVisible = ref(false)
const lightboxIndex = ref(0)
const lightboxImages = ref([])
function openGallery(mediaList, url) {
  const imgs = (mediaList || []).filter((med) => med.mimeType?.startsWith('image/')).map((med) => ({ url: med.url, alt: med.fileName }))
  if (!imgs.length) return
  lightboxImages.value = imgs
  const i = imgs.findIndex((x) => x.url === url)
  lightboxIndex.value = i >= 0 ? i : 0
  lightboxVisible.value = true
}
</script>
