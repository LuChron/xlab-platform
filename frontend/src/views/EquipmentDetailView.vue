<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useLedger } from '../lib/store'
import { useReveal } from '../lib/reveal'
import { useParallax } from '../lib/parallax'
import {
  EQUIPMENT_STATUS, EQUIPMENT_CONDITION, BORROW_STATUS, ACTION_LABEL,
  label, tone, fmtDate, fmtTime, daysBetween, detailText, whoIs, toInputDate,
} from '../lib/labels'
import StatusChip from '../components/StatusChip.vue'
import Timeline from '../components/Timeline.vue'
import CatIcon from '../components/CatIcon.vue'

const route = useRoute()
const router = useRouter()
const ledger = useLedger()

useReveal()
useParallax('[data-parallax]')

const eq = computed(() => ledger.equipmentMap[route.params.id])

const relatedBorrows = computed(() =>
  ledger.borrows
    .filter((b) => b.equipmentId === route.params.id)
    .sort((a, b) => (b.createdAt || '').localeCompare(a.createdAt || ''))
)

const currentBorrow = computed(() =>
  relatedBorrows.value.find((b) => ['BORROWED', 'RETURN_REQUESTED', 'APPROVED', 'PENDING', 'EXCEPTION'].includes(b.status))
)
// 设备状态非空闲但账本里查不到在借单：说明借用单明细只对管理方与当事团队可见，
// 不能显示"当前空闲"造成误导。
const borrowHidden = computed(() =>
  !!eq.value && ['BORROWED', 'RETURN_REQUESTED', 'APPROVED', 'PENDING', 'REQUESTED'].includes(eq.value.status) && !currentBorrow.value
)
const isAvailable = computed(() => eq.value?.status === 'AVAILABLE')
const currentProject = computed(() =>
  eq.value?.projectId ? ledger.projectMap[eq.value.projectId] : null
)

// 设备类别 → 现场照片：让详情页也能看到这件东西真实的样子与世界
const SCENE = {
  计算设备: { src: '/media/xlab-official-xr-space.png', cap: 'X-Lab · 混合现实空间', width: 960, height: 640 },
  存储设备: { src: '/media/xlab-official-opening.png', cap: 'X-Lab · XR 设备空间', width: 960, height: 720 },
  影音设备: { src: '/media/xlab-official-vr-demo.png', cap: 'X-Lab · VR 设备指导', width: 960, height: 720 },
  展示设备: { src: '/media/xlab-official-experience.png', cap: 'X-Lab · 沉浸式展示', width: 960, height: 640 },
  工程设备: { src: '/media/xlab-official-mr-demo.png', cap: 'X-Lab · MR 交互设备', width: 960, height: 640 },
  协作工具: { src: '/media/xlab-official-message-wall.png', cap: 'X-Lab · 创意留言墙', width: 960, height: 640 },
}
const scene = computed(() => SCENE[eq.value?.category] || {
  src: '/media/xlab-official-opening.png', cap: '浙江大学图书馆 X-Lab', width: 960, height: 720,
})

// 链上变更历史取自活动流水（logActivity 写入），而不是 GetHistoryForKey 的状态快照：
// 快照只有「某个时刻的完整对象」，读不出发生过什么，时间轴上就只能写「链上记录」。
// 借用类事件的 objectId 是借用单号，所以要先把这台设备的借用单号并进来。
const history = computed(() => {
  const ids = new Set([route.params.id, ...relatedBorrows.value.map((b) => b.id)])
  return ledger.activities
    .filter((a) => ids.has(a.objectId))
    .map((a) => ({
      txId: a.txId,
      title: ACTION_LABEL[a.action] || a.action || '链上记录',
      detail: detailText(a.detail),
      time: fmtTime(a.timestamp),
    }))
})
const loadingHistory = computed(() => ledger.loading && !history.value.length)

const dueLeft = computed(() => {
  if (!currentBorrow.value) return null
  return daysBetween(toInputDate(), currentBorrow.value.dueDate)
})
</script>

