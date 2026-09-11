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
      <span class="text-xs text-gray-400">每格 = 1 周 · 每列 = 1 年（点击格子查看详情）</span>
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
      <!-- 底部年份刻度：每 10 年标注实际年份 -->
      <div v-if="yearTicks.length" class="life-calendar-axis mt-1.5" :style="tickAxisStyle">
        <span
          v-for="tk in yearTicks"
          :key="tk.key"
          class="life-calendar-year"
          :style="{ gridColumn: `${tk.startCol} / span ${tk.span}` }"
        >{{ tk.text }}</span>
      </div>
    </div>

    <!-- 时段详情 -->
    <el-drawer v-model="showDetail" size="460px" :title="'时段详情 · 第' + (selectedCell?.year ?? '') + '年'" class="life-drawer">
      <div v-if="selectedCell" class="px-1 space-y-5">
        <div class="bg-gray-50 rounded-lg p-4">
          <div class="text-lg font-semibold text-gray-900">
            第 {{ selectedCell.year }} 年 · {{ selectedCell.weekLabel }}
          </div>
          <div class="text-sm text-gray-500 mt-1">{{ selectedCell.startDate }} ~ {{ selectedCell.endDate }}</div>
        </div>

        <!-- 常驻操作：无论该周是否已有内容均可直接新增 -->
        <div class="flex gap-3">
          <el-button type="primary" class="flex-1" @click="openRecordForm">
            <el-icon class="mr-1"><EditPen /></el-icon>写随记
          </el-button>
          <el-button type="warning" plain class="flex-1" @click="openMilestoneForm">
            <el-icon class="mr-1"><Trophy /></el-icon>记大事
          </el-button>
        </div>

        <!-- 大事记 -->
        <div v-if="selectedCell.data.milestones.length">
          <div class="text-sm font-medium text-gray-500 mb-2 flex items-center gap-1.5">
            <i class="w-2 h-2 rounded-full bg-amber-400 inline-block" />大事记
            <span class="text-xs text-gray-400 font-normal">（{{ selectedCell.data.milestones.length }} 件）</span>
          </div>
          <div v-for="m in selectedCell.data.milestones" :key="m.id" class="p-3 bg-amber-50 rounded-xl mb-2 border border-amber-100">
            <div class="flex items-center justify-between mb-1">
              <span class="text-sm font-medium text-amber-800">{{ m.title }}</span>
              <el-tag size="small" type="warning" effect="plain">{{ categoryLabel(m.category) }}</el-tag>
            </div>
            <div class="text-xs text-gray-500 mb-1">{{ m.eventDate }}<span v-if="m.importance"> · {{ '★'.repeat(m.importance) }}</span></div>
            <div v-if="m.description" class="text-sm text-gray-700 prose prose-sm mt-1" v-html="m.description" />
            <div v-if="m.media?.length" class="mt-2 space-y-2">
              <template v-for="med in m.media" :key="med.id">
                <img v-if="med.mimeType?.startsWith('image/')" :src="med.url" :alt="med.fileName" loading="lazy" class="w-full rounded-lg object-cover max-h-60 cursor-pointer hover:opacity-90 transition-opacity" @click.stop="openGallery(m.media, med.url)" />
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

        <!-- 日常记录：按条展示标题 -->
        <div>
          <div class="text-sm font-medium text-gray-500 mb-2 flex items-center gap-1.5">
            <i class="w-2 h-2 rounded-full bg-blue-300 inline-block" />日常记录
            <span v-if="cellRecordItems.length" class="text-xs text-gray-400 font-normal">（{{ cellRecordItems.length }} 条）</span>
          </div>

          <div v-if="cellLoadingRecords" class="text-center py-6 text-gray-400 text-sm">加载中…</div>

          <div v-else-if="cellRecordItems.length" class="space-y-2">
            <div v-for="r in cellRecordItems" :key="r.id" class="flex items-center gap-2 bg-blue-50 border border-blue-100 rounded-xl px-3 py-2.5">
              <span v-if="moodEmoji(r.mood)" class="text-lg leading-none shrink-0">{{ moodEmoji(r.mood) }}</span>
              <div class="flex-1 min-w-0">
                <div class="text-sm font-medium text-gray-800 truncate">{{ r.title || '（无标题随记）' }}</div>
                <div class="text-xs text-gray-400 mt-0.5">{{ r.recordDate }}</div>
              </div>
              <el-button size="small" text type="primary" @click="openComments(r.title || '随记', 'record', r.id)">
                <el-icon class="mr-0.5"><ChatLineSquare /></el-icon>评价
              </el-button>
              <el-button size="small" type="primary" plain @click="viewRecordItem(r)">查看记录</el-button>
            </div>
          </div>

          <div v-else class="bg-gray-50 rounded-xl p-5 text-center">
            <p class="text-sm text-gray-400">该周暂无随记</p>
          </div>
        </div>
      </div>
    </el-drawer>

    <!-- 记录详情弹框 -->
    <el-dialog v-model="showRecordDetail" width="760px" top="5vh" destroy-on-close title="随记详情">
      <div v-if="detailRecord" class="space-y-4">
        <div class="flex flex-wrap items-center gap-2 text-sm text-gray-500">
          <span class="font-semibold text-gray-700">{{ detailRecord.recordDate }}</span>
          <span v-if="detailRecord.mood" class="text-base">{{ moodEmoji(detailRecord.mood) }}</span>
          <el-tag v-for="tag in detailRecord.tags || []" :key="tag.id" size="small" type="info" effect="plain">{{ tag.name }}</el-tag>
        </div>
        <h3 v-if="detailRecord.title" class="text-xl font-bold text-gray-900">{{ detailRecord.title }}</h3>
        <div v-if="detailRecord.content" class="prose max-w-none" v-html="detailRecord.content" />
        <div v-else class="text-sm text-gray-400">（无正文内容）</div>
        <div v-if="detailRecord.media?.length" class="grid grid-cols-2 sm:grid-cols-3 gap-3">
          <template v-for="med in detailRecord.media" :key="med.id">
            <img
              v-if="med.mimeType?.startsWith('image/')"
              :src="med.url"
              :alt="med.fileName"
              loading="lazy"
              class="w-full rounded-xl object-cover aspect-square cursor-pointer hover:opacity-90 transition-opacity"
              @click="openGallery(detailRecord.media, med.url)"
            />
            <video v-else-if="med.mimeType?.startsWith('video/')" :src="med.url" controls class="w-full rounded-xl max-h-56" />
          </template>
        </div>
        <div class="flex justify-end pt-1">
          <el-button type="primary" plain @click="openComments(detailRecord.title || '随记', 'record', detailRecord.id)">
            <el-icon class="mr-1"><ChatLineSquare /></el-icon>私密评价
          </el-button>
        </div>
      </div>
    </el-dialog>

    <!-- 写随记弹框 -->
    <el-dialog v-model="showRecordForm" width="920px" top="5vh" destroy-on-close class="editor-dialog">
      <template #header>
        <div class="flex items-center gap-2.5">
          <div class="w-9 h-9 rounded-xl flex items-center justify-center bg-blue-50 text-blue-600">
            <el-icon :size="18"><Notebook /></el-icon>
          </div>
          <div>
            <div class="text-base font-semibold text-gray-800">写一篇新随记</div>
            <div class="text-xs text-gray-400 font-normal">所见即所得编辑，支持标题、加粗、图片与链接</div>
          </div>
        </div>
      </template>
      <el-form label-position="top">
        <div class="grid grid-cols-2 gap-4">
          <el-form-item label="日期" required>
            <el-date-picker v-model="recordForm.recordDate" type="date" class="w-full" value-format="YYYY-MM-DD" />
          </el-form-item>
          <el-form-item label="心情">
            <MoodPicker v-model="recordForm.mood" />
          </el-form-item>
        </div>
        <el-form-item label="标题（可选）">
          <el-input v-model="recordForm.title" placeholder="给这一天起个标题..." />
        </el-form-item>
        <el-form-item label="内容">
          <RichTextEditor v-model="recordForm.content" placeholder="记录这段时间的所见所闻，支持标题、加粗、列表等格式..." />
        </el-form-item>
        <el-form-item label="标签">
          <TagInput v-model="recordForm.tagIds" />
        </el-form-item>
        <el-form-item label="附件">
          <MediaUploader v-model="recordForm.mediaIds" :initial-items="recordForm.mediaObjects" multiple />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showRecordForm = false">取消</el-button>
        <el-button type="primary" :loading="savingRecord" @click="saveRecordFromCalendar">保存随记</el-button>
      </template>
    </el-dialog>

    <!-- 记大事弹框 -->
    <el-dialog v-model="showMilestoneForm" width="920px" top="5vh" destroy-on-close class="editor-dialog">
      <template #header>
        <div class="flex items-center gap-2.5">
          <div class="w-9 h-9 rounded-xl flex items-center justify-center bg-amber-50 text-amber-500">
            <el-icon :size="18"><Trophy /></el-icon>
          </div>
          <div>
            <div class="text-base font-semibold text-gray-800">记录一件人生大事</div>
            <div class="text-xs text-gray-400 font-normal">所见即所得编辑，支持标题、加粗、图片与链接</div>
          </div>
        </div>
      </template>
      <el-form label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="milestoneForm.title" placeholder="事件名称" />
        </el-form-item>
        <div class="grid grid-cols-2 gap-4">
          <el-form-item label="日期" required>
            <el-date-picker v-model="milestoneForm.eventDate" type="date" class="w-full" value-format="YYYY-MM-DD" />
          </el-form-item>
          <el-form-item label="分类">
            <el-select v-model="milestoneForm.category" class="w-full">
              <el-option v-for="cat in milestoneCategories" :key="cat.value" :label="cat.label" :value="cat.value" />
            </el-select>
          </el-form-item>
        </div>
        <el-form-item label="重要程度">
          <div class="flex gap-1 items-center">
            <button v-for="i in 5" :key="i" type="button" class="text-2xl transition-transform hover:scale-110" @click="milestoneForm.importance = i">
              <span :class="i <= milestoneForm.importance ? 'text-amber-400' : 'text-gray-200'">★</span>
            </button>
            <span class="text-xs text-gray-400 ml-2">{{ milestoneImportanceLabel }}</span>
          </div>
        </el-form-item>
        <el-form-item label="描述">
          <RichTextEditor v-model="milestoneForm.description" placeholder="详细描述这个重要事件，支持标题、加粗、列表等格式..." />
        </el-form-item>
        <el-form-item label="附件">
          <MediaUploader v-model="milestoneForm.mediaIds" :initial-items="milestoneForm.mediaObjects" multiple />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showMilestoneForm = false">取消</el-button>
        <el-button type="primary" :loading="savingMilestone" @click="saveMilestoneFromCalendar">保存大事记</el-button>
      </template>
    </el-dialog>

    <ImageLightbox
      v-model:visible="lightboxVisible"
      :images="lightboxImages"
      v-model:image-index="lightboxIndex"
    />

    <!-- 双方可见的私密评价 -->
    <CalendarCommentDialog
      v-model:visible="commentVisible"
      :owner-id="auth.user?.id"
      :target-type="commentType"
      :target-id="commentTarget"
      :title="commentTitle"
    />
  </div>
