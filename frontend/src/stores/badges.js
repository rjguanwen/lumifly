import { defineStore } from 'pinia'
import { badgeApi } from '../api'

export const useBadgeStore = defineStore('badges', {
  state: () => ({
    friendRequests: 0,
    squareUnread: 0,
  }),
  actions: {
    async refresh() {
      try {
        const data = await badgeApi.unread()
        this.friendRequests = data.friendRequests || 0
        this.squareUnread = data.squareUnread || 0
      } catch {
        /* 忽略：接口异常时保持原值 */
      }
    },
  },
})