<template>
  <div v-if="eq" class="detail">
    <button class="back" @click="router.back()">← 设备清单</button>

    <!-- 首屏：这一件设备的状态与实物 -->
    <header class="d-hero">
      <div class="d-copy" data-reveal>
        <h1>{{ eq.name }}</h1>

        <div class="d-meta">
          <span class="d-code">{{ eq.id }}</span>
          <span class="d-cat"><CatIcon :category="eq.category" />{{ eq.category }}</span>
          <StatusChip :label="label(EQUIPMENT_STATUS, eq.status)" :tone="tone(EQUIPMENT_STATUS, eq.status)" />
        </div>

        <p class="d-lede">
          {{ eq.brand }} {{ eq.model }}，存放在{{ eq.location }}，由{{ eq.custodian }}保管。借出、归还和验收记录可在本页查看。
        </p>

        <ul class="d-tally">
          <li><b>{{ eq.totalBorrows }}</b><span>次累计借出</span></li>
          <li><b>{{ relatedBorrows.filter((b) => b.status === 'RETURNED').length }}</b><span>次已验收归还</span></li>
          <li><b>{{ EQUIPMENT_CONDITION[eq.condition] || eq.condition }}</b><span>当前成色</span></li>
          <li :class="{ warn: !isAvailable }">
            <b>{{ isAvailable ? '空闲' : label(EQUIPMENT_STATUS, eq.status) }}</b>
            <span>{{ currentBorrow ? currentBorrow.borrower : (isAvailable ? '可提交申请' : '借用明细仅限当事方可见') }}</span>
          </li>
        </ul>

        <div class="d-actions">
          <router-link class="d-cta" to="/equipment">
            {{ isAvailable ? '去设备目录申请借用' : '这台暂不可借，去看别的' }}
          </router-link>
          <router-link class="d-alt" :to="`/chain`">这台设备的链上记录</router-link>
        </div>
      </div>

      <figure data-parallax class="d-photo" data-reveal data-delay="120">
        <img :src="scene.src" :alt="`${eq.category}现场`" :width="scene.width" :height="scene.height" fetchpriority="high" />
        <figcaption data-ui>{{ scene.cap }}</figcaption>
      </figure>
    </header>

    <div class="d-split">
      <!-- 左：规格 + 记录 -->
      <div class="d-main">
        <section class="block" data-ui data-reveal>
          <h2>规格与保管</h2>
          <dl class="spec">
            <div><dt>品牌</dt><dd>{{ eq.brand }}</dd></div>
            <div><dt>型号</dt><dd>{{ eq.model }}</dd></div>
            <div><dt>序列号</dt><dd class="num">{{ eq.serialNo }}</dd></div>
            <div><dt>存放位置</dt><dd>{{ eq.location }}</dd></div>
            <div><dt>保管责任人</dt><dd>{{ eq.custodian }}</dd></div>
            <div><dt>随附配件</dt><dd>{{ eq.accessories?.length ? eq.accessories.join(' · ') : '无' }}</dd></div>
          </dl>
          <p v-if="eq.note" class="d-note">{{ eq.note }}</p>
        </section>

        <section class="block" data-ui data-reveal>
          <div class="block-head">
            <h2>交接记录</h2>
            <span>{{ relatedBorrows.length }} 条</span>
          </div>

          <div v-if="relatedBorrows.length" data-ui>
          <table class="book">
            <thead>
              <tr>
                <th>借用人</th>
                <th>项目</th>
                <th>到期</th>
                <th>状态</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="b in relatedBorrows" :key="b.id">
                <td>
                  <b class="cell-strong">{{ b.borrower }}</b>
                  <span class="cell-sub">{{ b.id }}</span>
                </td>
                <td><span class="cell-mini">{{ b.projectId || '个人' }}</span></td>
                <td class="num nowrap">{{ fmtDate(b.dueDate) }} 到期</td>
                <td><StatusChip :label="label(BORROW_STATUS, b.status)" :tone="tone(BORROW_STATUS, b.status)" /></td>
              </tr>
            </tbody>
          </table>
          </div>
          <p v-else class="d-empty">这台设备还没有被借出过。第一条记录会出现在第一次交接时。</p>
        </section>

        <section class="block" data-reveal>
          <div class="block-head">
            <h2>链上变更历史</h2>
            <span>审核与交接流水</span>
          </div>
          <div v-if="loadingHistory" class="skeleton" style="height: 80px" />
          <Timeline v-else :items="history" />
        </section>
      </div>

      <!-- 右：当前责任 -->
      <aside class="d-side">
        <section class="block" data-reveal data-delay="80">
          <h2>当前责任</h2>

          <template v-if="currentBorrow">
            <div class="chips">
              <StatusChip :label="label(BORROW_STATUS, currentBorrow.status)" :tone="tone(BORROW_STATUS, currentBorrow.status)" />
              <span v-if="currentBorrow.overdue" class="tag bad">已逾期</span>
            </div>

            <dl class="kv">
              <div><dt>借用单</dt><dd>{{ currentBorrow.id }}</dd></div>
              <div><dt>借用人</dt><dd>{{ currentBorrow.borrower }}</dd></div>
              <div><dt>所属项目</dt><dd>{{ currentProject?.name || currentBorrow.projectId || '个人借用' }}</dd></div>
              <div><dt>预计归还</dt><dd>{{ fmtDate(currentBorrow.dueDate) }}</dd></div>
              <div v-if="dueLeft !== null">
                <dt>剩余</dt>
                <dd :class="{ late: dueLeft <= 3 }">{{ dueLeft }} 天</dd>
              </div>
              <div><dt>用途</dt><dd>{{ currentBorrow.purpose }}</dd></div>
            </dl>

            <router-link v-if="currentProject" :to="`/projects/${currentProject.id}`" class="side-link">
              查看所属项目 →
            </router-link>
          </template>

          <div v-else-if="borrowHidden" class="free">
            <b>借用中</b>
            <span>设备当前有未结清的借用单，明细仅对管理方与当事团队可见。</span>
          </div>

          <div v-else class="free">
            <b>当前空闲</b>
            <span>可由入驻项目申请借用，一次借用对应一条链上凭证。</span>
          </div>
        </section>

        <section class="block" data-reveal data-delay="160">
          <h2>责任链</h2>
          <p class="side-note">
            设备状态变更只允许由资源管理委员会（Org1）发起审核与交接确认，项目团队（Org2）只能提交申请与发起归还。链码层校验组织身份，越权调用会被直接拒绝。
          </p>
          <dl class="kv">
            <div><dt>最后更新</dt><dd>{{ fmtTime(eq.updatedAt) }}</dd></div>
            <div><dt>更新者</dt><dd>{{ whoIs(eq.updatedBy) }}</dd></div>
          </dl>
        </section>

        <figure class="d-side-photo" data-reveal data-delay="220">
          <img src="/media/xlab-official-experience.png" alt="学生在 X-Lab 进行沉浸式绘画创作" width="960" height="640" loading="lazy" />
          <figcaption data-ui>X-Lab · 沉浸式创作</figcaption>
        </figure>
      </aside>
    </div>

    <!-- 收束：这件设备所在的现场 -->
    <section data-parallax class="d-strip" data-reveal>
      <figure>
        <img src="/media/xlab-official-opening.png" alt="浙江大学图书馆 X-Lab 的开放体验空间" width="960" height="720" loading="lazy" />
        <figcaption data-ui>X-Lab · 开放体验空间</figcaption>
      </figure>
      <figure data-reveal data-delay="90">
        <img src="/media/xlab-official-vr-demo.png" alt="X-Lab 工作人员指导学生使用 VR 设备" width="960" height="720" loading="lazy" />
        <figcaption data-ui>X-Lab · VR 设备指导</figcaption>
      </figure>
      <figure data-reveal data-delay="180">
        <img src="/media/xlab-official-mr-demo.png" alt="学生在 X-Lab 使用混合现实头显与控制器" width="960" height="640" loading="lazy" />
        <figcaption data-ui>X-Lab · MR 交互设备</figcaption>
      </figure>
    </section>
  </div>

  <div v-else class="empty panel">
    <strong>设备不存在</strong>
    <router-link class="btn btn-sm" style="margin-top: 14px" to="/equipment">返回设备清单</router-link>
  </div>
