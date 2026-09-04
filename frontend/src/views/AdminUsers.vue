<template>
  <div class="space-y-6">
    <div>
      <h1 class="text-2xl font-bold text-gray-900">用户管理</h1>
      <p class="text-gray-500 mt-1">停用/启用普通用户，控制系统注册，邀请特定邮箱加入</p>
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

    <!-- 邀请注册 -->
    <el-card shadow="never" class="!rounded-2xl">
      <template #header>
        <div class="flex items-center justify-between">
          <div>
            <span class="font-medium text-gray-800">邀请注册</span>
            <span class="text-xs text-gray-400 ml-2">向指定邮箱发送邀请链接，关闭公开注册后受邀邮箱仍可注册</span>
          </div>
          <el-button size="small" text type="primary" @click="loadInvites">
            <el-icon class="mr-1"><Refresh /></el-icon>刷新
          </el-button>
        </div>
      </template>

      <div class="flex items-start gap-3">
        <el-input
          v-model="inviteEmails"
          type="textarea"
          :rows="3"
          placeholder="输入受邀邮箱，每行一个或以逗号分隔（单次最多 20 个）"
        />
        <el-button type="primary" class="shrink-0 mt-1" :loading="sendingInvite" :disabled="!inviteEmails.trim()" @click="sendInvites">
          发送邀请
        </el-button>
      </div>

      <!-- 发送结果 -->
      <div v-if="inviteResults.length" class="mt-3 rounded-xl border border-gray-200 divide-y divide-gray-100">
        <div v-for="(r, i) in inviteResults" :key="i" class="flex items-start gap-2 px-3 py-2.5 text-sm">
          <span :class="r.ok ? 'text-emerald-500' : 'text-red-500'">{{ r.ok ? '✓' : '✗' }}</span>
          <div class="min-w-0 flex-1">
            <div class="font-medium text-gray-800">{{ r.email }}</div>
            <div class="text-xs text-gray-400 mt-0.5">
              {{ r.reason || (r.dev ? '开发模式：SMTP 未配置，以下为邀请链接（点击可打开注册页）' : '邀请邮件已发送') }}
            </div>
            <a
              v-if="r.ok && r.invite_url"
              :href="r.invite_url"
              target="_blank"
              rel="noopener"
              class="text-xs text-blue-600 break-all hover:underline mt-0.5 inline-block"
            >{{ r.invite_url }}</a>
          </div>
        </div>
      </div>

      <!-- 邀请列表 -->
      <div class="mt-4">
        <el-table v-if="invites.length" :data="invites" stripe v-loading="loadingInvites" style="width: 100%">
          <el-table-column label="受邀邮箱" min-width="200">
            <template #default="{ row }">
              <span class="text-sm text-gray-700">{{ row.email }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag v-if="row.status === 'pending' && !row.expired" size="small" type="warning" effect="plain">待接受</el-tag>
              <el-tag v-else-if="row.status === 'pending' && row.expired" size="small" type="info" effect="plain">已过期</el-tag>
              <el-tag v-else-if="row.status === 'registered'" size="small" type="success" effect="plain">已注册</el-tag>
              <el-tag v-else size="small" type="danger" effect="plain">已撤销</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="邀请人" width="140">
            <template #default="{ row }">
              <span class="text-sm text-gray-600">{{ row.invitedBy?.displayName || '—' }}</span>
            </template>
          </el-table-column>
          <el-table-column label="邀请时间" width="105">
            <template #default="{ row }">{{ (row.createdAt || '').slice(0, 10) }}</template>
          </el-table-column>
          <el-table-column label="有效至" width="105">
            <template #default="{ row }">{{ (row.expiresAt || '').slice(0, 10) }}</template>
          </el-table-column>
          <el-table-column label="操作" width="90">
            <template #default="{ row }">
              <el-button v-if="row.status === 'pending' && !row.expired" size="small" type="danger" plain @click="revokeInvite(row)">撤销</el-button>
              <el-button v-else-if="row.status === 'pending' && row.expired" size="small" type="primary" text @click="resendInvite(row)">重发</el-button>
              <span v-else class="text-xs text-gray-300">—</span>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-else-if="!loadingInvites" description="暂无邀请记录" :image-size="80" />
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
              <UserAvatar :src="row.avatarUrl" :name="row.displayName" :size="30" />
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
import UserAvatar from '../components/UserAvatar.vue'

const auth = useAuthStore()
const users = ref([])
const registrationEnabled = ref(true)
const savingSetting = ref(false)
const inviteEmails = ref('')
const sendingInvite = ref(false)
const inviteResults = ref([])
const invites = ref([])
const loadingInvites = ref(false)

async function loadUsers() {
  try {
    const data = await adminApi.users()
    users.value = data.items || []
  } catch { /* 拦截器已提示 */ }
}

async function loadInvites() {
  loadingInvites.value = true
  try {
    const data = await adminApi.invites()
    invites.value = data.items || []
  } catch { /* 拦截器已提示 */ } finally {
    loadingInvites.value = false
  }
}

// 解析输入框中的邮箱（换行/逗号/分号分隔，去重）
function parseEmails(text) {
  return [...new Set(text.split(/[\n,;，；、\s]+/).map((s) => s.trim()).filter(Boolean))]
}

async function sendInvitesFor(emailList) {
  sendingInvite.value = true
  try {
    const data = await adminApi.createInvites(emailList)
    const results = data.results || []
    inviteResults.value = results
    const okCount = results.filter((r) => r.ok).length
    if (okCount > 0) {
      ElMessage.success(`已为 ${okCount} 个邮箱${results[0]?.dev ? '生成邀请链接' : '发送邀请邮件'}`)
    }
    if (okCount === emailList.length) {
      inviteEmails.value = ''
    }
    loadInvites()
  } catch { /* 拦截器已提示 */ } finally {
    sendingInvite.value = false
  }
}

async function sendInvites() {
  const emailList = parseEmails(inviteEmails.value)
  if (!emailList.length) {
    ElMessage.warning('请先输入受邀邮箱')
    return
  }
  if (emailList.length > 20) {
    ElMessage.warning('单次最多邀请 20 个邮箱')
    return
  }
  await sendInvitesFor(emailList)
}

async function resendInvite(row) {
  inviteResults.value = []
  await sendInvitesFor([row.email])
}

async function revokeInvite(row) {
  try {
    await ElMessageBox.confirm(`撤销对「${row.email}」的邀请？撤销后该邀请链接将立即失效。`, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await adminApi.revokeInvite(row.id)
    ElMessage.success('邀请已撤销')
    loadInvites()
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
  loadInvites()
})
</script>