</template>

<script setup>
import { computed, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import dayjs from 'dayjs'
import ImageLightbox from './ImageLightbox.vue'
import RichTextEditor from './RichTextEditor.vue'
import MediaUploader from './MediaUploader.vue'
import TagInput from './TagInput.vue'
import MoodPicker from './MoodPicker.vue'
import CalendarCommentDialog from './CalendarCommentDialog.vue'
import { recordApi, milestoneApi } from '../api'
import { useAuthStore } from '../stores/auth'
import { categoryLabel, milestoneCategories, moodEmoji } from '../utils/helpers'
import {
  buildWeekBounds, firstCellReachable, lastCellStartingBefore,
} from '../utils/lifeCalendarCells'

const auth = useAuthStore()

const props = defineProps({
  birthDate: { type: String, required: true },
  lifespan: { type: Number, default: 80 },
  summaryData: { type: Object, default: null },
  compact: { type: Boolean, default: false },
})
const emit = defineEmits(['changed'])

// ---------- 静态色阶（字符串须保持字面量，便于 Tailwind 扫描） ----------
const recordShades = ['bg-blue-200', 'bg-blue-300', 'bg-blue-400', 'bg-blue-500', 'bg-blue-600']
const emptyCls = 'bg-transparent'
const pastCls = 'bg-gray-200'
const milestoneCls = 'bg-amber-400'
const currentCls = 'bg-green-500'
const futureCls = 'bg-gray-100'

const cellSize = computed(() => (props.compact ? 6 : 11))
const gridGap = computed(() => (props.compact ? 1 : 2))

// 竖排布局：固定 52 行（一年 52 周），年份作为列从左到右排列。
// 模板保持「先年、后周」的嵌套顺序，借助 grid-auto-flow: column 自动逐列填充。
const gridStyle = computed(() => ({
  display: 'grid',
  gridAutoFlow: 'column',
  gridAutoColumns: `${cellSize.value}px`,
  gridTemplateRows: `repeat(52, ${cellSize.value}px)`,
  gap: `${gridGap.value}px`,
}))

// 生命总年数（受限 1~120）与出生年份，供底部年份刻度使用
const lifeYears = computed(() => Math.max(1, Math.min(120, props.lifespan || 80)))
const birthYearVal = computed(() => {
  const b = dayjs(props.birthDate)
  return b.isValid() ? b.year() : new Date().getFullYear()
})

// 底部刻度：每 10 年标注一次实际年份
const yearTicks = computed(() => {
  const cols = lifeYears.value
  const ticks = []
  for (let yi = 0; yi < cols; yi += 10) {
    ticks.push({
      key: yi,
      startCol: yi + 1,
      span: Math.min(10, cols - yi),
      text: String(birthYearVal.value + yi),
    })
  }
  return ticks
})

const tickAxisStyle = computed(() => ({
  display: 'grid',
  gridTemplateColumns: `repeat(${lifeYears.value}, ${cellSize.value}px)`,
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
// 性能：每格的起止边界只算一次，数据用二分归入命中的格子，不再每格重扫全部
// 记录/大事记（原为 O(格数 × 数据量)）。命中口径、颜色优先级、提示文案与
// 改动前逐格完全一致，详见 utils/lifeCalendarCells.js 顶部说明。
const metaGrid = computed(() => {
  const birth = dayjs(props.birthDate)
  const now = dayjs()
  const lifespan = Math.max(1, Math.min(120, props.lifespan || 80))
  const { records, milestones } = parseSummary()

  const cellCount = lifespan * 52
  const bound = buildWeekBounds(birth, cellCount)
  const birthRaw = birth.valueOf()
  const nowRaw = now.valueOf()
  const recTotals = new Float64Array(cellCount)
  const msLists = new Array(cellCount) // 缺省（undefined）表示该格无大事记

  if (birth.isValid()) {
    // 记录条数按原顺序累加，大事记按原数组顺序入格，两者结果与逐格重扫相同
    for (const r of records) {
      const lo = firstCellReachable(bound, r.d.startOf('day').valueOf())
      const hi = lastCellStartingBefore(bound, r.d.endOf('day').valueOf())
      for (let p = lo; p <= hi; p++) recTotals[p] += r.n
    }
    for (const m of milestones) {
      const lo = firstCellReachable(bound, m.d.startOf('day').valueOf())
      const hi = lastCellStartingBefore(bound, m.d.endOf('day').valueOf())
      for (let p = lo; p <= hi; p++) {
        if (!msLists[p]) msLists[p] = []
        msLists[p].push(m.item)
      }
    }
  } else {
    // birthDate 非法时，原始的取反比较式会让每条数据都命中每一格，这里保持同样结果
    let total = 0
    for (const r of records) total += r.n
    recTotals.fill(total)
    const allMs = milestones.map((m) => m.item)
    for (let p = 0; p < cellCount; p++) msLists[p] = allMs
  }

  const rows = []
  for (let yi = 0; yi < lifespan; yi++) {
    const row = []
    for (let wi = 0; wi < 52; wi++) {
      const p = yi * 52 + wi
      const recordCount = recTotals[p]
      const cellMilestones = msLists[p] || []

      // 下面三个判定保持 !(a < b) 的取反形式，对应原来的
      // !weekEnd.isBefore(birth) / !weekStart.isAfter(now) && !weekEnd.isBefore(now)
      const isCurrent = !(nowRaw < bound.wsStartDay[p]) && !(bound.weEndDay[p] < nowRaw)
      const isPast = !isCurrent && bound.weEndDay[p] < nowRaw
      const born = !(bound.weEndDay[p] < birthRaw)

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

      const bits = [`出生第 ${yi + 1} 年 · 第 ${wi + 1} 周`, `${bound.startStr[p]} ~ ${bound.endStr[p]}`]
      if (recordCount > 0) bits.push(`日常记录 ${recordCount} 条`)
      if (cellMilestones.length) bits.push(`大事记 ${cellMilestones.length} 件`)

      row.push({
        key: `${yi}-${wi}`,
        yi,
        wi,
        year: yi + 1,
        weekLabel: `第 ${wi + 1} 周`,
        startDate: bound.startStr[p],
        endDate: bound.endStr[p],
        cls,
        title: bits.join('\n'),
        data: { milestones: cellMilestones, recordCount },
      })
    }
    rows.push(row)
  }
  return rows
})

// ---------- 抽屉 ----------
const showDetail = ref(false)
const selectedCell = ref(null)
const cellRecordItems = ref([])
const cellLoadingRecords = ref(false)

function todayISO() {
  return dayjs().format('YYYY-MM-DD')
}

async function openCell(cell) {
  selectedCell.value = cell
  showDetail.value = true
  await loadCellRecords(cell)
}

async function loadCellRecords(cell) {
  cellLoadingRecords.value = true
  try {
    const data = await recordApi.list({ from: cell.startDate, to: cell.endDate, limit: 100 })
    cellRecordItems.value = data.items || []
  } catch {
    cellRecordItems.value = []
  } finally {
    cellLoadingRecords.value = false
  }
}

// 抽屉内容同步（保存后更新当前格子的记录/大事展示）
async function refreshSelectedCell() {
  if (!showDetail.value || !selectedCell.value) return
  await loadCellRecords(selectedCell.value)
  // 从最新 metaGrid 中按 key 取新 cell，更新大事列表
  const key = selectedCell.value.key
  for (const row of metaGrid.value) {
    for (const cell of row) {
      if (cell.key === key) {
        selectedCell.value = cell
        return
      }
    }
  }
}

// ---------- 记录详情弹框 ----------
const showRecordDetail = ref(false)
const detailRecord = ref(null)

function viewRecordItem(r) {
  detailRecord.value = r
  showRecordDetail.value = true
}

// ---------- 写随记弹框 ----------
const showRecordForm = ref(false)
const savingRecord = ref(false)
const recordForm = reactive({
  title: '',
  content: '',
  recordDate: '',
  mood: null,
  weather: '',
  tagIds: [],
  mediaIds: [],
  mediaObjects: [],
})

function openRecordForm() {
  Object.assign(recordForm, {
    title: '',
    content: '',
    recordDate: selectedCell.value?.startDate || todayISO(),
    mood: null,
    weather: '',
    tagIds: [],
    mediaIds: [],
    mediaObjects: [],
  })
  showRecordForm.value = true
}

async function saveRecordFromCalendar() {
  if (!recordForm.recordDate) {
    ElMessage.warning('请选择日期')
    return
  }
  savingRecord.value = true
  try {
    const created = await recordApi.create({ ...recordForm })
    ElMessage.success('随记已保存')
    showRecordForm.value = false
    if (selectedCell.value && created.recordDate >= selectedCell.value.startDate && created.recordDate <= selectedCell.value.endDate) {
      cellRecordItems.value.push(created)
      cellRecordItems.value.sort((a, b) => (a.recordDate < b.recordDate ? 1 : -1))
    }
    emit('changed') // 通知父页面刷新日历汇总（格子颜色等）
  } catch {
    /* 拦截器已提示 */
  } finally {
    savingRecord.value = false
  }
}

// ---------- 记大事弹框 ----------
const showMilestoneForm = ref(false)
const savingMilestone = ref(false)
const milestoneForm = reactive({
  title: '',
  description: '',
  eventDate: '',
  category: 'other',
  importance: 3,
  mediaIds: [],
  mediaObjects: [],
})

const milestoneImportanceLabel = computed(() => ['', '普通', '值得记住', '重要', '非常重要', '人生大事'][milestoneForm.importance] || '')

function openMilestoneForm() {
  Object.assign(milestoneForm, {
    title: '',
    description: '',
    eventDate: selectedCell.value?.startDate || todayISO(),
    category: 'other',
    importance: 3,
    mediaIds: [],
    mediaObjects: [],
  })
  showMilestoneForm.value = true
}

async function saveMilestoneFromCalendar() {
  if (!milestoneForm.title || !milestoneForm.eventDate) {
    ElMessage.warning('请填写标题和日期')
    return
  }
  savingMilestone.value = true
  try {
    const created = await milestoneApi.create({ ...milestoneForm })
    ElMessage.success('大事记已保存')
    showMilestoneForm.value = false
    if (selectedCell.value && created.eventDate >= selectedCell.value.startDate && created.eventDate <= selectedCell.value.endDate) {
      selectedCell.value.data.milestones.push(created)
    }
    emit('changed')
  } catch {
    /* 拦截器已提示 */
  } finally {
    savingMilestone.value = false
  }
}

// ---------- 私密评价 ----------
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

// ---------- 图片灯箱 ----------
const lightboxVisible = ref(false)
const lightboxIndex = ref(0)
const lightboxImages = ref([])

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
</script>
