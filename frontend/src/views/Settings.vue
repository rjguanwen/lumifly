<template>
  <div class="max-w-2xl space-y-6">
    <div>
      <h1 class="text-2xl font-bold text-gray-900">设置</h1>
      <p class="text-gray-500 mt-1">管理你的个人信息</p>
    </div>

    <el-card shadow="never">
      <template #header>
        <span class="font-semibold">个人资料</span>
      </template>
      <el-form label-position="top">
        <div class="flex items-center gap-5 mb-6">
          <UserAvatar :src="auth.user?.avatarUrl" :name="auth.user?.displayName" :size="72" />
          <div class="space-y-2">
            <el-button size="small" :loading="avatarUploading" @click="avatarInput?.click()">
              <el-icon class="mr-1"><Camera /></el-icon>上传 / 更换头像
            </el-button>
            <input
              ref="avatarInput"
              type="file"
              accept="image/jpeg,image/png,image/gif,image/webp"
              class="hidden"
              @change="onAvatarChange"
            />
            <p class="text-xs text-gray-400">支持 jpg / png / gif / webp，不超过 5MB</p>
          </div>
        </div>
        <el-form-item label="昵称">
          <el-input v-model="form.displayName" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input :model-value="auth.user?.email" disabled />
        </el-form-item>
        <el-form-item label="出生日期">
          <el-date-picker v-model="form.birthDate" type="date" class="w-full" value-format="YYYY-MM-DD" />
        </el-form-item>
        <el-form-item label="预期寿命（年）">
          <el-slider v-model="form.expectedLifespan" :min="50" :max="120" show-input :show-input-controls="false" />
          <p class="text-xs text-gray-400 mt-1">用于生成您的人生日历</p>
        </el-form-item>
        <el-form-item label="个性签名">
          <el-input
            v-model="form.signature"
            type="textarea"
            :rows="2"
            maxlength="80"
            show-word-limit
            placeholder="一句话介绍自己，最多 80 字"
          />
        </el-form-item>
        <el-button type="primary" :loading="saving" @click="saveProfile">保存修改</el-button>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <span class="font-semibold">修改密码</span>
      </template>
      <el-form label-position="top">
        <el-form-item label="原密码" required>
          <el-input v-model="pwdForm.oldPassword" type="password" show-password placeholder="请输入原密码" class="max-w-sm" />
        </el-form-item>
        <el-form-item label="新密码" required>
          <el-input v-model="pwdForm.newPassword" type="password" show-password placeholder="至少 8 个字符" class="max-w-sm" />
        </el-form-item>
        <el-form-item label="确认新密码" required>
          <el-input v-model="pwdForm.confirm" type="password" show-password placeholder="再次输入新密码" class="max-w-sm" @keyup.enter="changePassword" />
        </el-form-item>
        <el-button type="primary" plain :loading="changingPwd" @click="changePassword">确认修改</el-button>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <span class="font-semibold">安全设置</span>
      </template>
      <p class="text-xs text-gray-400 mb-5 leading-relaxed">
        设置「密码提示词」与「安全问题」后，忘记密码时可前往登录页「忘记密码」通过问答重置密码。
      </p>

      <el-form label-position="top" class="space-y-4">
        <!-- 密码提示词 -->
        <div class="flex flex-wrap items-end gap-3">
          <el-form-item label="密码提示词（可选）" class="!mb-0 flex-1 min-w-[260px]">
            <el-input
              v-model="secForm.passwordHint"
              placeholder="给自己一句能想起密码的提示，如：常用昵称+生日组合"
              @keyup.enter="saveHint"
            />
          </el-form-item>
          <el-button type="primary" plain :loading="secSavingHint" @click="saveHint">保存提示</el-button>
        </div>

        <!-- 安全问答 -->
        <div class="pt-4 border-t border-gray-100">
          <div class="text-sm font-medium text-gray-700 mb-1.5">找回安全问题</div>
          <p v-if="secForm.existingQuestion" class="text-xs text-gray-400 mb-3">
            当前问题：<span class="text-gray-600">{{ secForm.existingQuestion }}</span>
            （修改问题时需重新填写答案）
          </p>
          <div class="flex flex-wrap items-end gap-3">
            <el-form-item label="选择问题" class="!mb-0 flex-1 min-w-[240px]">
              <el-select
                v-model="secForm.securityQuestion"
                placeholder="选择或输入自定义问题"
                filterable
                allow-create
                default-first-option
                clearable
                class="w-full"
              >
                <el-option v-for="q in securityQuestions" :key="q" :label="q" :value="q" />
              </el-select>
            </el-form-item>
            <el-form-item label="答案（仅用于找回验证）" class="!mb-0 flex-1 min-w-[200px]">
              <el-input
                v-model="secForm.securityAnswer"
                placeholder="回答你的安全问题"
                @keyup.enter="saveSecurityQA"
              />
            </el-form-item>
            <div class="flex gap-2">
              <el-button type="primary" plain :loading="secSavingQA" @click="saveSecurityQA">
                {{ secForm.existingQuestion ? '更新问答' : '保存问答' }}
              </el-button>
              <el-button
                v-if="secForm.existingQuestion"
                type="danger"
                plain
                :loading="secClearing"
                @click="clearSecurityQA"
              >清除问答</el-button>
            </div>
          </div>
        </div>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <template #header>
        <span class="font-semibold">账号信息</span>
      </template>
      <div class="text-sm text-gray-500 space-y-1">
        <div>账号 ID：{{ auth.user?.id }}</div>
        <div>角色：{{ auth.user?.role === 'admin' ? '管理员' : '普通用户' }}</div>
        <div>注册时间：{{ registeredAt }}</div>
      </div>
    </el-card>

    <!-- 头像裁剪 -->
    <AvatarCropper
      v-model:visible="cropVisible"
      :image-url="cropImageUrl"
      :file-name="cropFileName"
      @cropped="uploadCroppedAvatar"
    />
  </div>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useAuthStore } from '../stores/auth'
