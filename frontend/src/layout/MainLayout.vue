<template>
  <el-container class="app-layout">
    <!-- 侧边栏 -->
    <el-aside width="220px" class="app-aside">
      <div class="logo">
        <div class="flex items-center gap-2.5">
          <BrandLogo :size="38" />
          <div>
            <div class="logo-title">飞光</div>
            <div class="logo-sub">人生如逆旅　我亦是行人</div>
          </div>
        </div>
      </div>
      <el-menu :default-active="activeMenu" router class="app-menu" @select="handleSelect">
        <el-menu-item v-for="item in navItems" :key="item.path" :index="item.path">
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.label }}</span>
        </el-menu-item>
      </el-menu>
      <div class="aside-footer">
        <el-menu router class="app-menu">
          <el-menu-item index="/settings">
            <el-icon><Setting /></el-icon>
            <span>设置</span>
          </el-menu-item>
        </el-menu>
        <el-button text class="logout-btn" @click="handleLogout">
          <el-icon><SwitchButton /></el-icon>
          <span>退出登录</span>
        </el-button>
      </div>
    </el-aside>

    <!-- 主区域 -->
    <el-container class="app-main">
      <el-header class="app-header">
        <div class="header-title">{{ route.meta.title || '' }}</div>
        <div class="flex items-center gap-2">
          <UserAvatar :src="auth.user?.avatarUrl" :name="auth.user?.displayName" :size="28" />
          <span class="text-sm text-gray-600">{{ auth.user?.displayName }}</span>
        </div>
      </el-header>
      <el-main class="app-content">
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { ElMessageBox } from 'element-plus'
import BrandLogo from '../components/BrandLogo.vue'
import UserAvatar from '../components/UserAvatar.vue'

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()

const isAdmin = computed(() => auth.user?.role === 'admin')
const isModerator = computed(() => auth.user?.role === 'admin' || auth.user?.role === 'moderator')

// 详情子路由（/records/:id 等）时高亮对应菜单
const activeMenu = computed(() => {
  const p = route.path
  const sorted = [...navItems.value].sort((a, b) => b.path.length - a.path.length)
  const matched = sorted.find((i) => p === i.path || p.startsWith(i.path + '/'))
  return matched ? matched.path : p
})

const navItems = computed(() => {
  const items = [
    { path: '/dashboard', label: '概览', icon: 'HomeFilled' },
    { path: '/calendar', label: '人生日历', icon: 'Calendar' },
    { path: '/milestones', label: '大事记', icon: 'Trophy' },
    { path: '/records', label: '日常记录', icon: 'EditPen' },
    { path: '/ideas', label: '想法灵感', icon: 'Lightning' },
    { path: '/quick-notes', label: '速记语录', icon: 'ChatDotSquare' },
    { path: '/books', label: '读书记录', icon: 'Collection' },
    { path: '/friends', label: '好友', icon: 'UserFilled' },
    { path: '/shared-calendar', label: '好友日历', icon: 'View' },
    { path: '/square', label: '广场', icon: 'Compass' },
  ]
  if (isModerator.value) {
    items.push({ path: '/review', label: '内容审核', icon: 'Checked' })
  }
  if (isAdmin.value) {
    items.push({ path: '/admin/users', label: '用户管理', icon: 'User' })
  }
  return items
})

function handleSelect() {
  /* router 模式自动跳转 */
}

async function handleLogout() {
  try {
    await ElMessageBox.confirm('确定退出登录吗？', '提示', { type: 'warning' })
  } catch {
    return
  }
  await auth.logout()
  router.push('/login')
}
</script>

<style scoped>
.app-layout {
  height: 100vh;
}

.app-aside {
  background: #fff;
  border-right: 1px solid #e5e7eb;
  display: flex;
  flex-direction: column;
}

.logo {
  padding: 20px 24px;
  border-bottom: 1px solid #f3f4f6;
}

.logo-title {
  font-size: 20px;
  font-weight: 700;
  color: #1f2937;
}

.logo-sub {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 2px;
}

.app-menu {
  border-right: none;
  flex: 1;
  padding: 8px;
}

.aside-footer {
  border-top: 1px solid #f3f4f6;
  padding: 8px;
}

.logout-btn {
  width: 100%;
  justify-content: flex-start;
  padding-left: 20px;
}

.app-main {
  background: #f5f7fa;
}

.app-header {
  background: #fff;
  border-bottom: 1px solid #e5e7eb;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 56px;
}

.header-title {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
}

.app-content {
  padding: 20px;
  overflow-y: auto;
}
</style>
