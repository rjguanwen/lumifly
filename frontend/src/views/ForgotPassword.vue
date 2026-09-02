<template>
  <div class="auth-page">
    <div class="auth-box">
      <div class="auth-header">
        <BrandLogo :size="56" />
        <h1>找回密码</h1>
        <p>通过邮箱链接或安全问答重置登录密码</p>
      </div>

      <el-card shadow="never">
        <el-tabs v-model="activeTab">
          <!-- ===== 方式一：邮箱找回 ===== -->
          <el-tab-pane label="通过邮箱找回" name="email">
            <div v-if="!mailSent" class="py-2">
              <el-form label-position="top" @submit.prevent="sendMail">
                <el-form-item label="注册邮箱" required>
                  <el-input
                    v-model="mailEmail"
                    type="email"
                    placeholder="请输入注册时使用的邮箱"
                    size="large"
                    @keyup.enter="sendMail"
                  />
                </el-form-item>
                <p class="text-xs text-gray-400 -mt-1 mb-3">
                  我们会向该邮箱发送一封含重置链接的邮件，链接 30 分钟内有效。
                </p>
                <el-button type="primary" size="large" class="w-full" :loading="mailSending" @click="sendMail">
                  发送重置邮件
                </el-button>
              </el-form>
            </div>

            <div v-else class="py-2 text-center">
              <div class="w-16 h-16 mx-auto rounded-full bg-green-50 flex items-center justify-center mb-4">
                <el-icon :size="30" color="#16a34a"><Message /></el-icon>
              </div>
              <h3 class="text-lg font-semibold text-gray-900">邮件已发送</h3>
              <p class="text-sm text-gray-500 mt-2">
                请前往 <b class="text-gray-700">{{ mailEmail }}</b> 查收邮件，点击邮件中的链接设置新密码。
              </p>
              <p class="text-xs text-gray-400 mt-2">若 1 分钟内未收到，请检查垃圾箱后重试。</p>
              <div class="flex justify-center gap-3 mt-6">
                <el-button @click="mailSent = false">重新发送</el-button>
                <el-button type="primary" @click="$router.push('/login')">返回登录</el-button>
              </div>
            </div>

            <!-- 开发模式（未配置 SMTP）直接给出重置入口 -->
            <div v-if="devMode" class="mt-3 rounded-xl border border-dashed border-yellow-300 bg-yellow-50 p-4">
              <div class="flex items-center gap-2 text-yellow-700 text-sm font-medium">
                <el-icon><WarningFilled /></el-icon>开发模式提示
              </div>
              <p class="text-xs text-yellow-600 mt-1 mb-3">当前系统未配置邮件服务，无法真实发信。已为你生成了重置链接：</p>
              <el-button type="warning" plain size="small" @click="goResetUrl">前往设置新密码 →</el-button>
            </div>
          </el-tab-pane>

          <!-- ===== 方式二：安全问答 ===== -->
          <el-tab-pane label="通过安全问答找回" name="security">
            <!-- 步骤1：输入邮箱 -->
            <div v-if="qaStep === 1">
              <el-form label-position="top" @submit.prevent>
                <el-form-item label="注册邮箱" required>
                  <el-input v-model="qaEmail" type="email" placeholder="请输入注册邮箱" size="large" @keyup.enter="qaNext" />
                </el-form-item>
                <el-button type="primary" size="large" class="w-full" :loading="qaLoading" @click="qaNext">下一步</el-button>
              </el-form>
            </div>

            <!-- 步骤2：回答问题 -->
            <div v-else-if="qaStep === 2">
              <div v-if="qaHint" class="mb-4 p-4 rounded-xl border border-blue-100 bg-blue-50">
                <div class="text-xs text-blue-500 mb-1">你的密码提示</div>
                <div class="text-gray-700">{{ qaHint }}</div>
              </div>
              <el-alert
                v-if="!qaHasSecurity"
                type="warning"
                :closable="false"
                show-icon
                title="该账号未设置安全问题，无法通过问答找回"
                description="请改用「通过邮箱找回」；或联系管理员。"
                class="mb-4"
              />
              <el-form v-if="qaHasSecurity" label-position="top" @submit.prevent>
                <el-form-item label="安全问题" required>
                  <el-input :model-value="qaQuestion" disabled size="large" />
                </el-form-item>
                <el-form-item label="回答" required>
                  <el-input v-model="qaAnswer" placeholder="请输入答案" size="large" @keyup.enter="qaReset" />
                </el-form-item>
                <el-form-item label="新密码" required>
                  <el-input v-model="qaNewPassword" type="password" show-password placeholder="至少 8 个字符" size="large" />
                </el-form-item>
                <el-form-item label="确认新密码" required>
                  <el-input v-model="qaConfirm" type="password" show-password placeholder="再次输入新密码" size="large" @keyup.enter="qaReset" />
                </el-form-item>
                <el-button type="primary" size="large" class="w-full" :loading="qaLoading" @click="qaReset">重置密码</el-button>
              </el-form>
              <div class="mt-3 text-center">
                <button type="button" class="text-sm text-gray-400 hover:text-gray-600" @click="qaStep = 1">← 换个邮箱</button>
              </div>
            </div>

            <!-- 步骤3：成功 -->
            <div v-else class="text-center py-4">
              <el-icon :size="48" color="#16a34a"><CircleCheckFilled /></el-icon>
              <h3 class="text-lg font-semibold text-gray-900 mt-3">密码重置成功</h3>
              <el-button type="primary" class="mt-6 w-full" @click="$router.push('/login')">去登录</el-button>
            </div>
          </el-tab-pane>
        </el-tabs>

        <div class="mt-3 text-center text-sm text-gray-500">
          想起密码了？<router-link to="/login" class="text-blue-600">返回登录</router-link>
        </div>
      </el-card>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import BrandLogo from '../components/BrandLogo.vue'
