<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useLedger } from './lib/store'
import { useAuth } from './lib/auth'
import XLabLogo from './components/XLabLogo.vue'

const ledger = useLedger()
const auth = useAuth()
const route = useRoute()
const router = useRouter()

const toast = ref(null)
const menuOpen = ref(false)
const progress = ref(0)


// 阅读进度：滚动位置换算成 0-1，顶部一条 3px 的线跟着走
let ticking = false
const readProgress = () => {
  ticking = false
  const doc = document.documentElement
  const max = doc.scrollHeight - window.innerHeight
  progress.value = max > 0 ? Math.min(1, window.scrollY / max) : 0
}

const onScroll = () => {
  if (ticking) return
  ticking = true
  requestAnimationFrame(readProgress)
}

onMounted(() => {
  window.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('resize', onScroll)
  readProgress()
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', onScroll)
  window.removeEventListener('resize', onScroll)
})

const isLogin = computed(() => route.name === 'login')
const backlog = computed(() => ledger.backlogCount)

// 顶部导航：平铺全部入口，用克制的状态标记维持资源工作台的连续感
const navItems = computed(() => {
  const items = [
    { name: 'home', path: '/', label: '总览', icon: 'grid' },
    { name: 'equipment', path: '/equipment', label: '设备协作', icon: 'box' },
    { name: 'residency', path: '/residency', label: '项目与空间', icon: 'map' },
  ]
  if (auth.isAdmin) {
    items.push({
      name: 'console',
      path: '/console',
      label: '管理台',
      icon: 'inbox',
      badge: backlog.value,
    })
  }
  items.push({ name: 'chain', path: '/chain', label: '链上记录', icon: 'link' })
  return items
})

function isActive(item) {
  if (route.name === item.name) return true
  if (item.name === 'equipment' && route.name === 'equipment-detail') return true
  if (item.name === 'residency' && route.name === 'project-detail') return true
  return false
}

let timer
function notify(message, kind = 'ok') {
  toast.value = { message, kind }
  clearTimeout(timer)
  timer = setTimeout(() => (toast.value = null), 3400)
}
window.__xlabToast = notify

async function doLogout() {
  await auth.logout()
  menuOpen.value = false
  router.push('/login')
}

async function retryLedger() {
  ledger.loading = true
  await ledger.load()
}

watch(
  () => [isLogin.value, auth.isLoggedIn],
  async ([login, loggedIn]) => {
    if (!login && ledger.equipment.length === 0 && !ledger.loading) {
      ledger.loading = true
      await ledger.load()
    }
  },
  { immediate: true }
)
</script>

