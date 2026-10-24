import { defineStore } from 'pinia'
import { useAuth } from './auth'

async function api(path, options = {}) {
  const auth = useAuth()
  const headers = { 'Content-Type': 'application/json', ...(options.headers || {}) }
  if (auth.token) headers.Authorization = `Bearer ${auth.token}`

  const res = await fetch(`/api${path}`, { ...options, headers })
  const body = await res.json().catch(() => ({}))

  if (res.status === 401) {
    // 会话失效：清理本地状态，让路由回到登录页
    auth.token = ''
    auth.user = null
    localStorage.removeItem('xlab_token')
    throw new Error(body.message || '登录已过期，请重新登录')
  }
  if (!res.ok) throw new Error(body.message || `请求失败 (${res.status})`)
  return body
}

export const useLedger = defineStore('ledger', {
  state: () => ({
    equipment: [],
    spaces: [],
    projects: [],
    applications: [],
    borrows: [],
    allocations: [],
    activities: [],
    stats: {},
    mode: 'demo',
    channel: '',
    chaincode: '',
    // 真实链状态：高度 + 最近区块哈希链（qscc 读取），链上记录页据此展示真实出块
    chain: null,
    // 初始为 false：App 的加载守卫是「有数据 / 正在加载」两个条件，若初始为 true 将永远不会触发首拉
    loading: false,
    error: '',
  }),

  getters: {
    equipmentMap: (s) => Object.fromEntries(s.equipment.map((e) => [e.id, e])),
    spaceMap: (s) => Object.fromEntries(s.spaces.map((x) => [x.id, x])),
    projectMap: (s) => Object.fromEntries(s.projects.map((p) => [p.id, p])),
    borrowMap: (s) => Object.fromEntries(s.borrows.map((b) => [b.id, b])),

    activeProjects: (s) => s.projects.filter((p) => p.status === 'ACTIVE'),
    pendingApplications: (s) => s.applications.filter((a) => a.status === 'SUBMITTED'),
    approvedApplications: (s) => s.applications.filter((a) => a.status === 'APPROVED'),
    pendingBorrows: (s) => s.borrows.filter((b) => b.status === 'PENDING'),
    approvedBorrows: (s) => s.borrows.filter((b) => b.status === 'APPROVED'),
    borrowedOut: (s) => s.borrows.filter((b) => b.status === 'BORROWED'),
    returnPending: (s) => s.borrows.filter((b) => b.status === 'RETURN_REQUESTED'),
    exceptions: (s) => s.borrows.filter((b) => b.status === 'EXCEPTION'),

    backlogCount: (s) =>
      s.applications.filter((a) => a.status === 'SUBMITTED').length +
      s.borrows.filter((b) => b.status === 'PENDING').length +
      s.borrows.filter((b) => b.status === 'RETURN_REQUESTED').length,

    devicesOfProject: (s) => (projectId) =>
      s.equipment.filter((e) => e.projectId === projectId),

    occupantOfSpace: (s) => (spaceId) =>
      s.allocations.find((a) => a.spaceId === spaceId && a.status === 'ACTIVE'),
  },

  actions: {
    async load() {
      try {
        const data = await api('/ledger')
        this.equipment = data.equipment || []
        this.spaces = data.spaces || []
        this.projects = data.projects || []
        this.applications = data.applications || []
        this.borrows = data.borrows || []
        this.allocations = data.allocations || []
        this.activities = data.activities || []
        this.stats = data.stats || {}
        this.mode = data.mode || 'demo'
        this.channel = data.channel || ''
        this.chaincode = data.chaincode || ''
        this.error = ''
      } catch (e) {
        this.error = e.message
      } finally {
        this.loading = false
      }

      // 链状态独立拉取：失败不阻断主账本，链上记录页据 mode 决定是否展示
      if (this.mode === 'fabric') {
        try {
          this.chain = await api('/chain')
        } catch (e) {
          this.chain = null
        }
      }
    },

    async createApplication(input) {
      const r = await api('/applications', { method: 'POST', body: JSON.stringify(input) })
      await this.load()
      return r
    },

    async reviewApplication(id, approved, note) {
      await api(`/applications/${id}/review`, {
        method: 'POST',
        body: JSON.stringify({ approved, note }),
      })
      await this.load()
    },

    async activateProject(appId, spaceIds) {
      const r = await api(`/applications/${appId}/activate`, {
        method: 'POST',
        body: JSON.stringify({ spaceIds }),
      })
      await this.load()
      return r
    },

    async extendProject(id, newEndDate, note) {
      await api(`/projects/${id}/extend`, {
        method: 'POST',
        body: JSON.stringify({ newEndDate, note }),
      })
      await this.load()
    },

    async closeProject(id, note) {
      await api(`/projects/${id}/close`, {
        method: 'POST',
        body: JSON.stringify({ note }),
      })
      await this.load()
    },

    async submitMilestone(id, note) {
      await api(`/projects/${id}/milestone`, {
        method: 'POST',
        body: JSON.stringify({ note }),
      })
      await this.load()
    },

    async requestBorrow(input) {
      const r = await api('/borrows', { method: 'POST', body: JSON.stringify(input) })
      await this.load()
      return r
    },

    async reviewBorrow(id, approved, note) {
      await api(`/borrows/${id}/review`, {
        method: 'POST',
        body: JSON.stringify({ approved, note }),
      })
      await this.load()
    },

    async confirmPickup(id, condition, photos) {
      await api(`/borrows/${id}/pickup`, {
        method: 'POST',
        body: JSON.stringify({ condition, photos }),
      })
      await this.load()
    },

    async requestReturn(id, note) {
      await api(`/borrows/${id}/return-request`, {
        method: 'POST',
        body: JSON.stringify({ note }),
      })
      await this.load()
    },

    async confirmReturn(id, condition, note, photos) {
      await api(`/borrows/${id}/return-confirm`, {
        method: 'POST',
        body: JSON.stringify({ condition, note, photos }),
      })
      await this.load()
    },

    async resolveException(id, note) {
      await api(`/borrows/${id}/resolve`, {
        method: 'POST',
        body: JSON.stringify({ note }),
      })
      await this.load()
    },

    async history(type, id) {
      return api(`/history/${type}/${id}`)
    },
  },
})
