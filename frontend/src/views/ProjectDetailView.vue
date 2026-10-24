<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useLedger } from '../lib/store'
import { useReveal } from '../lib/reveal'
import { useParallax } from '../lib/parallax'
import {
  PROJECT_STATUS, BORROW_STATUS, EQUIPMENT_STATUS, ACTION_LABEL,
  label, tone, fmtDate, fmtTime, daysBetween, detailText, toInputDate,
} from '../lib/labels'
import StatusChip from '../components/StatusChip.vue'
import Timeline from '../components/Timeline.vue'
import AppModal from '../components/AppModal.vue'
import CatIcon from '../components/CatIcon.vue'

const route = useRoute()
const router = useRouter()
const ledger = useLedger()

useReveal()
useParallax('[data-parallax]')

const project = computed(() => ledger.projectMap[route.params.id])
const spaces = computed(() =>
  project.value ? ledger.spaces.filter((s) => project.value.spaceIds.includes(s.id)) : []
)
const devices = computed(() => (project.value ? ledger.devicesOfProject(project.value.id) : []))
const borrows = computed(() =>
  ledger.borrows
    .filter((b) => b.projectId === route.params.id)
    .sort((a, b) => (b.createdAt || '').localeCompare(a.createdAt || ''))
)
const outstanding = computed(() =>
  borrows.value.filter((b) => ['BORROWED', 'RETURN_REQUESTED', 'APPROVED', 'PENDING'].includes(b.status))
)

const remaining = computed(() => {
  if (!project.value) return 0
  const today = toInputDate()
  return Math.max(0, daysBetween(today, project.value.endDate))
})

const progress = computed(() => {
  if (!project.value) return 0
  const total = daysBetween(project.value.startDate, project.value.endDate) || 1
  const done = daysBetween(project.value.startDate, toInputDate())
  return Math.min(100, Math.round((done / total) * 100))
})

// 链上历史取活动流水（每条带 action 与 detail），而不是项目状态快照
const history = computed(() =>
  ledger.activities
    .filter((a) => a.objectId === route.params.id)
    .map((a) => ({
      txId: a.txId,
      title: ACTION_LABEL[a.action] || a.action || '链上记录',
      detail: detailText(a.detail),
      time: fmtTime(a.timestamp),
    }))
)
// 续期 / 结项 / 成果
const modal = ref('')
const noteText = ref('')
const newEnd = ref('')

function openModal(kind) {
  modal.value = kind
  noteText.value = ''
  if (kind === 'extend' && project.value) {
    const d = new Date(project.value.endDate)
    d.setDate(d.getDate() + 30)
    newEnd.value = toInputDate(d)
  }
}

async function doExtend() {
  if (!newEnd.value || newEnd.value <= project.value.endDate) {
    window.__xlabToast?.('新的结束日期必须晚于当前期限', 'err')
    requestAnimationFrame(() => document.querySelector('#extend-date')?.focus())
    return
  }
  if (!noteText.value.trim()) {
    window.__xlabToast?.('请填写续期理由', 'err')
    requestAnimationFrame(() => document.querySelector('#extend-note')?.focus())
    return
  }
  try {
    await ledger.extendProject(project.value.id, newEnd.value, noteText.value)
    modal.value = ''
    window.__xlabToast?.('项目已续期')
    await ledger.load()
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  }
}

async function doClose() {
  if (!noteText.value.trim()) {
    window.__xlabToast?.('请填写结项说明', 'err')
    requestAnimationFrame(() => document.querySelector('#close-note')?.focus())
    return
  }
  try {
    await ledger.closeProject(project.value.id, noteText.value)
    modal.value = ''
    window.__xlabToast?.('项目已结项，空间已释放')
    await ledger.load()
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  }
}

async function doMilestone() {
  if (!noteText.value.trim()) {
    window.__xlabToast?.('请填写阶段成果', 'err')
    requestAnimationFrame(() => document.querySelector('#milestone-note')?.focus())
    return
  }
  try {
    await ledger.submitMilestone(project.value.id, noteText.value)
    modal.value = ''
    window.__xlabToast?.('阶段成果已上链')
    await ledger.load()
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  }
}

async function doReturnRequest(id) {
  try {
    await ledger.requestReturn(id, '项目结项前归还')
    window.__xlabToast?.('已发起归还，等待管理方验收')
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  }
}
</script>

