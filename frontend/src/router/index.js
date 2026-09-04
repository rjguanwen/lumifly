import { createRouter, createWebHistory } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const routes = [
  {
    path: '/login',
    name: 'login',
    component: () => import('../views/Login.vue'),
    meta: { public: true },
  },
  {
    path: '/register',
    name: 'register',
    component: () => import('../views/Register.vue'),
    meta: { public: true },
  },
  {
    path: '/forgot',
    name: 'forgot',
    component: () => import('../views/ForgotPassword.vue'),
    meta: { public: true, title: '找回密码' },
  },
  {
    path: '/reset-password',
    name: 'reset-password',
    component: () => import('../views/ResetPassword.vue'),
    meta: { public: true, title: '设置新密码' },
  },
  {
    path: '/setup',
    name: 'setup',
    component: () => import('../views/Setup.vue'),
    meta: { public: true, title: '初始设置' },
  },
  {
    path: '/',
    component: () => import('../layout/MainLayout.vue'),
    redirect: '/dashboard',
    children: [
      { path: 'dashboard', name: 'dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '概览' } },
      { path: 'calendar', name: 'calendar', component: () => import('../views/Calendar.vue'), meta: { title: '人生日历' } },
      { path: 'milestones', name: 'milestones', component: () => import('../views/Milestones.vue'), meta: { title: '人生大事记' } },
      { path: 'milestones/:id', name: 'milestone-detail', component: () => import('../views/MilestoneDetailPage.vue'), meta: { title: '大事记详情' } },
      { path: 'records', name: 'records', component: () => import('../views/Records.vue'), meta: { title: '日常记录' } },
      { path: 'records/:id', name: 'record-detail', component: () => import('../views/RecordDetailPage.vue'), meta: { title: '日记详情' } },
      { path: 'ideas', name: 'ideas', component: () => import('../views/Ideas.vue'), meta: { title: '想法灵感' } },
      { path: 'ideas/:id', name: 'idea-detail', component: () => import('../views/IdeaDetailPage.vue'), meta: { title: '灵感详情' } },
      { path: 'books', name: 'books', component: () => import('../views/Books.vue'), meta: { title: '读书记录' } },
      { path: 'books/:id', name: 'book-detail', component: () => import('../views/BookDetailPage.vue'), meta: { title: '书籍详情' } },
      { path: 'square', name: 'square', component: () => import('../views/Square.vue'), meta: { title: '广场' } },
      { path: 'square/:id', name: 'square-detail', component: () => import('../views/SquareDetailPage.vue'), meta: { title: '广场内容' } },
      { path: 'settings', name: 'settings', component: () => import('../views/Settings.vue'), meta: { title: '设置' } },
      { path: 'admin/users', name: 'admin-users', component: () => import('../views/AdminUsers.vue'), meta: { title: '用户管理', requiresAdmin: true } },
    ],
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const auth = useAuthStore()
  if (!auth.isLoaded) {
    await auth.fetchUser()
  }
  if (!to.meta.public) {
    if (!auth.isAuthenticated) {
      return { name: 'login', query: { redirect: to.fullPath } }
    }
    // 管理员专属页面
    if (to.meta.requiresAdmin && auth.user?.role !== 'admin') {
      return { name: 'dashboard' }
    }
    // 未完成初始设置则强制进入设置页
    if (!auth.user?.isSetupComplete && to.name !== 'setup') {
      return { name: 'setup' }
    }
  }
  if ((to.name === 'login' || to.name === 'register') && auth.isAuthenticated) {
    return { name: 'dashboard' }
  }
  if (to.name === 'setup' && auth.isAuthenticated && auth.user?.isSetupComplete) {
    return { name: 'dashboard' }
  }
  return true
})

router.afterEach((to) => {
  document.title = to.meta.title ? `${to.meta.title} · 飞光 Lumifly` : '飞光 Lumifly'
})

export default router
