<template>
  <div class="auth-page">
    <div class="setup-box">
      <div class="auth-header">
        <BrandLogo :size="64" />
        <h1>欢迎使用飞光</h1>
        <p>人生如逆旅　我亦是行人</p>
      </div>
      <el-card shadow="never">
        <el-form label-position="top" @submit.prevent="handleSetup">
          <el-form-item label="出生日期" required>
            <el-date-picker
              v-model="birthDate"
              type="date"
              placeholder="选择出生日期"
              size="large"
              class="w-full"
              value-format="YYYY-MM-DD"
            />
          </el-form-item>
          <el-form-item label="预期寿命（年）">
            <el-slider v-model="expectedLifespan" :min="50" :max="120" show-input :show-input-controls="false" />
            <p class="text-xs text-gray-400 mt-1">用于生成您的人生日历</p>
          </el-form-item>
          <el-button type="primary" size="large" class="w-full" :loading="loading" @click="handleSetup">
            开始我的人生记录
          </el-button>
        </el-form>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import BrandLogo from '../components/BrandLogo.vue'

const router = useRouter()
const auth = useAuthStore()

const birthDate = ref(auth.user?.birthDate || '')
const expectedLifespan = ref(auth.user?.expectedLifespan || 80)
const loading = ref(false)

async function handleSetup() {
  if (!birthDate.value) {
    ElMessage.warning('请选择出生日期')
    return
  }
  loading.value = true
  try {
    await auth.updateProfile({
      birthDate: birthDate.value,
      expectedLifespan: expectedLifespan.value,
    })
    ElMessage.success('设置完成，欢迎开启人生记录之旅！')
    router.push('/dashboard')
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

.setup-box {
  width: 100%;
  max-width: 520px;
}

.auth-header {
  text-align: center;
  margin-bottom: 24px;
}

.auth-header h1 {
  font-size: 28px;
  font-weight: 700;
  color: #1f2937;
  margin-top: 12px;
}

.auth-header p {
  color: #6b7280;
  margin-top: 6px;
  font-size: 14px;
}
</style>