<template>
  <div v-if="project" class="detail">
    <button class="back" @click="router.back()">← 入驻项目</button>

    <!-- 首屏：谁在这里，占了什么 -->
    <header class="p-hero">
      <div class="p-copy" data-reveal>
        <h1>{{ project.name }}</h1>

        <div class="p-meta">
          <span class="p-code">{{ project.id }}</span>
          <StatusChip :label="label(PROJECT_STATUS, project.status)" :tone="tone(PROJECT_STATUS, project.status)" />
          <span v-if="project.extensions" class="tag">已续期 {{ project.extensions }} 次</span>
        </div>

        <p class="p-sub">{{ project.summary }}</p>

        <ul class="p-tally">
          <li><b>{{ project.leadName }}</b><span>负责人</span></li>
          <li><b>{{ project.memberCount }} 人</b><span>团队规模</span></li>
          <li><b>{{ devices.length }} 台</b><span>在用设备</span></li>
          <li :class="{ warn: remaining <= 7 }"><b>{{ remaining }} 天</b><span>空间使用剩余</span></li>
        </ul>

        <div class="p-actions">
          <button class="btn btn-primary btn-sm" @click="openModal('milestone')" :disabled="project.status !== 'ACTIVE'">
            提交阶段成果
          </button>
          <button class="btn btn-amber btn-sm" @click="openModal('extend')" :disabled="project.status !== 'ACTIVE'">
            申请续期
          </button>
          <button class="btn btn-danger btn-sm" @click="openModal('close')" :disabled="project.status === 'CLOSED'">
            结项退出
          </button>
        </div>
      </div>

      <figure data-parallax class="p-photo" data-reveal data-delay="120">
        <img src="/media/xlab-official-opening.png" alt="浙江大学图书馆 X-Lab 的开放体验空间" width="960" height="720" fetchpriority="high" />
        <figcaption data-ui>浙江大学图书馆 X-Lab · 开放空间</figcaption>
      </figure>
    </header>

    <div class="p-split">
      <!-- 左：占用 + 设备 + 交接 -->
      <div class="p-main">
        <section class="block" data-reveal>
          <div class="block-head">
            <h2>空间与团队</h2>
            <span>{{ spaces.length }} 个空间 · 入驻 {{ fmtDate(project.startDate) }} → {{ fmtDate(project.endDate) }}</span>
          </div>

          <div class="space-list" v-if="spaces.length">
            <div v-for="sp in spaces" :key="sp.id" class="space-item">
              <span class="sp-floor">{{ sp.floor }}</span>
              <div class="sp-main">
                <strong>{{ sp.name }}</strong>
                <span>{{ sp.id }} · {{ sp.zone }} · {{ sp.capacity }} 人</span>
              </div>
              <span class="sp-tag" :class="{ closed: project.status === 'CLOSED' }">
                {{ project.status === 'CLOSED' ? '已释放' : '占用中' }}
              </span>
            </div>
          </div>
          <p v-else class="p-empty">尚未分配空间。</p>

          <div class="team-row">
            <span class="team-label">团队成员</span>
            <div class="team-names">
              <span v-for="m in project.members" :key="m" class="member">{{ m }}</span>
            </div>
          </div>

          <div v-if="project.milestone" class="milestone">
            <span class="ms-label">阶段目标</span>
            <p>{{ project.milestone }}</p>
          </div>
        </section>

        <section class="block" data-ui data-reveal>
          <div class="block-head">
            <h2>关联设备</h2>
            <router-link to="/equipment" class="block-link">申请设备 →</router-link>
          </div>

          <ul class="dev-list" v-if="devices.length">
            <li v-for="(d, i) in devices" :key="d.id" data-reveal :data-delay="i * 50">
              <router-link class="dev" :to="`/equipment/${d.id}`">
                <span class="dev-cat"><CatIcon :category="d.category" /></span>
                <div class="dev-main">
                  <strong>{{ d.name }}</strong>
                  <span>{{ d.id }} · {{ d.location }}</span>
                </div>
                <StatusChip :label="label(EQUIPMENT_STATUS, d.status)" :tone="tone(EQUIPMENT_STATUS, d.status)" />
              </router-link>
            </li>
          </ul>
          <p v-else class="p-empty">暂无关联设备。设备协作页可以本项目名义申请。</p>
        </section>

        <section class="block" data-reveal>
          <div class="block-head">
            <h2>交接记录</h2>
            <span>{{ borrows.length }} 条</span>
          </div>

          <table class="book" v-if="borrows.length">
            <thead>
              <tr>
                <th>设备</th>
                <th>借用人</th>
                <th>到期</th>
                <th>状态</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(b, i) in borrows" :key="b.id" data-reveal :data-delay="i * 50">
                <td>
                  <b class="cell-strong">{{ b.equipmentName }}</b>
                  <span class="cell-sub">{{ b.id }}</span>
                </td>
                <td>{{ b.borrower }}</td>
                <td class="num nowrap">{{ fmtDate(b.dueDate) }}</td>
                <td><StatusChip :label="label(BORROW_STATUS, b.status)" :tone="tone(BORROW_STATUS, b.status)" /></td>
                <td class="cell-act">
                  <button v-if="b.status === 'BORROWED'" class="act" @click="doReturnRequest(b.id)">发起归还</button>
                </td>
              </tr>
            </tbody>
          </table>
          <p v-else class="p-empty">这个项目还没有设备交接记录。</p>
        </section>
      </div>

      <!-- 右：结项检查 + 链上历史 -->
      <aside class="p-side">
        <section class="block" data-reveal data-delay="80">
          <h2>结项检查</h2>
          <dl class="kv">
            <div>
              <dt>未归还设备</dt>
              <dd :class="{ late: outstanding.length > 0 }">{{ outstanding.length }} 件</dd>
            </div>
            <div>
              <dt>空间占用</dt>
              <dd>{{ project.status === 'CLOSED' ? '已释放' : '占用中' }}</dd>
            </div>
            <div>
              <dt>入驻周期</dt>
              <dd>{{ progress }}% 已过</dd>
            </div>
          </dl>
          <p class="side-note">
            结项前必须先归还全部设备；空间释放后，设备责任仍保留在原项目与借用人名下可追溯。
          </p>
        </section>

        <section class="block" data-reveal data-delay="160">
          <div class="block-head">
            <h2>链上历史</h2>
            <span>{{ history.length }} 条来自活动流水</span>
          </div>
          <Timeline :items="history" />
        </section>
      </aside>
    </div>

    <!-- 满幅现场带 -->
    <section data-parallax class="p-strip" data-reveal>
      <figure>
        <img src="/media/xlab-official-message-wall.png" alt="同学在 X-Lab 留言墙记录项目想法" width="960" height="640" loading="lazy" />
        <figcaption data-ui>项目协作 · 创意留言墙</figcaption>
      </figure>
      <figure data-reveal data-delay="90">
        <img src="/media/xlab-official-mr-demo.png" alt="学生在 X-Lab 使用混合现实设备" width="960" height="640" loading="lazy" />
        <figcaption data-ui>项目实践 · MR 交互</figcaption>
      </figure>
      <figure data-reveal data-delay="180">
        <img src="/media/xlab-official-xr-space.png" alt="学生在 X-Lab 混合现实空间浏览数字内容" width="960" height="640" loading="lazy" />
        <figcaption data-ui>项目实践 · 混合现实空间</figcaption>
      </figure>
    </section>


    <!-- 弹窗 -->
    <AppModal v-if="modal === 'milestone'" title="提交阶段成果" subtitle="成果摘要上链，正文与附件留在链下" @close="modal = ''">
      <div class="field">
        <label for="milestone-note">成果说明</label>
        <textarea id="milestone-note" v-model="noteText" name="milestoneNote" autocomplete="off" placeholder="完成了什么、如何验证、下一步计划…" />
      </div>
      <template #foot>
        <button class="btn btn-ghost" @click="modal = ''">取消</button>
        <button class="btn btn-primary" @click="doMilestone">提交上链</button>
      </template>
    </AppModal>

    <AppModal v-if="modal === 'extend'" title="申请续期" subtitle="只延长空间使用期限，设备借期按各自交接单管理" @close="modal = ''">
      <div class="field">
        <label for="extend-date">新的结束日期</label>
        <input id="extend-date" v-model="newEnd" name="newEndDate" type="date" autocomplete="off" />
      </div>
      <div class="field">
        <label for="extend-note">续期理由</label>
        <textarea id="extend-note" v-model="noteText" name="extendNote" autocomplete="off" placeholder="说明当前进展与后续计划…" />
      </div>
      <template #foot>
        <button class="btn btn-ghost" @click="modal = ''">取消</button>
        <button class="btn btn-primary" @click="doExtend">确认续期</button>
      </template>
    </AppModal>

    <AppModal v-if="modal === 'close'" title="结项退出" subtitle="释放全部空间，保留历史记录" @close="modal = ''">
      <div v-if="outstanding.length" class="note-box">
        当前仍有 <b>{{ outstanding.length }}</b> 件设备未归还，必须先完成全部归还才能结项。
      </div>
      <div class="field">
        <label for="close-note">结项说明</label>
        <textarea id="close-note" v-model="noteText" name="closeNote" autocomplete="off" placeholder="项目产出与复盘…" />
      </div>
      <template #foot>
        <button class="btn btn-ghost" @click="modal = ''">取消</button>
        <button class="btn btn-danger" :disabled="outstanding.length > 0" @click="doClose">确认结项</button>
      </template>
    </AppModal>
  </div>

  <div v-else class="empty panel">
    <strong>项目不存在</strong>
    <router-link class="btn btn-sm" style="margin-top: 14px" to="/residency">返回</router-link>
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
.p-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.22fr) minmax(0, 0.74fr);
  gap: 3px;
  margin-bottom: 3px;
}

