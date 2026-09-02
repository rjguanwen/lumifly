<template>
  <div class="auth-page">
    <div class="auth-box">
      <div class="auth-header">
        <BrandLogo :size="56" />
        <h1>设置新密码</h1>
        <p>通过邮箱重置链接为账号设置新密码</p>
      </div>

      <el-card shadow="never">
        <!-- 有 token：输入新密码 -->
        <div v-if="token">
          <div v-if="!done">
            <el-alert type="success" :closable="false" show-icon class="mb-4">
              <template #title>验证通过</template>
              <p class="text-xs mt-0.5 text-gray-400">请为你的账号设置一个新密码</p>
            </el-alert>
            <el-form label-position="top" @submit.prevent="doReset">
              <el-form-item label="新密码" required>
                <el-input v-model="newPassword" type="password" show-password placeholder="至少 8 个字符" size="large" />
              </el-form-item>
              <el-form-item label="确认新密码" required>
                <el-input v-model="confirm" type="password" show-password placeholder="再次输入新密码" size="large" @keyup.enter="doReset" />
              </el-form-item>
              <el-button type="primary" size="large" class="w-full" :loading="loading" @click="doReset">确认重置</el-button>
            </el-form>
          </div>
          <div v-else class="text-center py-4">
            <el-icon :size="48" color="#16a34a"><CircleCheckFilled /></el-icon>
            <h3 class="text-lg font-semibold text-gray-900 mt-3">密码已重置</h3>
            <p class="text-sm text-gray-500 mt-1">请使用新密码登录</p>
            <el-button type="primary" class="mt-6 w-full" @click="$router.push('/login')">去登录</el-button>
          </div>
        </div>

        <!-- 无 token：链接无效 -->
        <div v-else class="text-center py-6">
          <el-icon :size="48" color="#f59e0b"><WarningFilled /></el-icon>
          <h3 class="text-lg font-semibold text-gray-900 mt-3">链接无效或缺少令牌</h3>
          <p class="text-sm text-gray-500 mt-1">请从邮件中的重置链接进入，或重新发起找回。</p>
          <el-button type="primary" class="mt-6 w-full" @click="$router.push('/forgot')">重新找回</el-button>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import BrandLogo from '../components/BrandLogo.vue'
import { authApi } from '../api'

const route = useRoute()
const router = useRouter()
const token = ref('')
const newPassword = ref('')
const confirm = ref('')
const loading = ref(false)
const done = ref(false)

onMounted(() => {
  token.value = typeof route.query.token === 'string' ? route.query.token : ''
})

async function doReset() {
  if (newPassword.value.length < 8) {
    ElMessage.warning('新密码至少需要 8 个字符')
    return
  }
  if (newPassword.value !== confirm.value) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  loading.value = true
  try {
    await authApi.resetByToken({ token: token.value, newPassword: newPassword.value })
    done.value = true
  } catch {
    /* 拦截器已提示（链接过期等） */
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.auth-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #eef2ff 0%, #f5f7fa 100%);
  padding: 16px;
}

.auth-box {
  width: 100%;
  max-width: 460px;
}

.auth-header {
  text-align: center;
  margin-bottom: 20px;
}

.auth-header h1 {
  font-size: 28px;
  font-weight: 700;
  color: #1f2937;
  margin-top: 10px;
}

.auth-header p {
  color: #6b7280;
  margin-top: 4px;
  font-size: 14px;
}
</style>
