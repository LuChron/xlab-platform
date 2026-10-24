<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuth } from '../lib/auth'
import { useLedger } from '../lib/store'
import { ACTION_LABEL, fmtDate } from '../lib/labels'
import XLabLogo from '../components/XLabLogo.vue'

const router = useRouter()
const auth = useAuth()
const ledger = useLedger()

const username = ref('admin')
const password = ref('xlab2026')
const err = ref('')
const submitting = ref(false)

const accounts = [
  {
    key: 'admin',
    username: 'admin',
    password: 'xlab2026',
    title: '资源管理委员会',
    subtitle: 'X-Lab · Org1',
    desc: '处理入驻申请、空间分配和设备借还',
    scope: ['评审项目', '分配工位', '审批借用', '交接验收'],
  },
  {
    key: 'team',
    username: 'team',
    password: 'xlab2026',
    title: '入驻项目团队',
    subtitle: '项目方 · Org2',
    desc: '提交入驻申请，以项目名义借用和归还设备',
    scope: ['提交入驻', '申请设备', '提交成果', '发起归还'],
  },
]

// 轮播：实验室实景。每张带一句说明，切换时与底部小字同步
const slides = [
  {
    src: '/media/xlab-official-opening.png',
    cap: '浙江大学图书馆 X-Lab · XR 空间',
    short: 'X-Lab 空间',
    alt: '浙江大学图书馆 X-Lab 内，多位同学正在体验 XR 设备',
    kind: 'photo',
  },
  {
    src: '/media/xlab-official-vr-demo.png',
    cap: 'X-Lab · VR 设备指导',
    short: 'VR 设备',
    alt: 'X-Lab 工作人员指导学生使用 VR 头显与手柄',
    kind: 'photo',
  },
  {
    src: '/media/xlab-official-mr-demo.png',
    cap: 'X-Lab · MR 交互体验',
    short: 'MR 交互',
    alt: '学生在 X-Lab 使用混合现实头显和控制器',
    kind: 'photo',
  },
  {
    src: '/media/xlab-official-experience.png',
    cap: 'X-Lab · 沉浸式创作',
    short: '沉浸创作',
    alt: '学生在 X-Lab 通过头显进行沉浸式绘画创作',
    kind: 'photo',
  },
  {
    src: '/media/xlab-official-message-wall.png',
    cap: 'X-Lab · 体验留言墙',
    short: '留言墙',
    alt: '同学在 X-Lab 的白板留言墙上记录体验反馈',
    kind: 'photo',
  },
]

const slide = ref(0)
const current = computed(() => slides[slide.value])

// 主图旁边常驻三张缩略图：取当前之后的三张，循环
const thumbs = computed(() =>
  [1, 2, 3].map((n) => {
    const i = (slide.value + n) % slides.length
    return { ...slides[i], index: i }
  }),
)
function go(i) {
  slide.value = (i + slides.length) % slides.length
}

function pick(acc) {
  username.value = acc.username
  password.value = acc.password
  err.value = ''
}

async function submit() {
  err.value = ''
  submitting.value = true
  try {
    await auth.login(username.value, password.value)
    router.push('/')
  } catch (e) {
    err.value = e.message
  } finally {
    submitting.value = false
  }
}

onMounted(async () => {
  // 已有有效会话则直接进入
  if (await auth.restore()) router.push('/')
  // 登录页也读一次公开账本：让访客在进来之前就看到链上真实在发生什么
  if (!ledger.activities.length) await ledger.load()
})
</script>