</template>

<style scoped>
.back {
  border: 0;
  background: transparent;
  font-family: var(--sans);
  font-size: 14px;
  color: var(--ink-3);
  padding: 0;
  margin-bottom: 22px;
  cursor: pointer;
}
.back:hover { color: var(--ink); }

/* ---------- 首屏 ---------- */
.d-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.18fr) minmax(0, 0.78fr);
  gap: 3px;
  align-items: stretch;
  margin-bottom: 3px;
}

.d-copy { background: var(--surface); padding: 34px 38px 38px; display: flex; flex-direction: column; justify-content: center; }

.d-copy h1 {
  font-family: var(--serif);
  font-size: clamp(34px, 4.4vw, 66px);
  font-weight: 500;
  letter-spacing: -0.02em;
  line-height: 1.05;
  color: var(--ink);
  max-width: 12em;
}

.d-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 20px;
}

.d-code {
  font-size: 16px;
  color: var(--blue-deep);
  font-weight: 620;
  font-variant-numeric: tabular-nums;
}

.d-cat {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font-size: 16px;
  color: var(--ink-2);
  padding-left: 12px;
}
.d-cat svg { width: 17px; height: 17px; color: var(--blue-deep); }

.d-lede {
  margin-top: 22px;
  font-size: 16px;
  line-height: 1.72;
  color: var(--ink-2);
  max-width: 44ch;
}