import { authApi } from '../api'

const router = useRouter()

// ===== 邮箱找回 =====
const activeTab = ref('email')
const mailEmail = ref('')
const mailSending = ref(false)
const mailSent = ref(false)
const devMode = ref(false)
const devResetUrl = ref('')

async function sendMail() {
  if (!mailEmail.value || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(mailEmail.value)) {
    ElMessage.warning('请输入正确的邮箱')
    return
  }
  mailSending.value = true
  try {
    const res = await authApi.sendForgotEmail(mailEmail.value)
    if (res.dev) {
      devResetUrl.value = res.reset_url || ''
      devMode.value = true
    } else {
      devMode.value = false
    }
    mailSent.value = true
  } catch {
    /* 拦截器已提示（未注册/已发送） */
  } finally {
    mailSending.value = false
  }
}

function goResetUrl() {
  if (!devResetUrl.value) return
  const url = new URL(devResetUrl.value, window.location.origin)
  const token = url.searchParams.get('token')
  if (token) router.push({ path: '/reset-password', query: { token } })
  else window.location.href = url.href
}

// ===== 安全问答 =====
const qaStep = ref(1)
const qaEmail = ref('')
const qaHint = ref('')
const qaQuestion = ref('')
const qaHasSecurity = ref(false)
const qaAnswer = ref('')
const qaNewPassword = ref('')
const qaConfirm = ref('')
const qaLoading = ref(false)

async function qaNext() {
  if (!qaEmail.value || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(qaEmail.value)) {
    ElMessage.warning('请输入正确的邮箱')
    return
  }
  qaLoading.value = true
  try {
    const data = await authApi.getRecovery(qaEmail.value)
    qaHint.value = data.password_hint || ''
    qaQuestion.value = data.security_question || ''
    qaHasSecurity.value = !!data.has_security
    qaStep.value = 2
  } catch {
    /* 拦截器已提示 */
  } finally {
    qaLoading.value = false
  }
}

async function qaReset() {
  if (!qaAnswer.value.trim()) return ElMessage.warning('请回答安全问题')
  if (qaNewPassword.value.length < 8) return ElMessage.warning('新密码至少需要 8 个字符')
  if (qaNewPassword.value !== qaConfirm.value) return ElMessage.warning('两次输入的新密码不一致')
  qaLoading.value = true
  try {
    await authApi.resetPassword({
      email: qaEmail.value,
      answer: qaAnswer.value.trim(),
      newPassword: qaNewPassword.value,
    })
    qaStep.value = 3
  } catch {
    /* 拦截器已提示 */
  } finally {
    qaLoading.value = false
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