<template>
  <div class="login-stage">
    <!-- 左栏：居中的一句话 + 轮播 -->
    <main class="login-panel">
      <header class="panel-top">
        <span class="brand-logo" aria-hidden="true"><XLabLogo /></span>
        <span class="brand-name">X-Lab 资源协作平台</span>
      </header>

      <div class="panel-center">
        <h1>登录 X-Lab</h1>
        <a class="panel-cta" href="#login-form">登录</a>

        <p class="panel-desc">
          管理方处理入驻和借用，项目团队提交申请并归还设备。审批、领取和验收记录会写入联盟链。
        </p>

        <!-- 图片轮播：一张主图缓慢切换，右侧三张缩略图始终在场 -->
        <div class="carousel" aria-roledescription="carousel">
          <div class="shots">
            <div class="shot-main">
              <figure :key="slide" class="shot">
                <img :src="current.src" :alt="current.alt" width="960" height="720" />
                <figcaption data-ui>{{ current.cap }}</figcaption>
              </figure>
            </div>

            <div class="shot-thumbs" role="tablist" aria-label="实景切换">
              <button
                v-for="t in thumbs"
                :key="t.src"
                type="button"
                class="thumb"
                :class="{ on: t.index === slide }"
                role="tab"
                :aria-selected="t.index === slide"
                :aria-label="t.cap"
                @click="go(t.index)"
              >
                <img :src="t.src" :alt="t.alt" width="960" height="720" />
                <span>{{ t.short }}</span>
              </button>
            </div>
          </div>

          <div class="carousel-foot">
            <div class="dots" role="group" aria-label="选择实景">
              <button v-for="(s, i) in slides" :key="s.src" type="button" class="dot" :class="{ on: i === slide }" :aria-label="`查看${s.short}`" @click="go(i)" />
            </div>
          </div>
        </div>
      </div>

      <footer class="panel-foot">
        浙江大学启真交叉学科创新创业实验室 · 未来学习中心 3F / 4F
      </footer>
    </main>

    <!-- 右栏：登录 -->
    <aside class="login-box">
      <div class="box-inner">
        <header class="login-head">
          <h2>登录</h2>
          <p>选择身份，或直接输入账号密码</p>
        </header>

        <div class="role-grid">
          <button
            v-for="acc in accounts"
            :key="acc.key"
            type="button"
            class="role-card"
            :class="{ active: username === acc.username }"
            @click="pick(acc)"
          >
            <div class="role-head">
              <span class="role-avatar" :class="acc.key">{{ acc.key === 'admin' ? '管' : '团' }}</span>
              <span class="role-org">{{ acc.subtitle }}</span>
            </div>
            <strong>{{ acc.title }}</strong>
            <p>{{ acc.desc }}</p>
            <ul>
              <li v-for="s in acc.scope" :key="s">{{ s }}</li>
            </ul>
          </button>
        </div>

        <form id="login-form" @submit.prevent="submit">
          <div class="field">
            <label for="login-user">账号</label>
            <input id="login-user" v-model="username" name="username" autocomplete="username" spellcheck="false" placeholder="例如：admin 或 team…" />
          </div>
          <div class="field">
            <label for="login-pass">密码</label>
            <input
              id="login-pass"
              v-model="password"
              name="password"
              type="password"
              autocomplete="current-password"
              placeholder="演示密码：xlab2026…"
            />
          </div>

          <p v-if="err" class="login-err" role="alert">{{ err }}</p>

          <button class="btn btn-primary login-submit" type="submit" :disabled="submitting">
            {{ submitting ? '登录中…' : '进入平台' }}
          </button>
        </form>

        <div class="proof">
          <div class="proof-head">
            <b>链上正在发生</b>
            <span>{{ ledger.activities.length }} 笔 · {{ ledger.channel || 'xlabchannel' }}</span>
          </div>
          <ul class="proof-list" data-ui>
            <li v-for="a in ledger.activities.slice(0, 3)" :key="a.txId">
              <span class="proof-time">{{ fmtDate(a.timestamp) }}</span>
              <span class="proof-act">{{ ACTION_LABEL[a.action] || a.action }}</span>
              <span class="proof-obj">{{ a.objectId }}</span>
            </li>
            <li v-if="!ledger.activities.length" class="proof-empty">读不到链上记录 · 请确认 Fabric 网络已启动</li>
          </ul>
        </div>

        <p class="login-hint">演示账号 admin / team，密码均为 xlab2026</p>
      </div>
    </aside>
  </div>
</template>

<style scoped>
.login-stage {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 468px;
  min-height: 100dvh;
  background: var(--bg);
}

/* ---------- 左栏：一张白纸上居中一句话，下面浮着轮播 ---------- */
.login-panel {
  display: flex;
  flex-direction: column;
  padding: 34px 56px 26px;
  min-width: 0;
}

.panel-top {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}

.brand-logo { width: 26px; height: 26px; display: grid; place-items: center; }
.brand-logo svg { width: 100%; height: 100%; }

.brand-name {
  font-size: 16px;
  font-weight: 620;
  color: var(--ink);
  letter-spacing: -0.02em;
}

