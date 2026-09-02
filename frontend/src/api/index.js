import axios from 'axios'
import { ElMessage } from 'element-plus'
import router from '../router'

const api = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('lumifly_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const status = error.response?.status
    const detail = error.response?.data?.detail
    if (status === 401) {
      localStorage.removeItem('lumifly_token')
      localStorage.removeItem('lumifly_user')
      if (router.currentRoute.value.path !== '/login') {
        router.push('/login')
      }
    }
    const msg = typeof detail === 'string' ? detail : error.message || '请求失败'
    ElMessage.error(msg)
    return Promise.reject(error)
  },
)

export default api

// ===== 认证 =====
export const authApi = {
  login: (data) => api.post('/auth/login', data),
  register: (data) => api.post('/auth/register', data),
  me: () => api.get('/auth/me'),
  logout: () => api.post('/auth/logout'),
  changePassword: (data) => api.put('/auth/password', data),
  getSecurity: () => api.get('/auth/security'),
  setSecurity: (data) => api.put('/auth/security', data),
  getRecovery: (email) => api.get('/auth/forgot', { params: { email } }),
  resetPassword: (data) => api.post('/auth/forgot/reset', data),
  sendForgotEmail: (email) => api.post('/auth/forgot/send', { email }),
  resetByToken: (data) => api.post('/auth/reset', data),
}

// ===== 管理员 =====
export const adminApi = {
  users: () => api.get('/admin/users'),
  setUserActive: (id, isActive) => api.patch(`/admin/users/${id}`, { isActive }),
  settings: () => api.get('/admin/settings'),
  setRegistration: (enabled) => api.put('/admin/settings/registration', { enabled }),
}

// ===== 个人资料 =====
export const profileApi = {
  update: (data) => api.put('/profile', data),
}

// ===== 日常记录 =====
export const recordApi = {
  list: (params) => api.get('/records', { params }),
  get: (id) => api.get(`/records/${id}`),
  create: (data) => api.post('/records', data),
  update: (id, data) => api.put(`/records/${id}`, data),
  remove: (id) => api.delete(`/records/${id}`),
}

// ===== 人生大事记 =====
export const milestoneApi = {
  list: (params) => api.get('/milestones', { params }),
  get: (id) => api.get(`/milestones/${id}`),
  create: (data) => api.post('/milestones', data),
  update: (id, data) => api.put(`/milestones/${id}`, data),
  remove: (id) => api.delete(`/milestones/${id}`),
}

// ===== 未来规划 =====
export const planApi = {
  list: () => api.get('/plans'),
  get: (id) => api.get(`/plans/${id}`),
  create: (data) => api.post('/plans', data),
  update: (id, data) => api.put(`/plans/${id}`, data),
  remove: (id) => api.delete(`/plans/${id}`),
}

// ===== 想法灵感 =====
export const ideaApi = {
  list: () => api.get('/ideas'),
  get: (id) => api.get(`/ideas/${id}`),
  create: (data) => api.post('/ideas', data),
  update: (id, data) => api.put(`/ideas/${id}`, data),
  remove: (id) => api.delete(`/ideas/${id}`),
}

// ===== 标签 =====
export const tagApi = {
  list: () => api.get('/tags'),
  create: (data) => api.post('/tags', data),
}

// ===== 日历 =====
export const calendarApi = {
  summary: () => api.get('/calendar/summary'),
}

// ===== 上传 =====
export const uploadApi = {
  // 返回 Blob URL 或相对路径，由组件处理
  upload: (file, entityType, entityId) => {
    const form = new FormData()
    form.append('file', file)
    if (entityType) form.append('entityType', entityType)
    if (entityId !== undefined && entityId !== null) form.append('entityId', entityId)
    return api.post('/upload', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 120000,
    })
  },
}
