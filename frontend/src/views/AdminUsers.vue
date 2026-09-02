<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-bold text-gray-900">用户管理</h1>
      <p class="text-gray-500 mt-1">停用/启用普通用户，控制系统注册</p>
    </div>

    <!-- 注册开关 -->
    <el-card shadow="never" class="!rounded-2xl">
      <div class="flex items-center justify-between">
        <div>
          <div class="font-medium text-gray-800">允许新用户注册</div>
          <p class="text-xs text-gray-400 mt-1">关闭后注册入口将提示"系统已暂停注册"，现有用户不受影响</p>
        </div>
        <el-switch
          v-model="registrationEnabled"
          :loading="savingSetting"
          @change="toggleRegistration"
        />
      </div>
    </el-card>

    <!-- 用户列表 -->
    <el-card shadow="never" class="!rounded-2xl">
      <template #header>
        <div class="flex items-center justify-between">
          <span class="font-medium text-gray-800">用户列表（{{ users.length }}）</span>
          <el-button size="small" text @click="loadUsers">
            <el-icon class="mr-1"><Refresh /></el-icon>刷新
          </el-button>
        </div>
      </template>

      <el-table :data="users" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column label="用户">
          <template #default="{ row }">
            <div class="flex items-center gap-2.5">
              <el-avatar :size="30" class="!bg-brand-500">{{ (row.displayName || '?').charAt(0) }}</el-avatar>
              <div>
                <div class="text-sm font-medium text-gray-800">{{ row.displayName }}</div>
                <div class="text-xs text-gray-400">{{ row.email }}</div>
              </div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="角色" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.role === 'admin'" size="small" type="danger" effect="dark">管理员</el-tag>
            <el-tag v-else size="small" type="info" effect="plain">普通用户</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag v-if="row.isActive" size="small" type="success">正常</el-tag>
            <el-tag v-else size="small" type="danger">已停用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="注册时间" width="120">
          <template #default="{ row }">{{ (row.createdAt || '').slice(0, 10) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="130">
          <template #default="{ row }">
            <template v-if="row.id === auth.user?.id">
              <span class="text-xs text-gray-300">（本人）</span>
            </template>
            <template v-else-if="row.role === 'admin'">
              <span class="text-xs text-gray-300">管理员</span>
            </template>
            <el-button v-else size="small" :type="row.isActive ? 'danger' : 'success'" plain @click="toggleActive(row)">
              {{ row.isActive ? '停用' : '启用' }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { adminApi } from '../api'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const users = ref([])
const registrationEnabled = ref(true)
const savingSetting = ref(false)

async function loadUsers() {
  try {
    const data = await adminApi.users()
    users.value = data.items || []
  } catch { /* 拦截器已提示 */ }
}

async function loadSettings() {
  try {
    const data = await adminApi.settings()
    registrationEnabled.value = !!data.registration_enabled
  } catch { /* 忽略 */ }
}

async function toggleRegistration(value) {
  savingSetting.value = true
  try {
    await adminApi.setRegistration(value)
    ElMessage.success(value ? '已开放注册' : '已暂停注册')
  } catch {
    registrationEnabled.value = !value
  } finally {
    savingSetting.value = false
  }
}

async function toggleActive(row) {
  const action = row.isActive ? '停用' : '启用'
  try {
    await ElMessageBox.confirm(
      `${action}用户「${row.displayName}」？${action === '停用' ? '该用户将立即无法登录和使用。' : ''}`,
      '提示',
      { type: action === '停用' ? 'warning' : 'info' },
    )
  } catch {
    return
  }
  try {
    await adminApi.setUserActive(row.id, !row.isActive)
    ElMessage.success(`已${action}用户 ${row.displayName}`)
    loadUsers()
  } catch { /* 拦截器已提示 */ }
}

onMounted(() => {
  loadUsers()
  loadSettings()
})
</script>