.panel-center {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 34px 0 20px;
  min-height: 0;
}

.login-panel h1 {
  font-family: var(--serif);
  font-size: clamp(40px, 4.6vw, 68px);
  line-height: 1.06;
  letter-spacing: -0.03em;
  font-weight: 500;
  color: var(--ink);
  max-width: none;
  white-space: nowrap;
}

.panel-desc {
  margin-top: 26px;
  font-size: 16px;
  line-height: 1.78;
  color: var(--ink-3);
  max-width: 40em;
}

/* 窄屏首屏的落地按钮：桌面端隐藏，登录卡本身就在首屏 */
.panel-cta {
  display: none;
  margin-top: 22px;
}

/* 只做一次轻微就位，不改变透明度；避免登录页加载时整块闪现。 */
@keyframes login-settle {
  from { transform: translateY(8px); }
  to { transform: none; }
}

@media (prefers-reduced-motion: no-preference) {
  .panel-top,
  .login-panel h1,
  .panel-desc,
  .carousel,
  .login-head,
  .role-card,
  form,
  .proof {
    animation: login-settle 0.38s cubic-bezier(0.22, 1, 0.36, 1) both;
  }
  .login-panel h1 { animation-delay: 30ms; }
  .panel-desc { animation-delay: 60ms; }
  .carousel { animation-delay: 90ms; }
  .login-head { animation-delay: 40ms; }
  .role-card:nth-child(1) { animation-delay: 70ms; }
  .role-card:nth-child(2) { animation-delay: 100ms; }
  form { animation-delay: 130ms; }
  .proof { animation-delay: 160ms; }
}

/* ---------- 轮播 ---------- */
.carousel { margin-top: 34px; max-width: 840px; }

.shots {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 184px;
  gap: 8px;
}

.shot-main {
  position: relative;
  aspect-ratio: 16 / 9.4;
  background: var(--surface-2);
  border-radius: 18px;
  overflow: hidden;
}

.shot {
  position: absolute;
  inset: 0;
  margin: 0;
}

.shot { z-index: 1; }

.shot img,
.shot video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.shot::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(0deg, rgba(5, 18, 31, 0.7), transparent 36%); /* taste-skill-ignore: image caption scrim */
}

.shot figcaption {
  position: absolute;
  z-index: 3;
  left: 18px;
  bottom: 16px;
  right: 18px;
  background: transparent;
  color: #fff;
  font-size: 14px;
  font-weight: 500;
  padding: 0;
  line-height: 1.4;
}

/* 右侧常驻三张缩略图 */
.shot-thumbs {
  display: grid;
  grid-template-rows: repeat(3, 1fr);
  gap: 8px;
}

.thumb {
  position: relative;
  border: 0;
  padding: 0;
  border-radius: 14px;
  overflow: hidden;
  cursor: pointer;
  background: var(--surface-2);
  font-family: var(--sans);
}

.thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform 0.6s cubic-bezier(0.16, 1, 0.3, 1);
}

.thumb::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(0deg, rgba(5, 18, 31, 0.68), transparent 52%); /* taste-skill-ignore: thumbnail caption scrim */
}

@media (hover: hover) and (pointer: fine) {
  .thumb:hover img { transform: scale(1.025); }
}

.thumb span {
  position: absolute;
  z-index: 2;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 14px 9px 7px;
  font-size: 13px;
  font-weight: 560;
  color: #fff;
  text-align: left;
}

.thumb.on { outline: 2px solid var(--ink); outline-offset: -2px; }

.carousel-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-top: 16px;
}

.dots { display: flex; align-items: center; gap: 8px; }

.dot {
  width: 7px;
  height: 7px;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: var(--surface-3);
  cursor: pointer;
  transition: width 0.28s ease, background 0.2s ease;
}

.dot.on { width: 26px; background: var(--ink); }

.count {
  font-size: 14px;
  color: var(--ink-3);
  text-align: right;
}

.panel-foot {
  font-size: 14px;
  color: var(--ink-3);
  flex-shrink: 0;
}

/* ---------- 右栏：登录卡 ---------- */
.login-box {
  background: var(--surface);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 34px 40px;
}

.box-inner { width: 100%; max-width: 348px; display: grid; gap: 20px; }

