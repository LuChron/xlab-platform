<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useLedger } from '../lib/store'
import { useReveal } from '../lib/reveal'
import { usePointerTilt } from '../lib/pointer'
import { useParallax } from '../lib/parallax'
import CountUp from '../components/CountUp.vue'
import { EQUIPMENT_STATUS, label, tone, toInputDate } from '../lib/labels'
import StatusChip from '../components/StatusChip.vue'
import AppModal from '../components/AppModal.vue'
import { scrollElementIntoView } from '../lib/motion'

const ledger = useLedger()
const router = useRouter()
const route = useRoute()

useReveal()
usePointerTilt()
useParallax('[data-parallax]')

const validStatuses = new Set(['ALL', 'AVAILABLE', 'BORROWED', 'MAINTENANCE'])
const q = ref(typeof route.query.q === 'string' ? route.query.q : '')
const status = ref(validStatuses.has(route.query.status) ? route.query.status : 'ALL')

watch([q, status], ([nextQ, nextStatus]) => {
  const query = { ...route.query }
  if (nextQ) query.q = nextQ
  else delete query.q
  if (nextStatus !== 'ALL') query.status = nextStatus
  else delete query.status
  router.replace({ query })
})

// 按类别分组：清单按「类别成册」组织，而不是一堆同宽卡片
const groups = computed(() => {
  const map = new Map()
  for (const e of ledger.equipment.filter(match)) {
    if (!map.has(e.category)) map.set(e.category, [])
    map.get(e.category).push(e)
  }
  return [...map.entries()].map(([name, items]) => ({ name, items }))
})

function match(e) {
  const text = `${e.id} ${e.name} ${e.brand} ${e.model} ${e.category} ${e.location}`.toLowerCase()
  const okQ = text.includes(q.value.toLowerCase())
  const okS = status.value === 'ALL' || e.status === status.value
  return okQ && okS
}

const filtered = computed(() => ledger.equipment.filter(match))

// 借用中的设备列在谁手上：编号太长为断行难看，直接说项目名
function projectOf(e) {
  return e.projectId ? ledger.projectMap[e.projectId]?.name || e.projectId : ''
}

const counts = computed(() => {
  const c = { ALL: ledger.equipment.length }
  for (const e of ledger.equipment) c[e.status] = (c[e.status] || 0) + 1
  return c
})

const filters = [
  { key: 'ALL', label: '全部' },
  { key: 'AVAILABLE', label: '可借用' },
  { key: 'BORROWED', label: '借用中' },
  { key: 'MAINTENANCE', label: '维护中' },
]

const showBorrow = ref(false)
const borrowTarget = ref(null)
const form = ref({ projectId: '', borrower: '赵同学', purpose: '', dueDate: '' })
const submitting = ref(false)

// 首屏的主行动：把视线直接带到第一册可借设备上
function jumpToAvailable() {
  const first = document.querySelector('.rows tbody tr')
  scrollElementIntoView(first, 'center')
}

function openBorrow(eq) {
  borrowTarget.value = eq
  const d = new Date()
  d.setDate(d.getDate() + 14)
  form.value = {
    projectId: ledger.activeProjects[0]?.id || '',
    borrower: '赵同学',
    purpose: '',
    dueDate: toInputDate(d),
  }
  showBorrow.value = true
}

