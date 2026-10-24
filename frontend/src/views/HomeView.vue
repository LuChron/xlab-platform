<script setup>
import { computed } from 'vue'
import { useLedger } from '../lib/store'
import { usePointerTilt } from '../lib/pointer'
import { useParallax } from '../lib/parallax'
import CountUp from '../components/CountUp.vue'
import { ACTION_LABEL, fmtTime, shortHash } from '../lib/labels'
import './home.css'

usePointerTilt()
useParallax('[data-parallax]')

const ledger = useLedger()

const s = computed(() => ledger.stats || {})

const utilization = computed(() => {
  const total = s.value.equipmentTotal || 0
  if (!total) return 0
  return Math.round(((s.value.equipmentOut || 0) / total) * 100)
})

// 首屏右侧的信号行：数字与说明同一行
// 数值与单位分开存：CountUp 要拿到数字才能从 0 滚上去
const signals = computed(() => [
  { n: s.value.equipmentAvailable ?? 0, suffix: '', t: '设备可借' },
  { n: s.value.spaceFree ?? 0, suffix: '', t: '空间可用' },
  { n: ledger.pendingApplications.length + ledger.pendingBorrows.length, suffix: '', t: '等待处理' },
])

// 首屏的界面切片：真实可借设备
const shelf = computed(() =>
  ledger.equipment
    .filter((e) => e.status === 'AVAILABLE')
    .slice(0, 4)
    .map((e) => ({
      id: e.id,
      name: e.name,
      where: e.location || '未来学习中心',
    }))
)


const projects = computed(() =>
  ledger.activeProjects.slice(0, 4).map((p) => {
    const home = ledger.spaces.find((x) => p.spaceIds?.includes(x.id))
    return {
      id: p.id,
      name: p.name,
      lead: p.leadName || '暂无',
      where: home ? `${home.id} · ${home.name}` : '待分配',
      devices: ledger.devicesOfProject(p.id).length,
      until: (p.endDate || '').slice(5).replace('-', '/') || '暂无',
    }
  })
)

// 左卡下方的补充：现在空着、随时可以申请的空间
const freeSpaces = computed(() =>
  ledger.spaces
    .filter((x) => x.status === 'OPEN' && !ledger.occupantOfSpace(x.id))
    .slice(0, 6)
    .map((x) => ({ id: x.id, name: x.name, cap: x.capacity })),
)

const feed = computed(() =>
  ledger.activities.slice(0, 7).map((a) => ({
    tx: a.txId,
    what: ACTION_LABEL[a.action] || a.action,
    who: a.actorMsp === 'Org1MSP' ? '委员会' : '团队',
    at: fmtTime(a.timestamp),
  }))
)

// 设备构成：按类别数一遍，给首屏右侧一条分布图
const CAT_ORDER = ['计算设备', '影音设备', '显示设备', '工程设备', '协作工具', '存储设备', '展示设备']
const dist = computed(() => {
  const n = {}
  for (const e of ledger.equipment) n[e.category] = (n[e.category] || 0) + 1
  const rows = CAT_ORDER.filter((k) => n[k]).map((k) => ({ k, n: n[k] }))
  const max = Math.max(1, ...rows.map((r) => r.n))
  return rows.map((r) => ({ ...r, w: Math.round((r.n / max) * 100) }))
})

// 环形仪表的周长（r = 42），用来把百分比换算成 dash 长度
const RING = 2 * Math.PI * 42
const ringOffset = computed(() => RING * (1 - utilization.value / 100))

// 结算带上浮卡：最近一次交接的真实记录
const latest = computed(() => {
  const a = ledger.activities[0]
  if (!a) return null
  return {
    what: ACTION_LABEL[a.action] || a.action,
    tx: shortHash(a.txId, 6, 4),
    who: a.actorMsp === 'Org1MSP' ? '资源管理委员会' : '入驻项目团队',
    at: fmtTime(a.timestamp),
  }
})