.login-head h2 { font-family: var(--serif); font-size: 30px; font-weight: 500; letter-spacing: -0.02em; color: var(--ink); }
.login-head p { margin-top: 6px; font-size: 14px; color: var(--ink-3); }

.role-grid { display: grid; gap: 8px; }

.role-card {
  text-align: left;
  border-radius: 14px;
  background: var(--surface-2);
  padding: 14px 16px 15px;
  cursor: pointer;
  font-family: var(--sans);
  transition: background 0.16s ease;
}

.role-card:hover { background: var(--surface-3); }
.role-card.active { background: var(--ink); }

.role-head { display: flex; align-items: center; gap: 8px; }

.role-avatar {
  width: 22px;
  height: 22px;
  border-radius: 7px;
  display: grid;
  place-items: center;
  font-size: 13px;
  font-weight: 650;
  color: #fff;
  flex-shrink: 0;
}

.role-avatar.admin { background: var(--ink); }
.role-avatar.team { background: var(--mint-deep); }

.role-org { font-size: 14px; color: var(--ink-3); }

.role-card strong {
  display: block;
  margin-top: 9px;
  font-size: 16px;
  font-weight: 620;
  color: var(--ink);
  letter-spacing: -0.02em;
}

.role-card p { margin-top: 4px; font-size: 14px; line-height: 1.6; color: var(--ink-2); }

.role-card ul {
  list-style: none;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 11px;
}

.role-card li {
  font-size: 13px;
  color: var(--blue-deep);
  background: var(--surface);
  padding: 4px 9px;
  border-radius: 8px;
}

.role-card.active strong { color: #fff; }
.role-card.active .role-org { color: rgba(255, 255, 255, 0.72); }
.role-card.active p { color: rgba(255, 255, 255, 0.82); }
.role-card.active li { color: #fff; background: rgba(255, 255, 255, 0.14); }
.role-card.active .role-avatar.admin { background: var(--blue); }

form { display: grid; gap: 12px; }

.login-err { font-size: 14px; color: var(--bad); }

.login-submit { width: 100%; padding: 14px; font-size: 16px; }

.proof { display: grid; gap: 8px; }

.proof-head { display: flex; align-items: baseline; justify-content: space-between; gap: 12px; }
.proof-head b { font-size: 14px; font-weight: 620; color: var(--ink); }
.proof-head span { font-size: 13px; color: var(--ink-3); }

.proof-list {
  list-style: none;
  background: var(--bg);
  border-radius: 14px;
  padding: 10px 14px;
}

.proof-list li {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 10px;
  align-items: baseline;
  padding: 6px 0;
  font-size: 13px;
}



.proof-time { color: var(--ink-3); font-variant-numeric: tabular-nums; white-space: nowrap; }
.proof-act { color: var(--ink); font-weight: 550; }
.proof-obj { color: var(--ink-3); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.proof-empty { color: var(--ink-3); grid-template-columns: 1fr; }

.login-hint { font-size: 13px; color: var(--ink-3); line-height: 1.6; }

/* ---------- 响应式 ---------- */
@media (max-width: 1180px) {
  .login-stage { grid-template-columns: minmax(0, 1fr) 404px; }
  .login-panel { padding: 28px 36px 22px; }
  .login-box { padding: 28px 26px; }
}

@media (max-width: 900px) {
  .login-stage { grid-template-columns: 1fr; }
  .login-panel { padding: 24px 22px 8px; min-height: auto; }
  .panel-center { padding: 26px 0 12px; }
  .carousel { margin-top: 24px; }
  .shot-main { aspect-ratio: 16 / 10.4; border-radius: 16px; }
  .shots { grid-template-columns: minmax(0, 1fr) 96px; gap: 6px; }
  .shot-thumbs { gap: 6px; }
  .thumb { border-radius: 11px; }
  .thumb span { display: none; }
  .panel-foot { padding-bottom: 18px; }
  .panel-cta {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: fit-content;
    min-height: 48px;
    padding: 0 26px;
    border-radius: 10px;
    background: var(--blue);
    color: #fff;
    font-size: 16px;
    font-weight: 600;
    text-decoration: none;
  }
  .login-box { padding: 8px 22px 34px; align-items: flex-start; }
  .box-inner { max-width: none; }
}
</style>
