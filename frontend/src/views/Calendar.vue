<template>
  <div class="space-y-6">
    <div class="flex items-start justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">人生日历</h1>
        <p class="text-gray-500 mt-1">你的一生，一目了然</p>
      </div>
      <el-button type="primary" plain @click="showShare = true">
        <el-icon class="mr-1"><Share /></el-icon>分享给好友
      </el-button>
    </div>

    <!-- 人生统计条 -->
    <div v-if="auth.user?.birthDate" class="flex items-center gap-6 text-sm text-gray-600">
      <span>已度过 <strong class="text-gray-900">{{ lifeStats.livedWeeks }}</strong> 周</span>
      <span>剩余 <strong class="text-gray-900">{{ lifeStats.remainingWeeks }}</strong> 周</span>
      <span>人生进度 <strong class="text-gray-900">{{ lifeStats.percentage }}%</strong></span>
    </div>

    <el-card shadow="never">
      <LifeCalendar
        v-if="auth.user?.birthDate && summaryData"
        :birth-date="auth.user.birthDate"
        :lifespan="auth.user.expectedLifespan || 80"
        :summary-data="summaryData"
        @changed="loadSummary"
      />
      <div v-else class="text-center py-8 text-gray-400">请先在设置中填写出生日期</div>
    </el-card>

    <ShareCalendarDialog v-model:visible="showShare" />
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import LifeCalendar from '../components/LifeCalendar.vue'
import ShareCalendarDialog from '../components/ShareCalendarDialog.vue'
import { useAuthStore } from '../stores/auth'
import { calendarApi } from '../api'
import { getLifeStats } from '../utils/helpers'

const auth = useAuthStore()
const summaryData = ref(null)
const showShare = ref(false)

const lifeStats = computed(() => {
  if (!auth.user?.birthDate) return { livedWeeks: 0, remainingWeeks: 0, percentage: 0 }
  return getLifeStats(auth.user.birthDate, auth.user.expectedLifespan || 80)
})

async function loadSummary() {
  try {
    summaryData.value = await calendarApi.summary()
  } catch {
    /* 忽略 */
  }
}

onMounted(loadSummary)
</script>