const live = computed(() => ledger.mode === 'fabric')
</script>

<template>
  <div class="homeview">
    <!-- 首屏 -->
    <section class="hero">
      <div class="hero-copy">
        <h1>项目、空间与设备</h1>
        <p class="hero-lede">
          在 X-Lab 申请项目入驻、使用空间和借还设备。审批与交接记录写入 Fabric 账本。
        </p>
        <div class="hero-actions">
          <router-link to="/residency" class="btn btn-primary">申请项目入驻</router-link>
          <router-link to="/equipment" class="btn btn-ghost">查看设备</router-link>
        </div>

        <p class="hero-quiet">{{ live ? `已连接 Fabric · ${ledger.channel || 'xlabchannel'}` : '当前使用演示账本' }}</p>

      </div>

      <aside class="hero-side">
        <p class="hero-side-title">资源概况</p>
        <ul class="signals">
          <li v-for="(g, i) in signals" :key="i" class="signal" data-reveal :data-delay="120 + i * 70">
            <b><CountUp :to="g.n" :suffix="g.suffix" /></b>
            <span>{{ g.t }}</span>
          </li>
        </ul>

        <div class="dist" data-reveal data-delay="300">
          <b class="dist-title">设备构成</b>
          <ul>
            <li v-for="(r, i) in dist" :key="r.k" data-reveal :data-delay="320 + i * 45">
              <span class="dist-k">{{ r.k }}</span>
              <span class="dist-bar"><i :style="{ width: r.w + '%' }" /></span>
              <span class="dist-n">{{ r.n }}</span>
            </li>
          </ul>
        </div>
      </aside>

      <div class="shelf" data-ui data-reveal data-delay="180">
        <div class="shelf-top">
          <b>可借设备</b>
          <span>当前共 {{ s.equipmentAvailable ?? 0 }} 件</span>
          <router-link to="/equipment" class="shelf-more">查看全部设备</router-link>
        </div>
        <ul>
          <li v-for="(d, i) in shelf" :key="d.id" data-reveal data-tilt :data-delay="200 + i * 50">
            <span class="nm">{{ d.name }}</span>
            <span class="wh">{{ d.where }}</span>
            <span class="tg">可借</span>
          </li>
        </ul>
      </div>
    </section>

    <section class="portals" aria-label="资源协作入口" data-reveal>
      <router-link to="/residency" class="portal portal-space">
        <span class="portal-copy">
          <strong>项目入驻与空间</strong>
          <span>填写团队、项目周期和空间需求，评审通过后分配工位。</span>
        </span>
        <span class="portal-action">查看空间 <b aria-hidden="true">↗</b></span>
      </router-link>
      <router-link to="/equipment" class="portal portal-equipment">
        <span class="portal-copy">
          <strong>设备借用与交接</strong>
          <span>查看设备状态和存放位置，申请后完成领取与归还验收。</span>
        </span>
        <span class="portal-action">查看设备 <b aria-hidden="true">↗</b></span>
      </router-link>
    </section>

    <!-- 实景带：一张大图压浮卡，右侧两张叠放 -->
    <section class="band" data-reveal>
      <figure class="b1" data-reveal data-tilt data-parallax :data-delay="0">
        <img src="/media/xlab-official-opening.png" alt="浙江大学图书馆 X-Lab 内，多位同学正在体验 XR 设备" width="960" height="720" loading="lazy" />
        <figcaption data-ui>浙江大学图书馆 X-Lab · XR 空间</figcaption>
      </figure>

      <div class="band-side">
        <figure data-reveal data-parallax :data-delay="100">
          <img src="/media/xlab-official-experience.png" alt="学生在 X-Lab 进行沉浸式绘画创作" width="960" height="640" loading="lazy" />
          <figcaption data-ui>X-Lab · 沉浸式创作</figcaption>
        </figure>
        <figure data-reveal data-parallax :data-delay="200">
          <img src="/media/xlab-official-message-wall.png" alt="同学在 X-Lab 留言墙上记录体验反馈" width="960" height="640" loading="lazy" />
          <figcaption data-ui>X-Lab · 创意留言墙</figcaption>
        </figure>
      </div>

      <figure class="b3" data-reveal :data-delay="260">
        <img src="/media/xlab-official-xr-space.png" alt="学生在 X-Lab 使用混合现实头显浏览数字内容" width="960" height="640" loading="lazy" />
        <figcaption data-ui>X-Lab · 混合现实空间</figcaption>
      </figure>

      <div v-if="latest" class="band-card" data-ui data-reveal data-delay="320">
        <span class="bc-tag"><i class="bc-dot" aria-hidden="true"></i>最新记录</span>
        <b>{{ latest.what }}</b>
        <dl>
          <div><dt>调用方</dt><dd>{{ latest.who }}</dd></div>
          <div><dt>时间</dt><dd>{{ latest.at }}</dd></div>
          <div><dt>交易</dt><dd class="bc-tx">{{ latest.tx }}</dd></div>
        </dl>
      </div>
    </section>

    <!-- 在驻项目与链上流水 -->
    <section class="split">
      <div class="card" data-reveal data-tilt>
        <div class="card-head" data-reveal>
          <h2>在驻项目</h2>
          <router-link to="/residency">全部 {{ ledger.activeProjects.length }} 个 →</router-link>
        </div>
        <ul class="rows">
          <li v-for="(p, i) in projects" :key="p.id" data-reveal :data-delay="i * 60">
            <router-link :to="`/projects/${p.id}`" class="row">
              <span class="row-name">
                <b>{{ p.name }}</b>
                <i>{{ p.lead }} · 至 {{ p.until }}</i>
              </span>
              <span class="row-where">{{ p.where }}</span>
              <span class="row-dev">{{ p.devices }} 台设备</span>
            </router-link>
          </li>
          <li v-if="!projects.length" class="rows-empty">
            还没有在驻项目。提交一份入驻申请，委员会评审后分配空间。
          </li>
        </ul>

        <div v-if="freeSpaces.length" class="spaces-now">
          <p class="spaces-title">现在空着的空间 <span>{{ s.spaceFree ?? freeSpaces.length }} 处</span></p>
          <ul>
            <li v-for="x in freeSpaces" :key="x.id">
              <b>{{ x.id }}</b>
              <span>{{ x.name }}</span>
              <i>{{ x.cap }} 人</i>
            </li>
          </ul>
        </div>
      </div>

      <div class="card" data-reveal data-delay="90">
        <div class="card-head" data-reveal data-delay="60">
          <h2>最近操作</h2>
          <router-link to="/chain">全部 {{ ledger.activities.length }} 笔 →</router-link>
        </div>
        <ul class="feed">
          <li v-for="(a, i) in feed" :key="a.tx" data-reveal :data-delay="i * 50">
            <span class="feed-what">{{ a.what }}</span>
            <span class="feed-who">{{ a.who }}</span>
            <span class="feed-tx">{{ shortHash(a.tx, 8, 4) }}</span>
            <span class="feed-at">{{ a.at }}</span>
          </li>
          <li v-if="!feed.length" class="rows-empty">
            完成一次业务操作后，这里会出现链上轨迹。
          </li>
        </ul>
        <p class="feed-live">
          <span class="live-dot" :class="{ on: live }" />
          {{ live ? `已连接 Fabric · ${ledger.channel || 'xlabchannel'}` : '演示账本（未上链）' }}
        </p>
      </div>
    </section>

    <div class="home-end">
      <footer class="foot">
        <p class="foot-name">X-Lab 资源协作平台</p>
        <p class="foot-where">浙江大学启真交叉学科创新创业实验室 · 未来学习中心 3F / 4F</p>
      </footer>
    </div>
  </div>
</template>
