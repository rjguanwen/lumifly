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
          <div class="text-center font-semibold text-lg">登录</div>
        </template>
        <el-form :model="form" label-position="top" @submit.prevent="handleLogin">
          <el-form-item label="邮箱" required>
            <el-input v-model="form.email" type="email" placeholder="请输入邮箱" size="large" />
          </el-form-item>
          <el-form-item label="密码" required>
            <el-input v-model="form.password" type="password" show-password placeholder="请输入密码" size="large" @keyup.enter="handleLogin" />
          </el-form-item>
          <el-button type="primary" size="large" class="w-full mt-2" :loading="loading" @click="handleLogin">
            登录
          </el-button>
          <div class="mt-4 flex items-center justify-center gap-4 text-sm text-gray-500">
            <router-link to="/register" class="text-blue-600 hover:underline">立即注册</router-link>
            <span class="text-gray-300">|</span>
            <router-link to="/forgot" class="text-gray-500 hover:text-blue-600 hover:underline">忘记密码？</router-link>
          </div>
        </el-form>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import BrandLogo from '../components/BrandLogo.vue'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()

const form = reactive({ email: '', password: '' })
const loading = ref(false)

async function handleLogin() {
  if (!form.email || !form.password) {
    ElMessage.warning('请输入邮箱和密码')
    return
  }
  loading.value = true
  try {
    await auth.login(form.email, form.password)
    ElMessage.success('登录成功')
    router.push(route.query.redirect || (auth.user?.isSetupComplete ? '/dashboard' : '/setup'))
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