async function submitBorrow() {
  if (!form.value.borrower.trim()) {
    window.__xlabToast?.('请填写借用人', 'err')
    requestAnimationFrame(() => document.querySelector('#borrow-person')?.focus())
    return
  }
  if (!form.value.purpose.trim()) {
    window.__xlabToast?.('请填写使用用途', 'err')
    requestAnimationFrame(() => document.querySelector('#borrow-purpose')?.focus())
    return
  }
  if (!form.value.dueDate || form.value.dueDate < toInputDate()) {
    window.__xlabToast?.('归还日期不能早于今天', 'err')
    requestAnimationFrame(() => document.querySelector('#borrow-due')?.focus())
    return
  }
  submitting.value = true
  try {
    await ledger.requestBorrow({
      equipmentId: borrowTarget.value.id,
      projectId: form.value.projectId,
      borrower: form.value.borrower,
      purpose: form.value.purpose,
      dueDate: form.value.dueDate,
    })
    showBorrow.value = false
    window.__xlabToast?.('借用申请已提交，等待管理方审批')
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="eqview">
    <!-- 首屏：标题与账目在左，实物在右 -->
    <header class="eq-hero">
      <div class="eq-copy" data-reveal>
        <h1>设备借还</h1>
        <p class="eq-sub">
          这里列出设备状态、存放位置和保管人。申请通过后领取；归还时由双方确认设备状况。
        </p>
        <ul class="eq-tally">
          <li>
            <b><CountUp :to="ledger.stats.equipmentAvailable || 0" /></b>
            <span>台可借，共 {{ ledger.equipment.length }} 件</span>
          </li>
          <li>
            <b><CountUp :to="ledger.stats.equipmentOut || 0" /></b>
            <span>台借出中</span>
          </li>
          <li>
            <b><CountUp :to="ledger.stats.equipmentMaintenance || 0" /></b>
            <span>台维护中</span>
          </li>
          <li class="warn">
            <b><CountUp :to="ledger.stats.borrowPending || 0" /></b>
            <span>项待审批</span>
          </li>
        </ul>

        <div class="eq-actions">
          <button class="eq-cta" type="button" @click="jumpToAvailable()">看现在能借什么</button>
          <router-link class="eq-alt" to="/chain">查看借还记录</router-link>
        </div>
      </div>

      <figure data-parallax class="eq-photo" data-reveal data-delay="120">
        <img src="/media/xlab-official-vr-demo.png" alt="X-Lab 工作人员指导学生使用 VR 头显与手柄" width="960" height="720" />
        <figcaption data-ui>X-Lab · VR 设备指导</figcaption>
      </figure>
    </header>

    <!-- 四类设备的实物索引 -->
    <section data-parallax class="eq-gallery" aria-label="设备实物">
      <figure data-reveal>
        <img src="/media/xlab-official-mr-demo.png" alt="学生在 X-Lab 使用混合现实头显与控制器" width="960" height="640" loading="lazy" />
        <figcaption data-ui>交互设备 · MR 头显</figcaption>
      </figure>
      <figure data-reveal data-delay="90">
        <img src="/media/xlab-official-experience.png" alt="学生在 X-Lab 进行沉浸式绘画创作" width="960" height="640" loading="lazy" />
        <figcaption data-ui>创作设备 · 沉浸式绘画</figcaption>
      </figure>
      <figure data-reveal data-delay="180">
        <img src="/media/xlab-official-opening.png" alt="X-Lab 内多台 XR 设备正在同时使用" width="960" height="720" loading="lazy" />
        <figcaption data-ui>设备协作 · XR 教学现场</figcaption>
      </figure>
      <figure data-reveal data-delay="270">
        <img src="/media/xlab-official-xr-space.png" alt="学生在 X-Lab 混合现实空间浏览数字内容" width="960" height="640" loading="lazy" />
        <figcaption data-ui>交互设备 · 混合现实空间</figcaption>
      </figure>
    </section>

    <!-- 工具条 -->
    <div class="toolbar" data-reveal>
      <div class="search">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="7" /><path d="M21 21l-4-4" />
        </svg>
        <input v-model="q" name="equipmentSearch" autocomplete="off" aria-label="搜索设备" placeholder="搜索编号、名称、品牌、类别或位置…" />
      </div>
      <div class="seg" role="group" aria-label="设备状态筛选">
        <button
          v-for="f in filters"
          :key="f.key"
          type="button"
          :class="{ on: status === f.key }"
          :aria-pressed="status === f.key"
          @click="status = f.key"
        >
          {{ f.label }}
          <i>{{ counts[f.key] || 0 }}</i>
        </button>
      </div>
    </div>

    <!-- 清单：按类别成册，用表格陈列 -->
    <section v-for="g in groups" :key="g.name" class="shelf" data-reveal>
      <div class="shelf-head">
        <h2>{{ g.name }}</h2>
        <span class="shelf-count">{{ g.items.length }} 件</span>
      </div>

      <table class="rows">
        <colgroup>
          <col style="width: 36%" />
          <col style="width: 21%" />
          <col style="width: 21%" />
          <col style="width: 8%" />
          <col style="width: 14%" />
        </colgroup>
        <thead>
          <tr>
            <th scope="col">设备</th>
            <th scope="col">位置</th>
            <th scope="col">保管</th>
            <th scope="col" class="c-num">借出</th>
            <th scope="col" class="c-act"></th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="(e, i) in g.items"
            :key="e.id"
            data-reveal
            :data-delay="Math.min(i, 6) * 45"
          >
            <td class="c-name">
              <router-link class="row-link" :to="`/equipment/${e.id}`">{{ e.name }}</router-link>
              <span>{{ e.id }} · {{ e.brand }} {{ e.model }}</span>
            </td>
            <td class="c-cell">{{ e.location }}</td>
            <td class="c-cell">{{ e.custodian }}</td>
            <td class="c-num">{{ e.totalBorrows }} 次</td>
            <td class="c-act">
              <StatusChip :label="label(EQUIPMENT_STATUS, e.status)" :tone="tone(EQUIPMENT_STATUS, e.status)" />
              <button v-if="e.status === 'AVAILABLE'" class="ask" @click.stop="openBorrow(e)">
                <span>申请借用</span><span class="arw" aria-hidden="true">→</span>
              </button>
              <span v-else-if="e.projectId" class="row-proj">{{ projectOf(e) }}</span>
            </td>
          </tr>
        </tbody>
      </table>
    </section>

    <div v-if="!filtered.length" class="empty panel">
      <strong>没有匹配的设备</strong>
      <span>换个关键词或筛选条件试试</span>
    </div>


    <AppModal
      v-if="showBorrow"
      :title="`申请借用 · ${borrowTarget.name}`"
      :subtitle="`${borrowTarget.id} · ${borrowTarget.location}`"
      @close="showBorrow = false"
    >
      <div class="modal-grid">
        <div class="field span-2">
          <label for="borrow-project">关联项目</label>
          <select id="borrow-project" v-model="form.projectId" name="projectId" autocomplete="off">
            <option value="">个人借用（不关联项目）</option>
            <option v-for="p in ledger.activeProjects" :key="p.id" :value="p.id">{{ p.id }} · {{ p.name }}</option>
          </select>
        </div>
        <div class="field">
          <label for="borrow-person">借用人</label>
          <input id="borrow-person" v-model="form.borrower" name="borrower" autocomplete="name" required />
        </div>
        <div class="field">
          <label for="borrow-due">预计归还日期</label>
          <input id="borrow-due" v-model="form.dueDate" name="dueDate" type="date" autocomplete="off" required />
        </div>
        <div class="field span-2">
          <label for="borrow-purpose">使用用途</label>
          <textarea id="borrow-purpose" v-model="form.purpose" name="purpose" autocomplete="off" required placeholder="说明使用场景、地点与交接注意事项…" />
        </div>
      </div>

      <div class="note-box">
        提交后管理方审批 → 双方确认领取（记录设备现状与附件）→ 使用 → 发起归还 → 验收。
        用途文本与照片只存哈希，正文留在链下。
      </div>

      <template #foot>
        <button class="btn btn-ghost" @click="showBorrow = false">取消</button>
        <button class="btn btn-primary" :disabled="submitting" @click="submitBorrow">
          {{ submitting ? '提交中…' : '提交申请' }}
        </button>
      </template>
    </AppModal>
  </div>
</template>

<style scoped>
/* ---------- 首屏 ---------- */
.eq-hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 0.88fr);
  gap: 56px;
  align-items: stretch;
  padding-bottom: 8px;
}

