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
  inviteInfo: (token) => api.get('/auth/invite/info', { params: { token } }),
}

// ===== 管理员 =====
export const adminApi = {
  users: () => api.get('/admin/users'),
  setUserActive: (id, isActive) => api.patch(`/admin/users/${id}`, { isActive }),
  setRole: (id, role) => api.patch(`/admin/users/${id}/role`, { role }),
  settings: () => api.get('/admin/settings'),
  setRegistration: (enabled) => api.put('/admin/settings/registration', { enabled }),
  saveModerationTerms: (custom) => api.put('/admin/moderation/terms', { custom }),
  invites: () => api.get('/admin/invites'),
  createInvites: (emails) => api.post('/admin/invites', { emails }),
  revokeInvite: (id) => api.post(`/admin/invites/${id}/revoke`),
}

// ===== 个人资料 =====
export const profileApi = {
  update: (data) => api.put('/profile', data),
  uploadAvatar: (file) => {
    const form = new FormData()
    form.append('file', file)
    return api.put('/profile/avatar', form, {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 30000,
    })
  },
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
  list: (params) => api.get('/ideas', { params }),
  get: (id) => api.get(`/ideas/${id}`),
  create: (data) => api.post('/ideas', data),
  update: (id, data) => api.put(`/ideas/${id}`, data),
  remove: (id) => api.delete(`/ideas/${id}`),
}

// ===== 速记语录 =====
export const quickNoteApi = {
  list: (params) => api.get('/quick-notes', { params }),
  create: (data) => api.post('/quick-notes', data),
  update: (id, data) => api.put(`/quick-notes/${id}`, data),
  remove: (id) => api.delete(`/quick-notes/${id}`),
}

// ===== 好友 =====
export const friendApi = {
  searchUsers: (keyword) => api.get('/users/search', { params: { keyword } }),
  list: () => api.get('/friends'),
  requests: () => api.get('/friends/requests'),
  sendRequest: (friendId) => api.post('/friends/requests', { friendId }),
  acceptRequest: (id) => api.post(`/friends/requests/${id}/accept`),
  cancelOrReject: (id) => api.delete(`/friends/requests/${id}`),
  remove: (friendId) => api.delete(`/friends/${friendId}`),
}

// ===== 标签 =====
export const tagApi = {
  list: () => api.get('/tags'),
  create: (data) => api.post('/tags', data),
}

// ===== 读书记录 =====
export const bookApi = {
  list: (params) => api.get('/books', { params }),
  get: (id) => api.get(`/books/${id}`),
  create: (data) => api.post('/books', data),
  update: (id, data) => api.put(`/books/${id}`, data),
  remove: (id) => api.delete(`/books/${id}`),
}

// ===== 广场 =====
export const squareApi = {
  publish: (sourceType, sourceId) => api.post('/publications', { sourceType, sourceId }),
  unpublish: (id) => api.delete(`/publications/${id}`),
  mine: () => api.get('/publications/mine'),
  list: (params) => api.get('/publications', { params }),
  detail: (id) => api.get(`/publications/${id}`),
  like: (id) => api.post(`/publications/${id}/like`),
  comments: (id) => api.get(`/publications/${id}/comments`),
  addComment: (id, content) => api.post(`/publications/${id}/comments`, { content }),
  deleteComment: (id) => api.delete(`/publications/comment/${id}`),
}

// ===== 内容审核（审核员/管理员） =====
export const moderateApi = {
  queue: () => api.get('/moderation/queue'),
  approve: (id) => api.post(`/moderation/queue/${id}/approve`),
  reject: (id) => api.post(`/moderation/queue/${id}/reject`),
  terms: () => api.get('/moderation/terms'),
}

// ===== 人生日历分享 =====
export const calendarShareApi = {
  mine: () => api.get('/calendar/shares'),
  add: (friendId) => api.post('/calendar/shares', { friendId }),
  remove: (friendId) => api.delete(`/calendar/shares/${friendId}`),
  received: () => api.get('/calendar/shares/received'),
}

export const sharedCalendarApi = {
  summary: (userId) => api.get(`/calendar/shared/${userId}/summary`),
  period: (userId, from, to) => api.get(`/calendar/shared/${userId}/period`, { params: { from, to } }),
}

// ===== 人生日历私密评价 =====
export const calendarCommentApi = {
  list: (ownerId, targetType, targetId) =>
    api.get('/calendar/comments', { params: { ownerId, targetType, targetId } }),
  post: (data) => api.post('/calendar/comments', data),
  remove: (id) => api.delete(`/calendar/comments/${id}`),
}

// ===== 私密评论中心（与某好友的往来） =====
export const calendarCenterApi = {
  unread: () => api.get('/calendar/comment-center/unread'),
  center: (friendId) => api.get('/calendar/comment-center', { params: { friendId } }),
  read: (friendId) => api.post('/calendar/comment-center/read', { friendId }),
}

// ===== 菜单红点统计 =====
export const badgeApi = {
  unread: () => api.get('/badges/unread'),
}

// ===== 书籍领域库 =====
export const bookDomainApi = {
  list: () => api.get('/book-domains'),
  create: (name) => api.post('/book-domains', { name }),
  rename: (id, name) => api.put(`/book-domains/${id}`, { name }),
  remove: (id) => api.delete(`/book-domains/${id}`),
}

export const defaultBookDomains = ['文学', '小说', '科幻', '技术', '历史', '哲学', '心理学', '商业', '传记', '艺术', '其他']

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
  remove: (id) => api.delete(`/media/${id}`),
}