.d-tally {
  list-style: none;
  margin: 32px 0 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 22px;
}

.d-tally li {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 0;
  background: transparent;
}
.d-tally li.warn b,
.d-tally li.warn span { color: var(--ok); }

.d-actions {
  margin-top: 28px;
  display: flex;
  align-items: center;
  gap: 24px;
  flex-wrap: wrap;
}

.d-cta {
  border: 0;
  background: var(--ink);
  color: #fff;
  font-family: var(--sans);
  font-size: 16px;
  font-weight: 550;
  padding: 14px 24px;
  min-height: 48px;
  border-radius: 10px;
  cursor: pointer;
  text-decoration: none;
  transition: background 0.16s ease;
}

.d-cta:hover:not(:disabled) { background: var(--blue); }
.d-cta:disabled { background: var(--surface-3); color: var(--ink-3); cursor: default; }

.d-alt {
  font-size: 16px;
  font-weight: 550;
  color: var(--ink-2);
  text-decoration: none;
  transition: color 0.16s ease;
}

.d-alt:hover { color: var(--blue-deep); }

.d-tally b {
  font-family: var(--serif);
  font-size: 22px;
  font-weight: 500;
  letter-spacing: -0.015em;
  color: var(--ink);
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.d-tally span { font-size: 14px; color: var(--ink-2); white-space: nowrap; }

.d-photo { margin: 0; position: relative; overflow: hidden; min-height: 340px; border-radius: 0 18px 18px 0; }
.d-photo img { width: 100%; height: 100%; object-fit: cover; display: block; transition: transform 0.8s cubic-bezier(0.16, 1, 0.3, 1); }
.d-photo:hover img { transform: scale(1.04); }

.d-photo::after,
.d-side-photo::after,
.d-strip figure::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(0deg, rgba(5, 18, 31, 0.7), transparent 42%); /* taste-skill-ignore: image caption scrim */
}

.d-photo figcaption {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2;
  padding: 0 16px 14px;
  font-size: 14px;
  color: #fff;
  background: transparent;
}

/* ---------- 主体 ---------- */
.d-split {
  display: grid;
  grid-template-columns: minmax(0, 1.52fr) minmax(0, 1fr);
  gap: 3px;
  align-items: start;
}

.d-main, .d-side { display: grid; gap: 3px; }

/* 左栏是一整张白纸：块自身不上底色，靠留白分节，避免读成一叠同宽卡片 */
.d-main { background: var(--surface); padding: 4px 28px 8px; }
.d-main > .block { background: transparent; padding: 24px 0 28px; }

.block { background: var(--surface); padding: 26px 28px 30px; }

.block h2 {
  font-family: var(--serif);
  font-size: 24px;
  font-weight: 500;
  letter-spacing: -0.015em;
  margin-bottom: 18px;
}

.block-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}
.block-head h2 { margin-bottom: 0; }
.block-head span { font-size: 14px; color: var(--ink-3); }

.spec {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 2px;
  margin: 0;
}

.spec > div { background: var(--surface-2); padding: 13px 15px 15px; }

.spec dt { font-size: 14px; color: var(--ink-3); }
.spec dd {
  margin: 4px 0 0;
  font-size: 16px;
  color: var(--ink);
  font-weight: 500;
  line-height: 1.5;
}
.spec .num { font-variant-numeric: tabular-nums; }

.d-note {
  margin-top: 18px;
  padding: 14px 16px;
  background: var(--sun-soft);
  font-size: 14px;
  line-height: 1.65;
  color: var(--ink-2);
}

/* 表格：整棵子树豁免细线 */
.book { width: 100%; border-collapse: collapse; font-size: 16px; table-layout: fixed; }

.book th {
  text-align: left;
  padding: 12px 0;
  font-size: 14px;
  font-weight: 550;
  color: var(--ink-3);
  background: var(--bg);
  white-space: nowrap;
}

