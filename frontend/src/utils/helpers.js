import dayjs from 'dayjs'
import isoWeek from 'dayjs/plugin/isoWeek'

dayjs.extend(isoWeek)

// ===== 心情 =====
export const moodMap = {
  happy: '😊', excited: '🤩', grateful: '🙏', neutral: '😐',
  tired: '😴', anxious: '😰', sad: '😢', angry: '😤',
}

export const moodLabel = {
  happy: '开心', excited: '兴奋', grateful: '感恩', neutral: '平静',
  tired: '疲惫', anxious: '焦虑', sad: '难过', angry: '生气',
}

export function moodEmoji(mood) {
  return moodMap[mood] || mood
}

export function moodText(mood) {
  return moodLabel[mood] || mood
}

export const moodOptions = Object.entries(moodLabel).map(([value, label]) => ({ value, label }))

// ===== 大事记分类 =====
export const milestoneCategories = [
  { value: 'education', label: '教育' },
  { value: 'career', label: '事业' },
  { value: 'relationship', label: '情感' },
  { value: 'travel', label: '旅行' },
  { value: 'achievement', label: '成就' },
  { value: 'health', label: '健康' },
  { value: 'family', label: '家庭' },
  { value: 'other', label: '其他' },
]

const categoryColors = {
  education: '#409EFF', career: '#67C23A', relationship: '#E6607A',
  travel: '#00B4D8', achievement: '#E6A23C', health: '#F56C6C',
  family: '#7E57C2', other: '#909399',
}

const categoryBadges = {
  education: 'blue', career: 'green', relationship: 'danger',
  travel: 'cyan', achievement: 'warning', health: 'danger',
  family: 'purple', other: 'info',
}

export function categoryColor(cat) {
  return categoryColors[cat] || '#909399'
}

export function categoryBadge(cat) {
  return categoryBadges[cat] || 'info'
}

export function categoryLabel(cat) {
  return milestoneCategories.find((x) => x.value === cat)?.label || cat
}

// ===== 规划 =====
export const planPriorities = [
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' },
  { value: 'urgent', label: '紧急' },
]

export const planStatusColumns = [
  { status: 'not_started', label: '未开始', color: '#909399' },
  { status: 'in_progress', label: '进行中', color: '#409EFF' },
  { status: 'completed', label: '已完成', color: '#67C23A' },
  { status: 'abandoned', label: '已放弃', color: '#F56C6C' },
]

export const planStatusOptions = planStatusColumns.map(({ status, label }) => ({ value: status, label }))

export function priorityType(p) {
  return { low: 'info', medium: '', high: 'warning', urgent: 'danger' }[p] || 'info'
}

export function priorityLabel(p) {
  return planPriorities.find((x) => x.value === p)?.label || p
}

export function statusLabel(s) {
  return planStatusColumns.find((x) => x.status === s)?.label || s
}

export function statusColor(s) {
  return planStatusColumns.find((x) => x.status === s)?.color || '#909399'
}

// ===== 人生日历 =====
export function generateGrid(birthDate, lifespan) {
  const birth = dayjs(birthDate)
  const now = dayjs()
  const grid = []
  for (let yearIdx = 0; yearIdx < lifespan; yearIdx++) {
    const row = []
    for (let weekIdx = 0; weekIdx < 52; weekIdx++) {
      const weekStart = birth.add(yearIdx * 52 + weekIdx, 'week')
      const weekEnd = weekStart.add(6, 'day')
      row.push({
        year: yearIdx,
        period: weekIdx,
        startDate: weekStart.format('YYYY-MM-DD'),
        endDate: weekEnd.format('YYYY-MM-DD'),
        isPast: weekEnd.isBefore(now),
        isCurrent: !weekStart.isAfter(now) && !weekEnd.isBefore(now),
        isBorn: !weekStart.isBefore(birth),
      })
    }
    grid.push(row)
  }
  return grid
}

export function getLifeStats(birthDate, lifespan) {
  const birth = dayjs(birthDate)
  const now = dayjs()
  const totalWeeks = lifespan * 52
  const livedWeeks = Math.floor(now.diff(birth, 'week'))
  const livedDays = now.diff(birth, 'day')
  const livedYears = now.diff(birth, 'year')
  const percentage = Math.min(100, Math.round((livedWeeks / totalWeeks) * 1000) / 10)
  return {
    totalWeeks,
    livedWeeks: Math.max(0, livedWeeks),
    remainingWeeks: Math.max(0, totalWeeks - livedWeeks),
    livedDays: Math.max(0, livedDays),
    livedYears,
    percentage,
  }
}