<template>
  <router-view v-if="isLogin" />

  <div v-else class="shell">
	<a class="skip-link" href="#main-content">跳到主要内容</a>
    <!-- ===== 顶部导航：深色吸顶工作栏，品牌在左、入口居中、账号在右 ===== -->
    <header class="topbar">
      <router-link to="/" class="brand">
        <span class="brand-mark" aria-hidden="true"><XLabLogo /></span>
        <span class="brand-text">
          <b>X-Lab</b>
          <i>资源协作平台</i>
        </span>
      </router-link>

      <nav class="topnav" aria-label="主导航">
        <router-link
          v-for="item in navItems"
          :key="item.name"
          :to="item.path"
          class="nav-item"
          :class="{ on: isActive(item) }"
        >
          <span class="ico" aria-hidden="true">
            <svg v-if="item.icon === 'grid'" viewBox="0 0 20 20" fill="none">
              <rect x="3" y="3" width="6" height="6" rx="1.6" fill="currentColor" />
              <rect x="11" y="3" width="6" height="6" rx="1.6" fill="currentColor" opacity=".38" />
              <rect x="3" y="11" width="6" height="6" rx="1.6" fill="currentColor" opacity=".38" />
              <rect x="11" y="11" width="6" height="6" rx="1.6" fill="currentColor" />
            </svg>
            <svg v-else-if="item.icon === 'box'" viewBox="0 0 20 20" fill="none">
              <path d="M10 2.6 17 6.4v7.2L10 17.4 3 13.6V6.4L10 2.6Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" />
              <path d="M3.2 6.5 10 10.2l6.8-3.7M10 10.2v7" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" />
            </svg>
            <svg v-else-if="item.icon === 'map'" viewBox="0 0 20 20" fill="none">
              <path d="M3 5.4 7.6 3.4v13.2L3 18V5.4ZM7.6 3.4 12.4 5.6v13L7.6 16.6V3.4ZM12.4 5.6 17 3.6V16.8l-4.6 1.8V5.6Z" stroke="currentColor" stroke-width="1.4" stroke-linejoin="round" />
            </svg>
            <svg v-else-if="item.icon === 'inbox'" viewBox="0 0 20 20" fill="none">
              <path d="M3 11.5 5.2 4.6h9.6L17 11.5v4.4a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1v-4.4Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" />
              <path d="M3 11.5h4l1 2h4l1-2h4" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" />
            </svg>
            <svg v-else viewBox="0 0 20 20" fill="none">
              <path d="M8 11.4 11.4 8M7.2 9.2 5.6 10.8a2.9 2.9 0 0 0 4.1 4.1l1.6-1.6M12.8 10.8l1.6-1.6a2.9 2.9 0 0 0-4.1-4.1L8.7 6.7" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
            </svg>
          </span>
          <span class="nav-label">{{ item.label }}</span>
          <span v-if="item.badge > 0" class="nav-badge">{{ item.badge }}</span>
        </router-link>
      </nav>

      <div class="topbar-right">
        <div class="chain-state">
          <span class="dot" :class="{ live: ledger.mode === 'fabric' }"></span>
          <span class="chain-text">
            {{ ledger.mode === 'fabric' ? 'Fabric 已连接' : '演示模式' }}
          </span>
        </div>

        <router-link v-if="!auth.isLoggedIn" to="/login" class="nav-login">登录</router-link>

        <div v-else class="account-wrap">
          <button
			class="account"
			type="button"
			:aria-expanded="menuOpen"
			aria-controls="account-menu"
			@click="menuOpen = !menuOpen"
		  >
            <span class="av">{{ auth.user?.role === 'admin' ? '管' : '团' }}</span>
            <span class="who">
              <b>{{ auth.user?.displayName }}</b>
              <i>{{ auth.user?.role === 'admin' ? '可审核与分配' : '可申请与借用' }}</i>
            </span>
          </button>

          <div v-if="menuOpen" id="account-menu" class="account-menu" @click="menuOpen = false">
            <div class="menu-who">
              <b>{{ auth.user?.displayName }}</b>
              <span>{{ auth.user?.title }}</span>
            </div>
            <div class="perm">
              <div class="perm-row"><span>提交入驻申请</span><b>可以</b></div>
              <div class="perm-row"><span>申请借用设备</span><b>可以</b></div>
              <div class="perm-row">
                <span>评审与分配空间</span><b>{{ auth.isAdmin ? '可以' : '不可以' }}</b>
              </div>
              <div class="perm-row">
                <span>审批与交接确认</span><b>{{ auth.isAdmin ? '可以' : '不可以' }}</b>
              </div>
            </div>
            <button class="logout" type="button" @click.stop="doLogout">退出登录</button>
          </div>
        </div>
      </div>
    </header>

    <main id="main-content" class="stage" tabindex="-1">
      <div class="read-bar" aria-hidden="true"><i :style="{ transform: `scaleX(${progress})` }" /></div>
      <section v-if="ledger.error" class="ledger-error" role="alert">
        <div>
          <b>账本暂时不可用</b>
          <span>{{ ledger.error }}</span>
        </div>
        <button class="btn btn-sm" type="button" :disabled="ledger.loading" @click="retryLedger">
          {{ ledger.loading ? '正在重试…' : '重新连接' }}
        </button>
      </section>
      <router-view v-slot="{ Component }">
        <transition name="view" mode="out-in">
          <component :is="Component" />
        </transition>
      </router-view>
    </main>

    <div class="toast-wrap" aria-live="polite" aria-atomic="true">
      <div v-if="toast" class="toast" :class="toast.kind">
        <svg v-if="toast.kind === 'err'" viewBox="0 0 20 20" width="15" height="15" fill="none" aria-hidden="true">
          <path d="M10 3 18 17H2L10 3Z" stroke="currentColor" stroke-width="1.5" stroke-linejoin="round" />
          <path d="M10 8v4M10 14.2v.1" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
        </svg>
        <svg v-else viewBox="0 0 20 20" width="15" height="15" fill="none" aria-hidden="true">
          <path d="M4 10.5 8 14.5 16 6" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" />
        </svg>
        <span>{{ toast.message }}</span>
      </div>
    </div>
  </div>
</template>