.book th:first-child { padding-left: 14px; }
.book td:first-child { padding-left: 14px; }

.book td {
  padding: 15px 12px 15px 0;
  border-top: 1px solid var(--surface-2);
  color: var(--ink-2);
}

.cell-strong { display: block; color: var(--ink); font-weight: 550; }
.cell-sub { display: block; margin-top: 3px; font-size: 13px; color: var(--ink-3); font-variant-numeric: tabular-nums; }
/* 日期是不可拆的整体 */
.nowrap { white-space: nowrap; }
/* 项目编号是长串：小一号并允许在任意处断行，否则会顶到日期列上 */
.cell-mini { font-size: 14px; color: var(--ink-3); overflow-wrap: anywhere; }
.book th:nth-child(1), .book td:nth-child(1) { width: 34%; }
.book th:nth-child(2), .book td:nth-child(2) { width: 24%; }
.book th:nth-child(3), .book td:nth-child(3) { width: 23%; }
.book th:nth-child(4), .book td:nth-child(4) { width: 19%; }
.num { font-variant-numeric: tabular-nums; }

.d-empty { font-size: 16px; color: var(--ink-3); padding: 8px 0 4px; }

.chips { display: flex; gap: 10px; flex-wrap: wrap; margin-bottom: 16px; }

.kv { margin: 0; display: grid; gap: 2px; }

.kv > div {
  display: grid;
  grid-template-columns: 88px minmax(0, 1fr);
  gap: 16px;
  padding: 11px 13px;
  background: var(--surface-2);
  font-size: 16px;
}

.kv dt { color: var(--ink-3); }
.kv dd { margin: 0; color: var(--ink); text-align: right; line-height: 1.5; }
.kv dd.late { color: var(--bad); font-weight: 600; }
.break { word-break: break-all; }

.tag {
  font-size: 14px;
  padding: 5px 12px;
  background: var(--surface-2);
  color: var(--ink-2);
}
.tag.bad { background: #FBE3DF; color: var(--bad); }

.side-link {
  display: block;
  margin-top: 18px;
  font-size: 16px;
  font-weight: 600;
  color: var(--blue-deep);
  text-decoration: none;
}

.free { padding: 4px 0 2px; }
.free b { display: block; font-size: 22px; color: var(--mint-deep); letter-spacing: -0.02em; }
.free span { display: block; margin-top: 8px; font-size: 16px; color: var(--ink-2); line-height: 1.6; }

.side-note { font-size: 16px; line-height: 1.72; color: var(--ink-2); margin-bottom: 14px; }

.d-side-photo { margin: 0; position: relative; overflow: hidden; min-height: 220px; }
.d-side-photo video { width: 100%; height: 100%; object-fit: cover; display: block; }
.d-side-photo figcaption {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2;
  padding: 0 16px 14px;
  font-size: 14px;
  color: #fff;
  background: transparent;
}

/* 满幅现场带 */
.d-strip {
  margin: 3px calc(var(--gutter) * -1) 0;
  width: auto;
  display: grid;
  grid-template-columns: 1.2fr 1fr 1.06fr;
  gap: 3px;
  height: 34vh;
  min-height: 260px;
}

.d-strip figure { margin: 0; position: relative; overflow: hidden; }
.d-strip img,
.d-strip video { width: 100%; height: 100%; object-fit: cover; display: block; }
.d-strip figcaption {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2;
  padding: 0 16px 14px;
  font-size: 14px;
  color: #fff;
  background: transparent;
}

@media (max-width: 1180px) {
  .d-hero { grid-template-columns: 1fr; }
  .d-strip { grid-template-columns: 1fr 1fr; height: 30vh; }
  .d-strip figure:last-child { display: none; }
  .d-photo { min-height: 280px; max-height: 360px; }
  .d-copy { padding: 26px 24px 30px; }
  .d-split { grid-template-columns: 1fr; }
  .spec { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 720px) {
  .d-copy h1 { font-size: 32px; }
  .spec { grid-template-columns: 1fr; }
  .block { padding: 20px 18px 24px; }
  .kv > div { grid-template-columns: 78px minmax(0, 1fr); }
  .d-tally { gap: 16px; }
  .d-tally b { font-size: 19px; }
  .d-tally span { font-size: 14px; }
  .book th:nth-child(3), .book td:nth-child(3) { display: none; }
}
</style>