.p-copy {
  background: var(--surface);
  padding: 34px 38px 38px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.p-copy h1 {
  font-family: var(--serif);
  font-size: clamp(34px, 4.4vw, 66px);
  font-weight: 500;
  letter-spacing: -0.02em;
  line-height: 1.05;
  color: var(--ink);
  max-width: 13em;
}

.p-meta {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: 20px;
}

.p-code {
  font-size: 16px;
  color: var(--blue-deep);
  font-weight: 620;
  font-variant-numeric: tabular-nums;
}

.tag {
  font-size: 14px;
  padding: 5px 12px;
  background: var(--surface-2);
  color: var(--ink-2);
}

.p-sub {
  margin-top: 22px;
  font-size: 16px;
  line-height: 1.72;
  color: var(--ink-2);
  max-width: 46ch;
}

.p-tally {
  list-style: none;
  margin: 30px 0 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 22px;
}

.p-tally li {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 0;
  background: transparent;
}
.p-tally li.warn b,
.p-tally li.warn span { color: var(--warn); }

.p-tally b {
  font-family: var(--serif);
  font-size: 22px;
  font-weight: 500;
  letter-spacing: -0.015em;
  line-height: 1;
  color: var(--ink);
  font-variant-numeric: tabular-nums;
}

.p-tally span { font-size: 14px; color: var(--ink-2); white-space: nowrap; }

.p-actions { display: flex; gap: 10px; margin-top: 28px; flex-wrap: wrap; }

.p-photo { margin: 0; position: relative; overflow: hidden; min-height: 340px; }
.p-photo img { width: 100%; height: 100%; object-fit: cover; display: block; transition: transform 0.8s cubic-bezier(0.16, 1, 0.3, 1); }
.p-photo:hover img { transform: scale(1.04); }
.p-photo::after,
.p-strip figure::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(0deg, rgba(5, 18, 31, 0.7), transparent 42%); /* taste-skill-ignore: image caption scrim */
}
.p-photo figcaption {
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
.p-split {
  display: grid;
  grid-template-columns: minmax(0, 1.42fr) minmax(0, 1fr);
  gap: 3px;
  align-items: start;
}

.p-main, .p-side { display: grid; gap: 3px; min-width: 0; }

/* 网格子项默认 min-width:auto，长串文本会把轨道撑出视口 */
.p-main > *, .p-side > * { min-width: 0; }

/* 右栏比左栏短时，让最后一格长到与左栏齐平：
   否则下方会露出整片灰底，看起来像内容漏了 */
.p-side {
  align-self: stretch;
  grid-template-rows: auto 1fr;
}

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

.block-link { font-size: 16px; font-weight: 600; color: var(--blue-deep); text-decoration: none; }

/* 空间 */
.space-list { display: grid; gap: 2px; }

.space-item {
  display: grid;
  grid-template-columns: 46px minmax(0, 1fr) auto;
  gap: 16px;
  align-items: center;
  background: var(--surface-2);
  padding: 14px 16px;
}

.sp-floor { font-size: 14px; color: var(--ink-3); font-variant-numeric: tabular-nums; }

.sp-tag {
  font-size: 14px;
  padding: 5px 12px;
  background: var(--blue-soft);
  color: var(--blue-deep);
  font-weight: 550;
}
.sp-tag.closed { background: var(--surface-3); color: var(--ink-3); }

.sp-main strong { display: block; font-size: 16px; font-weight: 600; }
.sp-main span { display: block; margin-top: 3px; font-size: 14px; color: var(--ink-3); }

/* 团队 */
.team-row {
  display: grid;
  grid-template-columns: 104px minmax(0, 1fr);
  gap: 16px;
  align-items: baseline;
  margin-top: 3px;
  padding: 16px 16px 18px;
  background: var(--surface-2);
}

.team-label { font-size: 14px; color: var(--ink-3); }
.team-names { display: flex; gap: 6px; flex-wrap: wrap; }

.member {
  font-size: 14px;
  padding: 4px 12px;
  background: var(--surface);
  color: var(--ink);
}

.milestone {
  margin-top: 3px;
  padding: 16px 18px;
  background: var(--blue-soft);
}

.ms-label { font-size: 14px; color: var(--blue-deep); font-weight: 550; }
.milestone p { margin-top: 6px; font-size: 16px; line-height: 1.65; color: var(--ink); }

/* 设备 */
.dev-list { display: grid; gap: 2px; list-style: none; margin: 0; padding: 0; }

.dev {
  display: grid;
  grid-template-columns: 38px minmax(0, 1fr) auto;
  gap: 14px;
  align-items: center;
  padding: 13px 12px;
  background: var(--surface-2);
  color: inherit;
  text-decoration: none;
  cursor: pointer;
  transition: background 0.15s ease;
}
.dev:hover { background: var(--bg); }

.dev-cat {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  background: var(--surface);
  color: var(--blue-deep);
}
.dev-cat svg { width: 18px; height: 18px; }

.dev-main strong { display: block; font-size: 16px; font-weight: 600; }
.dev-main span { display: block; margin-top: 3px; font-size: 14px; color: var(--ink-3); }

/* 表格 */
.book { width: 100%; border-collapse: collapse; font-size: 16px; table-layout: fixed; }
.book th:nth-child(1), .book td:nth-child(1) { width: 36%; }
.book th:nth-child(2), .book td:nth-child(2) { width: 12%; }
.book th:nth-child(3), .book td:nth-child(3) { width: 22%; }
.book th:nth-child(4), .book td:nth-child(4) { width: 20%; }
.book th:nth-child(5), .book td:nth-child(5) { width: 10%; }

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
/* 日期是不可拆的整体：让它换行会把一行撑成三行 */
.nowrap { white-space: nowrap; }
.num { font-variant-numeric: tabular-nums; }
.cell-act { text-align: right; padding-right: 0; }

.act {
  border: 0;
  background: transparent;
  font-family: var(--sans);
  font-size: 14px;
  font-weight: 600;
  color: var(--blue-deep);
  cursor: pointer;
  padding: 0;
}
.act:hover { text-decoration: underline; text-underline-offset: 4px; }

.p-empty { font-size: 16px; color: var(--ink-3); padding: 6px 0; }

/* 右栏 */
.kv { margin: 0; display: grid; gap: 2px; }

.kv > div {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 16px;
  padding: 11px 13px;
  background: var(--surface-2);
  font-size: 16px;
}
.kv dt { color: var(--ink-3); }
.kv dd { margin: 0; color: var(--ink); font-weight: 550; font-variant-numeric: tabular-nums; }
.kv dd.late { color: var(--bad); }

.side-note { margin-top: 16px; font-size: 16px; line-height: 1.72; color: var(--ink-2); }

.note-box {
  margin-bottom: 16px;
  padding: 14px 16px;
  background: #FBE3DF;
  font-size: 14px;
  line-height: 1.65;
  color: var(--ink);
}

/* 满幅现场带 */
.p-strip {
  margin: 3px calc(var(--gutter) * -1) 0;
  width: auto;
  display: grid;
  grid-template-columns: 1.16fr 1fr 1.1fr;
  gap: 3px;
  height: 34vh;
  min-height: 260px;
}

.p-strip figure { margin: 0; position: relative; overflow: hidden; }
.p-strip img,
.p-strip video { width: 100%; height: 100%; object-fit: cover; display: block; }
.p-strip figcaption {
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
  .p-hero { grid-template-columns: 1fr; }
  .p-photo { min-height: 280px; max-height: 360px; }
  .p-copy { padding: 26px 24px 30px; }
  .p-split { grid-template-columns: 1fr; }
  /* 平板宽下左栏三块改成一张白纸，避免读成一排同宽卡片 */
  .p-main { background: var(--surface); padding: 4px 24px 6px; }
  .p-main > .block { background: transparent; padding: 22px 0 24px; }
}

@media (max-width: 720px) {
  .p-copy h1 { font-size: 32px; }
  .block { padding: 20px 18px 24px; }
  .p-tally { gap: 16px; }
  .p-tally b { font-size: 19px; }
  .p-tally span { font-size: 14px; }
  .space-item { grid-template-columns: 38px minmax(0, 1fr); }
  .space-item > :last-child { grid-column: 2; justify-self: start; margin-top: 6px; }
  .team-row { grid-template-columns: 1fr; gap: 10px; }
  .p-strip { grid-template-columns: 1fr 1fr; height: 28vh; min-height: 200px; }
  .p-strip figure:last-child { display: none; }
  /* 手机上交接记录只留设备、到期与状态 */
  .book th:nth-child(2), .book td:nth-child(2) { display: none; }
  .book th:nth-child(5), .book td:nth-child(5) { display: none; }
}
</style>