.eq-copy { display: flex; flex-direction: column; justify-content: center; padding: 22px 0 10px; }

.eq-copy h1 {
  font-family: var(--serif);
  font-size: clamp(40px, 5.4vw, 84px);
  font-weight: 500;
  line-height: 1.06;
  letter-spacing: -0.02em;
  color: var(--ink);
}

.eq-sub {
  margin-top: 26px;
  font-size: 16px;
  line-height: 1.72;
  color: var(--ink-3);
  max-width: 44ch;
}

/* 账目：数字与说明排在同一行，标签不压在数字上方 */
.eq-tally {
  list-style: none;
  margin: 42px 0 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 26px;
}

.eq-tally li {
  display: flex;
  align-items: baseline;
  gap: 9px;
  padding: 0;
  background: transparent;
}

.eq-tally li.warn b,
.eq-tally li.warn span { color: var(--warn); }

.eq-actions {
  margin-top: 30px;
  display: flex;
  align-items: center;
  gap: 26px;
  flex-wrap: wrap;
}

.eq-cta {
  border: 0;
  background: var(--ink);
  color: #fff;
  font-family: var(--sans);
  font-size: 16px;
  font-weight: 550;
  padding: 14px 26px;
  min-height: 48px;
  border-radius: 10px;
  cursor: pointer;
  transition: background 0.16s ease;
}