import UserAvatar from '../components/UserAvatar.vue'
import AvatarCropper from '../components/AvatarCropper.vue'
import { authApi, profileApi } from '../api'

const auth = useAuthStore()
const saving = ref(false)
const changingPwd = ref(false)
const avatarUploading = ref(false)
const avatarInput = ref(null)
const cropVisible = ref(false)
const cropImageUrl = ref('')
const cropFileName = ref('')

async function onAvatarChange(e) {
  const file = e.target.files?.[0]
  if (avatarInput.value) avatarInput.value.value = ''
  if (!file) return
  if (file.size > 5 * 1024 * 1024) {
    ElMessage.warning('图片不能超过 5MB')
    return
  }
  cropFileName.value = file.name
  cropImageUrl.value = URL.createObjectURL(file)
  cropVisible.value = true
}

async function uploadCroppedAvatar(file) {
  avatarUploading.value = true
  try {
    const user = await profileApi.uploadAvatar(file)
    await auth.applyUser(user)
    ElMessage.success('头像已更新')
  } catch {
    /* 拦截器已提示 */
  } finally {
    avatarUploading.value = false
    releaseCropUrl()
  }
}

function releaseCropUrl() {
  if (cropImageUrl.value) URL.revokeObjectURL(cropImageUrl.value)
  cropImageUrl.value = ''
}

// 关闭裁剪框（取消或完成）后释放临时预览 URL
watch(cropVisible, (v) => {
  if (!v) releaseCropUrl()
})

const form = reactive({
  displayName: '',
  birthDate: '',
  expectedLifespan: 80,
  signature: '',
})

const pwdForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirm: '',
})

