<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-2xl font-bold text-gray-900">未来规划</h1>
        <p class="text-gray-500 mt-1">规划人生，追踪进度</p>
      </div>
      <el-button type="primary" @click="openCreate">
        <el-icon class="mr-1"><Plus /></el-icon>新建规划
      </el-button>
    </div>

    <!-- 看板 -->
    <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
      <div v-for="column in planStatusColumns" :key="column.status" class="bg-gray-100 rounded-xl p-4">
        <div class="flex items-center justify-between mb-3">
          <h3 class="font-medium text-gray-700 flex items-center gap-2">
            <span class="w-2.5 h-2.5 rounded-full" :style="{ background: column.color }" />
            {{ column.label }}
          </h3>
          <span class="text-xs text-gray-400 bg-white px-2 py-0.5 rounded-full">{{ getColumnPlans(column.status).length }}</span>
        </div>
        <div class="space-y-3 min-h-[100px]">
          <el-card
            v-for="plan in getColumnPlans(column.status)"
            :key="plan.id"
            shadow="hover"
            class="cursor-pointer"
            @click="viewPlan(plan)"
          >
            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <el-tag size="small" :type="priorityType(plan.priority)">{{ priorityLabel(plan.priority) }}</el-tag>
                <el-dropdown trigger="click" @command="(cmd) => handlePlanCommand(cmd, plan)">
                  <el-button size="small" text><el-icon><MoreFilled /></el-icon></el-button>
                  <template #dropdown>
                    <el-dropdown-menu>
                      <el-dropdown-item command="edit">编辑</el-dropdown-item>
                      <el-dropdown-item v-for="c in otherColumns(plan.status)" :key="c.status" :command="`move:${c.status}`">
                        移至{{ c.label }}
                      </el-dropdown-item>
                      <el-dropdown-item command="delete" divided>删除</el-dropdown-item>
                    </el-dropdown-menu>
                  </template>
                </el-dropdown>
              </div>
              <h4 class="font-medium text-gray-900 text-sm">{{ plan.title }}</h4>
              <div v-if="plan.description" class="text-xs text-gray-500 line-clamp-2 prose prose-sm" v-html="plan.description" />
              <div class="flex items-center gap-2 text-xs text-gray-400">
                <span v-if="plan.targetDate"><el-icon :size="12"><Calendar /></el-icon> {{ plan.targetDate }}</span>
              </div>
            </div>
          </el-card>
        </div>
      </div>
    </div>

    <!-- 新建/编辑弹窗 -->
    <el-dialog v-model="showForm" :title="editingId ? '编辑规划' : '新建规划'" width="560px" destroy-on-close>
      <el-form label-position="top">
        <el-form-item label="标题" required>
          <el-input v-model="form.title" placeholder="目标名称" />
        </el-form-item>
        <el-form-item label="描述">
          <RichTextEditor v-model="form.description" placeholder="详细描述你的目标..." />
        </el-form-item>
        <div class="grid grid-cols-2 gap-4">
          <el-form-item label="开始日期">
            <el-date-picker v-model="form.startDate" type="date" class="w-full" value-format="YYYY-MM-DD" />
          </el-form-item>
          <el-form-item label="目标日期">
            <el-date-picker v-model="form.targetDate" type="date" class="w-full" value-format="YYYY-MM-DD" />
          </el-form-item>
        </div>
        <div class="grid grid-cols-2 gap-4">
          <el-form-item label="优先级">
            <el-select v-model="form.priority" class="w-full">
              <el-option v-for="p in planPriorities" :key="p.value" :label="p.label" :value="p.value" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="form.status" class="w-full">
              <el-option v-for="s in planStatusOptions" :key="s.value" :label="s.label" :value="s.value" />
            </el-select>
          </el-form-item>
        </div>
      </el-form>
      <template #footer>
        <el-button @click="resetForm">取消</el-button>
        <el-button type="primary" :loading="saving" @click="savePlan">保存</el-button>
      </template>
    </el-dialog>

    <!-- 详情抽屉 -->
    <el-drawer v-model="showDetail" size="420px" title="规划详情">
      <PlanDetail v-if="selectedItem" :item="selectedItem" />
    </el-drawer>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import RichTextEditor from '../components/RichTextEditor.vue'
import PlanDetail from '../components/detail/PlanDetail.vue'
import { planApi } from '../api'
import {
  planPriorities,
  planStatusColumns,
  planStatusOptions,
  priorityType,
  priorityLabel,
} from '../utils/helpers'

const showForm = ref(false)
const saving = ref(false)
const editingId = ref(null)
const selectedItem = ref(null)
const showDetail = ref(false)
const items = ref([])

const form = reactive({
  title: '',
  description: '',
  startDate: '',
  targetDate: '',
  priority: 'medium',
  status: 'not_started',
})

async function load() {
  try {
    const data = await planApi.list()
    items.value = data.items || []
  } catch {
    /* 忽略 */
  }
}
load()

function getColumnPlans(status) {
  return items.value.filter((p) => p.status === status)
}

function otherColumns(status) {
  return planStatusColumns.filter((c) => c.status !== status)
}

function viewPlan(plan) {
  selectedItem.value = plan
  showDetail.value = true
}

function openCreate() {
  resetForm()
  showForm.value = true
}

function editPlan(plan) {
  editingId.value = plan.id
  form.title = plan.title
  form.description = plan.description || ''
  form.startDate = plan.startDate || ''
  form.targetDate = plan.targetDate || ''
  form.priority = plan.priority
  form.status = plan.status
  showForm.value = true
}

function resetForm() {
  editingId.value = null
  form.title = ''
  form.description = ''
  form.startDate = ''
  form.targetDate = ''
  form.priority = 'medium'
  form.status = 'not_started'
  showForm.value = false
}

function handlePlanCommand(cmd, plan) {
  if (cmd === 'edit') return editPlan(plan)
  if (cmd === 'delete') return deletePlan(plan.id)
  if (cmd.startsWith('move:')) return updateStatus(plan.id, cmd.split(':')[1])
}

async function savePlan() {
  if (!form.title) {
    ElMessage.warning('请填写标题')
    return
  }
  saving.value = true
  try {
    if (editingId.value) {
      await planApi.update(editingId.value, { ...form })
      ElMessage.success('规划已更新')
    } else {
      await planApi.create({ ...form })
      ElMessage.success('规划已创建')
    }
    resetForm()
    load()
  } catch {
    /* 错误已由拦截器提示 */
  } finally {
    saving.value = false
  }
}

async function updateStatus(id, status) {
  try {
    await planApi.update(id, { status })
    load()
  } catch {
    /* 忽略 */
  }
}

async function deletePlan(id) {
  try {
    await ElMessageBox.confirm('确定删除这条规划？', '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await planApi.remove(id)
    ElMessage.success('已删除')
    load()
  } catch {
    /* 忽略 */
  }
}
</script>