.eq-cta:hover { background: var(--blue); }

.eq-alt {
  font-size: 16px;
  font-weight: 550;
  color: var(--ink-2);
  text-decoration: none;
  transition: color 0.16s ease;
}

.eq-alt:hover { color: var(--blue-deep); }

.eq-tally b {
  font-family: var(--serif);
  font-size: 32px;
  font-weight: 500;
  letter-spacing: -0.02em;
  line-height: 1;
  color: var(--ink);
  font-variant-numeric: tabular-nums;
}

.eq-tally span {
  font-size: 14px;
  color: var(--ink-2);
  white-space: nowrap;
}

.eq-photo {
  margin: 0;
  position: relative;
  overflow: hidden;
  border-radius: 18px;
  min-height: 360px;
}

.eq-photo img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.eq-photo::after,
.eq-gallery figure::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(0deg, rgba(5, 18, 31, 0.7), transparent 42%); /* taste-skill-ignore: image caption scrim */
}

.eq-photo figcaption {
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

/* ---------- 实物索引 ---------- */
.eq-gallery {
  margin: 64px calc(var(--gutter) * -1) 0;
  width: auto;
  display: grid;
  grid-template-columns: 1.35fr 1fr 1fr 0.86fr;
  gap: 3px;
  height: 40vh;
  min-height: 280px;
}

.eq-gallery video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.eq-gallery figure { margin: 0; position: relative; overflow: hidden; }

.eq-gallery img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform 0.9s cubic-bezier(0.16, 1, 0.3, 1);
}

.eq-gallery figure:hover img { transform: scale(1.05); }

.eq-gallery figcaption {
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

/* ---------- 工具条 ---------- */
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  flex-wrap: wrap;
  padding: 56px 0 4px;
}

