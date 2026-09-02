import { defineStore } from 'pinia'
import { authApi, profileApi } from '../api'

export const useAuthStore = defineStore('auth', {
  state: () => ({
    user: null,
    isLoaded: false,
  }),
  getters: {
    isAuthenticated: (state) => !!state.user,
  },
  actions: {
    setAuth(token, user) {
      if (token) localStorage.setItem('lumifly_token', token)
      localStorage.setItem('lumifly_user', JSON.stringify(user))
      this.user = user
    },
    async fetchUser() {
      this.isLoaded = false
      const token = localStorage.getItem('lumifly_token')
      if (!token) {
        this.user = null
        this.isLoaded = true
        return null
      }
      try {
        const user = await authApi.me()
        this.user = user
        localStorage.setItem('lumifly_user', JSON.stringify(user))
        return user
      } catch {
        this.user = null
        return null
      } finally {
        this.isLoaded = true
      }
    },
    async login(email, password) {
      const data = await authApi.login({ email, password })
      this.setAuth(data.token, data.user)
      return data
    },
    async register(email, password, displayName) {
      const data = await authApi.register({ email, password, displayName })
      this.setAuth(data.token, data.user)
      return data
    },
    async updateProfile(payload) {
      const user = await profileApi.update(payload)
      this.user = user
      localStorage.setItem('lumifly_user', JSON.stringify(user))
      return user
    },
    async logout() {
      try {
        await authApi.logout()
      } catch {
        /* 忽略退出接口错误 */
      }
      localStorage.removeItem('lumifly_token')
      localStorage.removeItem('lumifly_user')
      this.user = null
    },
  },
})