if (auth.user) {
  form.displayName = auth.user.displayName
  form.birthDate = auth.user.birthDate || ''
  form.expectedLifespan = auth.user.expectedLifespan || 80
  form.signature = auth.user.signature || ''
}

const registeredAt = computed(() => {
  const createdAt = auth.user?.createdAt
  return createdAt ? String(createdAt).slice(0, 10) : '-'
})

async function saveProfile() {
  if (!form.displayName) {
    ElMessage.warning('昵称不能为空')
    return
  }
  saving.value = true
  try {
    await auth.updateProfile({
      displayName: form.displayName,
      birthDate: form.birthDate || null,
      expectedLifespan: form.expectedLifespan,
      signature: form.signature,
    })
    ElMessage.success('已保存')
  } catch {
    /* 错误已由拦截器提示 */
  } finally {
    saving.value = false
  }
}

async function changePassword() {
  if (!pwdForm.oldPassword) {
    ElMessage.warning('请输入原密码')
    return
  }
  if (pwdForm.newPassword.length < 8) {
    ElMessage.warning('新密码至少需要 8 个字符')
    return
  }
  if (pwdForm.newPassword !== pwdForm.confirm) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }
  changingPwd.value = true
  try {
    await authApi.changePassword({
      oldPassword: pwdForm.oldPassword,
      newPassword: pwdForm.newPassword,
    })
    ElMessage.success('密码修改成功，下次登录请使用新密码')
    pwdForm.oldPassword = ''
    pwdForm.newPassword = ''
    pwdForm.confirm = ''
  } catch {
    /* 错误已由拦截器提示 */
  } finally {
    changingPwd.value = false
  }
}

// ===== 安全设置（密码提示与问答找回） =====
const securityQuestions = [
  '你母亲的姓名是？',
  '你父亲的姓名是？',
  '你的出生城市是？',
  '你最喜欢的电影是？',
  '你的小学名称是？',
  '你初中班主任的名字是？',
  '你的宠物名字是？',
]

const secForm = reactive({
  passwordHint: '',
  securityQuestion: '',
  securityAnswer: '',
  existingQuestion: '',
})
const secSavingHint = ref(false)
const secSavingQA = ref(false)
const secClearing = ref(false)

async function loadSecurity() {
  try {
    const data = await authApi.getSecurity()
    secForm.passwordHint = data.password_hint || ''
    secForm.existingQuestion = data.security_question || ''
    secForm.securityQuestion = data.security_question || ''
    secForm.securityAnswer = ''
  } catch { /* 拦截器已提示 */ }
}

async function saveHint() {
  secSavingHint.value = true
  try {
    await authApi.setSecurity({ passwordHint: secForm.passwordHint })
    ElMessage.success('密码提示词已保存')
    loadSecurity()
  } catch { /* 拦截器已提示 */ } finally {
    secSavingHint.value = false
  }
}

async function saveSecurityQA() {
  if (!secForm.securityQuestion) {
    ElMessage.warning('请选择或输入安全问题')
    return
  }
  if (!secForm.securityAnswer.trim()) {
    ElMessage.warning('请填写安全问题答案')
    return
  }
  secSavingQA.value = true
  try {
    await authApi.setSecurity({
      securityQuestion: secForm.securityQuestion,
      securityAnswer: secForm.securityAnswer,
    })
    ElMessage.success('安全问题已保存')
    loadSecurity()
  } catch { /* 拦截器已提示 */ } finally {
    secSavingQA.value = false
  }
}

async function clearSecurityQA() {
  try {
    await ElMessageBox.confirm('确定清除安全问题？清除后将无法通过问答找回密码。', '提示', { type: 'warning' })
  } catch {
    return
  }
  secClearing.value = true
  try {
    await authApi.setSecurity({ securityQuestion: '', securityAnswer: '' })
    ElMessage.success('已清除安全问题')
    loadSecurity()
  } catch { /* 拦截器已提示 */ } finally {
    secClearing.value = false
  }
}

loadSecurity()
</script>