/* ---------- 分册 ---------- */
/* 每一册是一条浅底账页带：满幅浅灰底色把长清单分段，白纸舞台不至于大片留白 */
.shelf {
  margin: 3px calc(var(--gutter) * -1) 0;
  width: auto;
  padding: 30px var(--gutter) 34px;
  background: #e1e7ec;
}
.shelf:nth-of-type(even) { background: #f8fafb; }

.shelf-head {
  display: flex;
  align-items: baseline;
  gap: 12px;
  padding-bottom: 12px;
}

.shelf-head h2 {
  font-size: 22px;
  font-weight: 620;
  letter-spacing: -0.025em;
  color: var(--ink);
}

.shelf-count {
  font-size: 14px;
  color: var(--ink-3);
}

/* 表格：行间用 1px 底色线分隔，悬停整行浮起 */
.rows {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.rows th {
  text-align: left;
  font-size: 14px;
  font-weight: 500;
  color: var(--ink-3);
  padding: 10px 18px 12px 0;
  border-bottom: 1px solid var(--surface-3);
}

.rows th.c-num,
.rows th.c-act { text-align: right; }

.rows td {
  padding: 18px 18px 18px 0;
  border-bottom: 1px solid var(--surface-2);
  vertical-align: middle;
}

.rows tbody tr { transition: background 0.16s ease; }

.rows tbody tr:hover { background: var(--surface); }

@media (min-width: 1181px) {
  .rows, .rows tbody { display: block; }
  .rows colgroup, .rows thead { display: none; }
  .rows tbody {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    column-gap: 34px;
  }
  .rows tbody tr {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    grid-template-areas:
      'name action'
      'location action'
      'keeper count';
    align-items: center;
    gap: 5px 18px;
    min-width: 0;
    padding: 18px 20px;
    border-bottom: 1px solid var(--surface-3);
    background: transparent;
  }
  .rows tbody tr:hover { background: rgba(255, 255, 255, 0.54); }
  .rows td { padding: 0; border: 0; min-width: 0; }
  .rows .c-name { grid-area: name; }
  .rows td:nth-child(2) { grid-area: location; }
  .rows td:nth-child(3) { grid-area: keeper; }
  .rows .c-num { grid-area: count; }
  .rows .c-act { grid-area: action; }
}

.row-link {
  display: block;
  font-size: 16px;
  font-weight: 620;
  letter-spacing: -0.015em;
  color: var(--ink);
  text-decoration: none;
}
.row-link:hover { color: var(--blue-deep); text-decoration: underline; text-underline-offset: 3px; }

.c-name span {
  display: block;
  margin-top: 4px;
  font-size: 14px;
  color: var(--ink-3);
}

.c-cell {
  font-size: 14px;
  color: var(--ink-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.c-num {
  font-size: 14px;
  color: var(--ink-2);
  text-align: right;
  font-variant-numeric: tabular-nums;
}

.c-act {
  text-align: right;
  white-space: normal;
}

/* 借用中的行：状态与项目编号上下排。编号是不可断的长串，必须显式允许断行，
   否则 nowrap 会把整行撑出表格右边界 */
.row-proj {
  display: block;
  margin-top: 6px;
  font-size: 14px;
  color: var(--ink-3);
  line-height: 1.4;
}

/* 行内动作：默认是文字，悬停才落成实心块，避免一屏十几颗蓝按钮 */
.ask {
  border: 0;
  background: transparent;
  color: var(--blue-deep);
  font-family: var(--sans);
  font-size: 14px;
  font-weight: 600;
  min-width: 86px;
  padding: 9px 0;
  margin-left: 12px;
  cursor: pointer;
  transition: color 0.16s ease;
  white-space: nowrap;
}

.ask .arw {
  display: inline-block;
  transition: transform 0.22s cubic-bezier(0.16, 1, 0.3, 1);
}

.rows tbody tr:hover .ask {
  color: var(--blue);
}

.rows tbody tr:hover .ask .arw { transform: translateX(3px); }

.row-proj {
  font-size: 14px;
  color: var(--blue-deep);
  font-variant-numeric: tabular-nums;
  /* 项目编号是不可断的长串：父级 c-act 是 nowrap，这里必须显式放开，否则会把表格撑出视口 */
  white-space: normal;
  overflow-wrap: anywhere;
  word-break: break-all;
}

.modal-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.span-2 { grid-column: span 2; }

.note-box {
  margin-top: 18px;
  padding: 14px 16px;
  background: var(--blue-soft);
  font-size: 14px;
  color: var(--ink-2);
  line-height: 1.7;
}

.seg button i {
  font-style: normal;
  margin-left: 6px;
  color: var(--ink-4);
  font-variant-numeric: tabular-nums;
}

.seg button.on i { color: #c8d8e7; }

@media (max-width: 1180px) {
  .eq-hero { grid-template-columns: 1fr; gap: 32px; }
  .eq-photo { min-height: 300px; max-height: 380px; }
  .c-cell:nth-child(3) { display: none; }
  .rows th:nth-child(3) { display: none; }
}

@media (max-width: 720px) {
  .eq-copy h1 { font-size: 36px; }
  .eq-tally { gap: 18px; }
  .eq-tally b { font-size: 24px; }
  .eq-tally span { font-size: 14px; }
  .eq-gallery { grid-template-columns: 1fr 1fr; height: auto; }
  .eq-gallery figure { height: 168px; }
  /* 手机端各分册不再是一叠同宽白卡：底色交给表格自身 */
  .shelf { background: transparent; padding: 24px 24px 26px; }
  .shelf-head { padding-bottom: 10px; }
  .c-cell { display: none; }
  .rows th:nth-child(2), .rows th:nth-child(3) { display: none; }
  .rows { table-layout: auto; }
  .c-act { display: flex; flex-direction: column; align-items: flex-start; gap: 7px; }
  .ask { padding: 0; }
  .ask .arw { display: none; }
  .modal-grid { grid-template-columns: 1fr; }
  .span-2 { grid-column: span 1; }
  /* 四个筛选按钮在 390 宽下放不进一行：允许换行，别把视口撑开 */
  .toolbar { flex-wrap: wrap; gap: 10px; }
  .seg { flex-wrap: wrap; gap: 6px; }
  .seg button { padding: 7px 11px; font-size: 14px; }
}
</style>
