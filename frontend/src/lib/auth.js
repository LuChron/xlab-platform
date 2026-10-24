import { defineStore } from 'pinia'

const TOKEN_KEY = 'xlab_token'

export const useAuth = defineStore('auth', {
  state: () => ({
    token: localStorage.getItem(TOKEN_KEY) || '',
    user: null,
    loading: false,
    error: '',
  }),

  getters: {
    isLoggedIn: (s) => !!s.token && !!s.user,
    isAdmin: (s) => s.user?.role === 'admin',
    isMember: (s) => s.user?.role === 'member',
    roleLabel: (s) => (s.user?.role === 'admin' ? '管理方' : '项目团队'),
  },

  actions: {
    async login(username, password) {
      this.loading = true
      this.error = ''
      try {
        const res = await fetch('/api/auth/login', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ username, password }),
        })
        const body = await res.json().catch(() => ({}))
        if (!res.ok) throw new Error(body.message || '登录失败')
        this.token = body.token
        this.user = body.user
        localStorage.setItem(TOKEN_KEY, body.token)
        return body.user
      } catch (e) {
        this.error = e.message
        throw e
      } finally {
        this.loading = false
      }
    },

    // 用已保存的 token 恢复会话
    async restore() {
      if (!this.token) return false
      try {
        const res = await fetch('/api/auth/me', {
          headers: { Authorization: `Bearer ${this.token}` },
        })
        if (!res.ok) throw new Error('expired')
        this.user = await res.json()
        return true
      } catch {
        this.token = ''
        this.user = null
        localStorage.removeItem(TOKEN_KEY)
        return false
      }
    },

    async logout() {
      try {
        await fetch('/api/auth/logout', {
          method: 'POST',
          headers: { Authorization: `Bearer ${this.token}` },
        })
      } catch {
        /* 忽略网络错误 */
      }
      this.token = ''
      this.user = null
      localStorage.removeItem(TOKEN_KEY)
    },
  },
})
