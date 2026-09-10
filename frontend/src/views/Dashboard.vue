<template>
  <div class="space-y-6 pb-4">
    <!-- ============ Hero ============ -->
    <div
      v-if="auth.user?.birthDate && stats"
      class="relative rounded-2xl overflow-hidden text-white shadow-xl"
      style="background: linear-gradient(120deg, #1f3bb3 0%, #3b5bdb 45%, #5B7CFF 100%)"
    >
      <!-- 装饰光斑 -->
      <div class="absolute -top-16 -left-10 w-64 h-64 rounded-full opacity-20" style="background: radial-gradient(circle, #fff 0%, transparent 70%)" />
      <div class="absolute top-6 right-40 w-32 h-32 rounded-full opacity-10" style="background: radial-gradient(circle, #fff 0%, transparent 70%)" />
      <div class="absolute -bottom-24 -right-10 w-80 h-80 rounded-full opacity-10" style="background: radial-gradient(circle, #FFD766 0%, transparent 70%)" />

      <div class="relative flex flex-wrap items-center justify-between gap-6 p-7 sm:p-9">
        <div class="min-w-[260px]">
          <div class="flex items-center gap-2.5 mb-4">
            <BrandLogo :size="34" />
            <span class="text-white/90 text-sm tracking-wide">飞光 · Lumifly</span>
          </div>
          <p class="text-white/75 text-sm">{{ greetingText }}，{{ auth.user.displayName }}</p>
          <h1 class="text-2xl sm:text-3xl font-bold mt-1.5 leading-snug">
            人生就像一场旅行，愿你一路有光。
          </h1>
          <div class="flex flex-wrap gap-2 mt-5">
            <span class="text-xs px-3 py-1.5 rounded-full bg-white/15 backdrop-blur">出生于 {{ auth.user.birthDate }}</span>
            <span class="text-xs px-3 py-1.5 rounded-full bg-white/15 backdrop-blur">预期寿命 {{ auth.user.expectedLifespan }} 岁</span>
            <span class="text-xs px-3 py-1.5 rounded-full bg-white/15 backdrop-blur">已走过 {{ stats.percentage }}%</span>
          </div>
        </div>

        <!-- 进度环 -->
        <div class="flex items-center gap-5">
          <div class="relative w-32 h-32 shrink-0">
            <svg viewBox="0 0 140 140" class="w-full h-full" style="transform: rotate(-90deg)">
              <circle cx="70" cy="70" r="60" fill="none" stroke="rgba(255,255,255,0.18)" stroke-width="11" />
              <circle
                cx="70" cy="70" r="60" fill="none"
                stroke="#FFD766" stroke-width="11" stroke-linecap="round"
                :stroke-dasharray="`${(stats.percentage / 100) * 376.99} 376.99`"
              />
            </svg>
            <div class="absolute inset-0 flex flex-col items-center justify-center">
              <span class="text-2xl font-bold">{{ stats.percentage }}%</span>
              <span class="text-[11px] text-white/70 mt-0.5">人生进度</span>
            </div>
          </div>
          <div class="text-sm space-y-1.5 text-white/85 hidden sm:block">
            <div>已度过 <b class="text-white font-semibold">{{ stats.livedYears }}</b> 年 {{ stats.livedDays.toLocaleString() }} 天</div>
            <div>相当于 <b class="text-white font-semibold">{{ stats.livedWeeks.toLocaleString() }}</b> 周</div>
            <div>还剩约 <b class="text-white font-semibold">{{ stats.remainingYears }}</b> 年 <b class="text-white font-semibold">{{ stats.remainingWeeks.toLocaleString() }}</b> 周</div>
          </div>
        </div>
      </div>
    </div>

    <div v-else class="bg-white rounded-2xl shadow-sm p-10 text-center text-gray-400">
      请先在「设置」中填写出生日期与预期寿命，即可生成人生日历。
      <el-button type="primary" plain class="mt-4 block mx-auto" @click="$router.push('/settings')">去设置</el-button>
    </div>

    <!-- ============ 数据速览 ============ -->
    <div v-if="counts" class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div
        v-for="card in statCards"
        :key="card.label"
        class="bg-white rounded-2xl p-5 shadow-sm hover:shadow-md transition-shadow cursor-pointer border border-gray-100"
        @click="$router.push(card.to)"
      >
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-3">
            <div class="w-11 h-11 rounded-xl flex items-center justify-center" :style="{ background: card.bg, color: card.color }">
              <el-icon :size="20"><component :is="card.icon" /></el-icon>
            </div>
            <div>
              <div class="text-xl font-bold text-gray-900 leading-none">{{ card.value }}</div>
              <div class="text-xs text-gray-400 mt-1.5">{{ card.label }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- ============ 人生日历 ============ -->
    <div class="bg-white rounded-2xl shadow-sm p-6">
      <div class="flex items-center justify-between mb-4">
        <div>
          <h2 class="text-base font-semibold text-gray-800">人生日历</h2>
          <p class="text-xs text-gray-400 mt-0.5">以周为单位，一览已经历与仍可把握的时光</p>
        </div>
        <el-button text type="primary" @click="$router.push('/calendar')">全屏查看<el-icon class="ml-1"><ArrowRight /></el-icon></el-button>
      </div>
      <LifeCalendar
        v-if="auth.user?.birthDate && summaryData"
        :birth-date="auth.user.birthDate"
        :lifespan="auth.user.expectedLifespan || 80"
        :summary-data="summaryData"
        compact
        @changed="load"
      />
    </div>

    <!-- ============ 最近记录 + 快捷入口 ============ -->
    <div class="grid lg:grid-cols-3 gap-6">
      <div class="lg:col-span-2 bg-white rounded-2xl shadow-sm p-6">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-base font-semibold text-gray-800">最近的记录</h2>
          <el-button text type="primary" @click="$router.push('/records')">全部记录<el-icon class="ml-1"><ArrowRight /></el-icon></el-button>
        </div>
        <div v-if="recentRecords.length" class="space-y-3">
          <div
            v-for="record in recentRecords"
            :key="record.id"
            class="flex items-center gap-4 p-3 -mx-3 rounded-xl hover:bg-gray-50 transition-colors cursor-pointer"
            @click="goRecord(record)"
          >
            <div class="w-12 h-12 rounded-xl flex items-center justify-center text-xl bg-gradient-to-br from-brand-50 to-blue-100 shrink-0">
              {{ moodEmoji(record.mood) || '✍️' }}
            </div>
            <div class="flex-1 min-w-0">
              <div class="font-medium text-gray-800 text-sm truncate">{{ record.title || '(无标题)' }}</div>
              <div class="text-xs text-gray-400 mt-0.5 flex items-center gap-2">
                <span>{{ record.recordDate }}</span>
                <template v-for="tag in record.tags || []" :key="tag.id">
                  <span class="text-gray-300">·</span>
                  <span>{{ tag.name }}</span>
                </template>
              </div>
            </div>
            <el-icon class="text-gray-300"><ArrowRight /></el-icon>
          </div>
        </div>
        <el-empty v-else description="还没有日记，写下一篇吧" :image-size="64" class="py-6" />
      </div>

      <div class="space-y-4">
        <div
          v-for="entry in quickActions"
          :key="entry.label"
          class="bg-white rounded-2xl p-5 shadow-sm hover:shadow-md transition-shadow cursor-pointer border border-gray-100"
          @click="$router.push(entry.to)"
        >
          <div class="flex items-center gap-3">
            <div class="w-10 h-10 rounded-lg flex items-center justify-center" :style="{ background: entry.bg, color: entry.color }">
              <el-icon :size="18"><component :is="entry.icon" /></el-icon>
            </div>
            <div>
              <div class="font-medium text-gray-800 text-sm">{{ entry.label }}</div>
              <div class="text-xs text-gray-400 mt-0.5">{{ entry.desc }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import LifeCalendar from '../components/LifeCalendar.vue'
import BrandLogo from '../components/BrandLogo.vue'
import { useAuthStore } from '../stores/auth'
import { calendarApi, recordApi } from '../api'
import { getLifeStats, moodEmoji } from '../utils/helpers'

const router = useRouter()
const auth = useAuthStore()

const summaryData = ref(null)
const recentRecords = ref([])

const greetingText = computed(() => {
  const h = new Date().getHours()
  if (h < 5) return '夜深了'
  if (h < 12) return '早上好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})

const stats = computed(() => {
  if (!auth.user?.birthDate) return null
  const s = getLifeStats(auth.user.birthDate, auth.user.expectedLifespan || 80)
  const totalYears = auth.user.expectedLifespan || 80
  return {
    ...s,
    remainingYears: Math.max(0, totalYears - s.livedYears),
    percentage: s.percentage,
  }
})

const counts = computed(() => {
  const s = summaryData.value
  if (!s) return null
  return {
    records: (s.records || []).reduce((acc, r) => acc + (Number(r && r.count) || 0), 0),
    milestones: (s.milestones || []).length,
    ideas: (s.ideas || []).length,
  }
})

const statCards = computed(() => {
  if (!counts.value) return []
  return [
    { label: '日常记录', value: counts.value.records, icon: 'EditPen', to: '/records', bg: '#eef2ff', color: '#4A6CF7' },
    { label: '人生大事', value: counts.value.milestones, icon: 'Trophy', to: '/milestones', bg: '#fef3c7', color: '#d97706' },
    { label: '想法灵感', value: counts.value.ideas, icon: 'Lightning', to: '/ideas', bg: '#ecfeff', color: '#0891b2' },
  ]
})

const quickActions = [
  { label: '写日记', desc: '记录今天的生活', icon: 'EditPen', to: '/records', bg: '#eef2ff', color: '#4A6CF7' },
  { label: '记大事', desc: '标记重要时刻', icon: 'Trophy', to: '/milestones', bg: '#fef3c7', color: '#d97706' },
  { label: '记灵感', desc: '捕捉灵光一闪', icon: 'Lightning', to: '/ideas', bg: '#ecfeff', color: '#0891b2' },
]

async function load() {
  if (!auth.user?.birthDate) return
  // 两个请求互不依赖，改为并发发出，把首屏串行的两次 RTT 压成一次。
  // 用 allSettled 而非 all：保持原来「各自 try/catch，任一失败仅跳过它自己的赋值、
  // 不影响另一个」的语义（成功/失败的可观测结果与改动前完全一致）。
  const [sumRes, recRes] = await Promise.allSettled([
    calendarApi.summary(),
    recordApi.list({ limit: 4 }),
  ])
  if (sumRes.status === 'fulfilled') summaryData.value = sumRes.value
  if (recRes.status === 'fulfilled') recentRecords.value = recRes.value.items || []
}

function goRecord(record) {
  router.push({ path: '/records', query: { focus: String(record.id) } })
}

onMounted(load)
</script>
