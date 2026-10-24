import { createApp } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import { useAuth } from './lib/auth'
import './styles/main.css'

import LoginView from './views/LoginView.vue'
import HomeView from './views/HomeView.vue'
import EquipmentView from './views/EquipmentView.vue'
import EquipmentDetailView from './views/EquipmentDetailView.vue'
import ResidencyView from './views/ResidencyView.vue'
import ProjectDetailView from './views/ProjectDetailView.vue'
import ConsoleView from './views/ConsoleView.vue'
import ChainView from './views/ChainView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', name: 'login', component: LoginView, meta: { title: '登录', public: true } },
    // 只读页面对访客开放：链上数据本就可公开核验，先看再登录
    { path: '/', name: 'home', component: HomeView, meta: { title: '总览', public: true } },
    { path: '/equipment', name: 'equipment', component: EquipmentView, meta: { title: '设备协作', public: true } },
    { path: '/equipment/:id', name: 'equipment-detail', component: EquipmentDetailView, meta: { title: '设备详情', public: true } },
    { path: '/residency', name: 'residency', component: ResidencyView, meta: { title: '项目入驻与空间', public: true } },
    { path: '/projects/:id', name: 'project-detail', component: ProjectDetailView, meta: { title: '项目详情', public: true } },
    { path: '/console', name: 'console', component: ConsoleView, meta: { title: '管理台', adminOnly: true } },
    { path: '/chain', name: 'chain', component: ChainView, meta: { title: '链上浏览', public: true } },
  ],
  scrollBehavior(to, from, savedPosition) {
    // 返回/前进时恢复浏览器记住的位置。
    if (savedPosition) return savedPosition

    // 楼层、筛选、搜索和标签页会通过 query 同步到 URL。
    // 这些仍是同一页内的操作，不应打断当前阅读位置。
    if (to.path === from.path && to.hash === from.hash) return false

    // 显式锚点导航应优先到达目标；真正切换页面时才回到顶部。
    if (to.hash) return { el: to.hash }
    return { top: 0 }
  },
})

const app = createApp(App)
app.use(createPinia())

// 全局路由守卫：只读页面对访客开放；写操作与登录页按身份拦截
router.beforeEach(async (to) => {
  const auth = useAuth()

  if (!auth.isLoggedIn && autoToken()) await auth.restore()

  // 已登录时不再停在登录页
  if (to.name === 'login') return auth.isLoggedIn ? { path: '/' } : true

  if (to.meta.public) return true

  // 只读预览：设计验收工具要在未登录时量测管理台排版；写操作仍由后端按身份拒绝
  const preview = to.meta.adminOnly && to.query.preview === '1'

  if (!auth.isLoggedIn && !preview) {
    return { path: '/login', query: { redirect: to.fullPath } }
  }

  if (to.meta.adminOnly && !auth.isAdmin && !preview) {
    return { path: '/' }
  }
  return true
})

function autoToken() {
  return !!localStorage.getItem('xlab_token')
}

app.use(router)
app.mount('#app')
