<template>
  <div class="auth-page">
    <div class="auth-box">
      <div class="auth-header">
        <BrandLogo :size="64" />
        <h1>飞光</h1>
        <p>人生如逆旅　我亦是行人</p>
      </div>
      <el-card shadow="never" class="auth-card">
        <template #header>
          <div class="text-center font-semibold text-lg">注册账号</div>
        </template>
        <el-form :model="form" label-position="top" @submit.prevent="handleRegister">
          <el-form-item label="昵称" required>
            <el-input v-model="form.displayName" placeholder="给自己起个昵称" size="large" />
          </el-form-item>
          <el-form-item label="邮箱" required>
            <el-input v-model="form.email" type="email" placeholder="请输入邮箱" size="large" />
          </el-form-item>
          <el-form-item label="密码" required>
            <el-input v-model="form.password" type="password" show-password placeholder="至少 8 个字符" size="large" />
          </el-form-item>
          <el-form-item label="确认密码" required>
            <el-input v-model="form.confirm" type="password" show-password placeholder="再次输入密码" size="large" @keyup.enter="handleRegister" />
          </el-form-item>
          <el-button type="primary" size="large" class="w-full mt-2" :loading="loading" @click="handleRegister">
            注册
          </el-button>
          <div class="mt-4 text-center text-sm text-gray-500">
            已有账号？
            <router-link to="/login" class="text-blue-600">去登录</router-link>
          </div>
        </el-form>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import BrandLogo from '../components/BrandLogo.vue'

const router = useRouter()
const auth = useAuthStore()

const form = reactive({ displayName: '', email: '', password: '', confirm: '' })
const loading = ref(false)

async function handleRegister() {
  if (!form.displayName || !form.email || !form.password) {
    ElMessage.warning('请填写完整信息')
    return
  }
  if (form.password.length < 8) {
    ElMessage.warning('密码至少需要 8 个字符')
    return
  }
  if (form.password !== form.confirm) {
    ElMessage.warning('两次输入的密码不一致')
    return
  }
  loading.value = true
  try {
    await auth.register(form.email, form.password, form.displayName)
    ElMessage.success('注册成功，欢迎使用飞光！')
    router.push('/setup')
  } catch {
    /* 错误已由拦截器提示 */
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
  max-width: 420px;
}

.auth-header {
  text-align: center;
  margin-bottom: 24px;
}

.auth-header h1 {
  font-size: 32px;
  font-weight: 700;
  color: #1f2937;
  margin-top: 10px;
}

.auth-header p {
  color: #6b7280;
  margin-top: 6px;
  font-size: 14px;
}
</style>
