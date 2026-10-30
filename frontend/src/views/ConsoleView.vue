<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useLedger } from '../lib/store'
import { useReveal } from '../lib/reveal'
import { usePointerTilt } from '../lib/pointer'
import { useParallax } from '../lib/parallax'
import CountUp from '../components/CountUp.vue'
import { fmtDate } from '../lib/labels'
import AppModal from '../components/AppModal.vue'

const ledger = useLedger()
const router = useRouter()
const route = useRoute()

useReveal()
usePointerTilt()
useParallax('[data-parallax]')

const tabKeys = new Set(['applications', 'borrows', 'pickup', 'returns', 'exceptions', 'allocate'])
const tab = ref(tabKeys.has(route.query.tab) ? route.query.tab : 'applications')

watch(tab, (nextTab) => {
  const query = { ...route.query }
  if (nextTab !== 'applications') query.tab = nextTab
  else delete query.tab
  router.replace({ query })
})

const tabs = computed(() => [
  { key: 'applications', label: '入驻评审', count: ledger.pendingApplications.length },
  { key: 'borrows', label: '借出审批', count: ledger.pendingBorrows.length },
  { key: 'pickup', label: '待领取', count: ledger.approvedBorrows.length },
  { key: 'returns', label: '待验收', count: ledger.returnPending.length },
  { key: 'exceptions', label: '异常', count: ledger.exceptions.length },
  { key: 'allocate', label: '待分配空间', count: ledger.approvedApplications.length },
])

// 评审
const review = ref(null)
const reviewNote = ref('')
const approving = ref(false)

function openReview(app) {
  review.value = app
  reviewNote.value = ''
}

async function doReview(approved) {
  approving.value = true
  try {
    await ledger.reviewApplication(review.value.id, approved, reviewNote.value)
    review.value = null
    window.__xlabToast?.(approved ? '已通过评审，请分配空间' : '已驳回申请')
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  } finally {
    approving.value = false
  }
}

// 借用审批
const borrow = ref(null)
const borrowNote = ref('')
const borrowBusy = ref(false)

async function doReviewBorrow(approved) {
  if (borrowBusy.value) return
  borrowBusy.value = true
  try {
    await ledger.reviewBorrow(borrow.value.id, approved, borrowNote.value)
    borrow.value = null
    window.__xlabToast?.(approved ? '已批准，等待现场核对领取' : '已驳回借用申请')
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  } finally {
    borrowBusy.value = false
  }
}

// 领取确认
const pickup = ref(null)
const pickupCond = ref('GOOD')
const pickupBusy = ref(false)

async function doPickup() {
  if (pickupBusy.value) return
  pickupBusy.value = true
  try {
    await ledger.confirmPickup(pickup.value.id, pickupCond.value, [])
    pickup.value = null
    window.__xlabToast?.('领取已确认，设备进入借出状态')
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  } finally {
    pickupBusy.value = false
  }
}

// 归还验收
const ret = ref(null)
const retCond = ref('NORMAL')
const retNote = ref('')
const retBusy = ref(false)

async function doReturn() {
  if (retBusy.value) return
  retBusy.value = true
  try {
    await ledger.confirmReturn(ret.value.id, retCond.value, retNote.value, [])
    ret.value = null
    window.__xlabToast?.(retCond.value === 'NORMAL' ? '验收通过，设备已恢复可借' : '已记录异常，设备转入维护')
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  } finally {
    retBusy.value = false
  }
}

// 异常处理
const resolveBusy = ref(false)

async function doResolve(b) {
  if (resolveBusy.value) return
  resolveBusy.value = true
  try {
    await ledger.resolveException(b.id, '异常已处理，设备恢复可用')
    window.__xlabToast?.('异常已处理')
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  } finally {
    resolveBusy.value = false
  }
}

// 空间分配
const allocate = ref(null)
const chosenSpaces = ref([])
const activateBusy = ref(false)

function openAllocate(app) {
  allocate.value = app
  chosenSpaces.value = [...(app.preferredSpace || [])].filter((id) => {
    const sp = ledger.spaceMap[id]
    return sp && sp.status === 'OPEN' && !spaceConflict(sp, app)
  })
}

function toggleAlloc(id) {
  const i = chosenSpaces.value.indexOf(id)
  if (i >= 0) chosenSpaces.value.splice(i, 1)
  else chosenSpaces.value.push(id)
}

function periodsOverlap(aStart, aEnd, bStart, bEnd) {
  return aStart < bEnd && aEnd > bStart
}

function spaceConflict(sp, app = allocate.value) {
  if (!app) return null
  const alloc = ledger.allocations.find((a) =>
    a.spaceId === sp.id &&
    a.status === 'ACTIVE' &&
    periodsOverlap(app.startDate, app.endDate, a.startDate, a.endDate)
  )
  if (!alloc) return null
  return ledger.projectMap[alloc.projectId]?.name || '已占用'
}

function spaceBlocked(sp) {
  return spaceConflict(sp)
}

async function doActivate() {
  if (!chosenSpaces.value.length) {
    window.__xlabToast?.('请选择至少一个空间', 'err')
    return
  }
  if (activateBusy.value) return
  activateBusy.value = true
  try {
    const p = await ledger.activateProject(allocate.value.id, chosenSpaces.value)
    allocate.value = null
    window.__xlabToast?.('项目已入驻，空间已分配')
    router.push(`/projects/${p.id}`)
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  } finally {
    activateBusy.value = false
  }
}
</script>

<template>
  <div class="console">
    <!-- 首屏：这里是谁在做什么 -->
    <header class="cs-hero">
      <div class="cs-copy" data-reveal>
        <h1>管理台</h1>

        <p class="cs-sub">
          在这里处理入驻评审、空间分配和设备借还。系统会记录操作组织和处理时间。
        </p>

        <ul class="cs-tally">
          <li :class="{ warn: ledger.backlogCount > 0 }" data-reveal data-delay="40">
            <b><CountUp :to="ledger.backlogCount" /></b><span>项待处理</span>
          </li>
          <li data-reveal data-delay="90"><b><CountUp :to="ledger.stats.projectActive || 0" /></b><span>个在驻项目</span></li>
          <li data-reveal data-delay="140"><b><CountUp :to="ledger.stats.equipmentOut || 0" /></b><span>台借出中</span></li>
        </ul>
      </div>

      <figure data-parallax class="cs-photo" data-reveal data-delay="120">
        <img src="/media/xlab-official-opening.png" alt="浙江大学图书馆 X-Lab 的开放体验空间" width="960" height="720" fetchpriority="high" />
        <figcaption data-ui>浙江大学图书馆 X-Lab · 资源现场</figcaption>
      </figure>
    </header>

    <!-- 任务切换：色块 -->
    <nav class="cs-tabs" data-reveal aria-label="管理任务筛选">
      <button
        v-for="t in tabs"
        :key="t.key"
        type="button"
        :class="{ on: tab === t.key, has: t.count > 0 }"
        :aria-pressed="tab === t.key"
        @click="tab = t.key"
      >
        <span>{{ t.label }}</span>
        <b>{{ t.count }}</b>
      </button>
    </nav>

    <!-- 入驻评审 -->
    <section v-if="tab === 'applications'" class="board" data-ui data-reveal>
      <div class="board-head">
        <h2>待评审的入驻申请</h2>
        <span>{{ ledger.pendingApplications.length }} 条</span>
      </div>

      <table class="book" v-if="ledger.pendingApplications.length">
        <thead>
          <tr>
            <th>申请编号</th>
            <th>项目</th>
            <th>负责人</th>
            <th>团队</th>
            <th>周期</th>
            <th>偏好空间</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(a, i) in ledger.pendingApplications" :key="a.id" data-reveal :data-delay="i * 40">
            <td class="cell-id">{{ a.id }}</td>
            <td>
              <strong class="cell-strong">{{ a.projectName }}</strong>
              <p class="cell-note">{{ a.ideaSummary }}</p>
            </td>
            <td>{{ a.leadName }}</td>
            <td>{{ a.memberCount }} 人</td>
            <td class="num">{{ fmtDate(a.startDate) }} → {{ fmtDate(a.endDate) }}</td>
            <td class="num">{{ (a.preferredSpace || []).join(', ') || '暂无' }}</td>
            <td class="cell-act"><button class="btn btn-sm btn-primary" @click="openReview(a)">评审</button></td>
          </tr>
        </tbody>
      </table>
      <p v-else class="board-empty">没有待评审的申请，全部处理完了。</p>
    </section>

    <!-- 待分配空间 -->
    <section v-if="tab === 'allocate'" class="board" data-ui data-reveal>
      <div class="board-head">
        <h2>已通过评审 · 待分配空间</h2>
        <span>{{ ledger.approvedApplications.length }} 条</span>
      </div>

      <table class="book" v-if="ledger.approvedApplications.length">
        <thead>
          <tr>
            <th>申请编号</th>
            <th>项目</th>
            <th>周期</th>
            <th>申请工位</th>
            <th>偏好</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(a, i) in ledger.approvedApplications" :key="a.id" data-reveal :data-delay="i * 40">
            <td class="cell-id">{{ a.id }}</td>
            <td><strong class="cell-strong">{{ a.projectName }}</strong></td>
            <td class="num">{{ fmtDate(a.startDate) }} → {{ fmtDate(a.endDate) }}</td>
            <td>{{ a.requestedSeats }}</td>
            <td class="num">{{ (a.preferredSpace || []).join(', ') || '暂无' }}</td>
            <td class="cell-act"><button class="btn btn-sm btn-primary" @click="openAllocate(a)">分配空间</button></td>
          </tr>
        </tbody>
      </table>
      <p v-else class="board-empty">没有待分配的申请。</p>
    </section>

    <!-- 借出审批 -->
    <section v-if="tab === 'borrows'" class="board" data-ui data-reveal>
      <div class="board-head">
        <h2>待审批的借用申请</h2>
        <span>{{ ledger.pendingBorrows.length }} 条</span>
      </div>

      <table class="book" v-if="ledger.pendingBorrows.length">
        <thead>
          <tr>
            <th>借用单</th>
            <th>设备</th>
            <th>借用人</th>
            <th>项目</th>
            <th>用途</th>
            <th>应还</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(b, i) in ledger.pendingBorrows" :key="b.id" data-reveal :data-delay="i * 40">
            <td class="cell-id">{{ b.id }}</td>
            <td>
              <strong class="cell-strong">{{ b.equipmentName }}</strong>
              <p class="cell-note">{{ b.equipmentId }}</p>
            </td>
            <td>{{ b.borrower }}</td>
            <td class="num">{{ b.projectId || '个人' }}</td>
            <td class="cell-note" style="max-width: 220px">{{ b.purpose }}</td>
            <td class="num">{{ fmtDate(b.dueDate) }}</td>
            <td class="cell-act">
              <button class="btn btn-sm btn-primary" @click="(borrow = b), (borrowNote = '')">审批</button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="board-empty">没有待审批的借用。</p>
    </section>

    <!-- 待领取 -->
    <section v-if="tab === 'pickup'" class="board" data-ui data-reveal>
      <div class="board-head">
        <h2>已批准 · 待领取交接</h2>
        <span>{{ ledger.approvedBorrows.length }} 条</span>
      </div>
      <p class="board-lede">管理方核对设备与附件并记录后，设备才正式借出。</p>

      <table class="book" v-if="ledger.approvedBorrows.length">
        <thead>
          <tr><th>借用单</th><th>设备</th><th>借用人</th><th>项目</th><th></th></tr>
        </thead>
        <tbody>
          <tr v-for="(b, i) in ledger.approvedBorrows" :key="b.id" data-reveal :data-delay="i * 40">
            <td class="cell-id">{{ b.id }}</td>
            <td><strong class="cell-strong">{{ b.equipmentName }}</strong></td>
            <td>{{ b.borrower }}</td>
            <td class="num">{{ b.projectId || '个人' }}</td>
            <td class="cell-act">
              <button class="btn btn-sm btn-primary" @click="(pickup = b), (pickupCond = 'GOOD')">确认领取</button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="board-empty">没有待领取的设备。</p>
    </section>

    <!-- 待验收 -->
    <section v-if="tab === 'returns'" class="board" data-ui data-reveal>
      <div class="board-head">
        <h2>待验收的归还</h2>
        <span>{{ ledger.returnPending.length }} 条</span>
      </div>
      <p class="board-lede">记录归还结论；异常设备转入维护，不可再借。</p>

      <table class="book" v-if="ledger.returnPending.length">
        <thead>
          <tr><th>借用单</th><th>设备</th><th>借用人</th><th>起租 / 应还</th><th></th></tr>
        </thead>
        <tbody>
          <tr v-for="(b, i) in ledger.returnPending" :key="b.id" data-reveal :data-delay="i * 40">
            <td class="cell-id">{{ b.id }}</td>
            <td><strong class="cell-strong">{{ b.equipmentName }}</strong></td>
            <td>{{ b.borrower }}</td>
            <td class="num">{{ fmtDate(b.createdAt) }} / {{ fmtDate(b.dueDate) }}</td>
            <td class="cell-act">
              <button class="btn btn-sm btn-primary" @click="(ret = b), (retCond = 'NORMAL'), (retNote = '')">验收</button>
            </td>
          </tr>
        </tbody>
      </table>
      <p v-else class="board-empty">没有待验收的归还。</p>
    </section>

    <!-- 异常 -->
    <section v-if="tab === 'exceptions'" class="board" data-ui data-reveal>
      <div class="board-head">
        <h2>异常待处理</h2>
        <span>{{ ledger.exceptions.length }} 条</span>
      </div>

      <table class="book" v-if="ledger.exceptions.length">
        <thead>
          <tr><th>借用单</th><th>设备</th><th>借用人</th><th>说明</th><th></th></tr>
        </thead>
        <tbody>
          <tr v-for="(b, i) in ledger.exceptions" :key="b.id" data-reveal :data-delay="i * 40">
            <td class="cell-id">{{ b.id }}</td>
            <td><strong class="cell-strong">{{ b.equipmentName }}</strong></td>
            <td>{{ b.borrower }}</td>
            <td class="cell-note">{{ b.exceptionNote || '暂无' }}</td>
            <td class="cell-act"><button class="btn btn-sm btn-amber" :disabled="resolveBusy" @click="doResolve(b)">处理完毕</button></td>
          </tr>
        </tbody>
      </table>
      <p v-else class="board-empty">没有异常记录，所有设备状态正常。</p>
    </section>

    <!-- 满幅实景：委员会每天核对的现场 -->
    <section data-parallax class="cs-strip" data-reveal>
      <figure>
        <img src="/media/xlab-official-xr-space.png" alt="学生在 X-Lab 使用混合现实头显" width="960" height="640" loading="lazy" />
        <figcaption data-ui>资源使用 · 混合现实空间</figcaption>
      </figure>
      <figure data-reveal data-delay="90">
        <img src="/media/xlab-official-message-wall.png" alt="同学在 X-Lab 留言墙上记录体验反馈" width="960" height="640" loading="lazy" />
        <figcaption data-ui>资源反馈 · 创意留言墙</figcaption>
      </figure>
      <figure data-reveal data-delay="180">
        <img src="/media/xlab-official-vr-demo.png" alt="X-Lab 工作人员指导学生使用 VR 设备" width="960" height="720" loading="lazy" />
        <figcaption data-ui>设备交接 · VR 设备指导</figcaption>
      </figure>
    </section>


    <!-- 评审弹窗 -->
    <AppModal
      v-if="review"
      :title="`评审入驻申请 · ${review.projectName}`"
      :subtitle="`${review.id} · ${review.memberCount} 人 · 申请 ${review.requestedSeats} 个工位`"
      width="640px"
      @close="review = null"
    >
      <div class="modal-stack">
        <div class="detail-box">
          <p class="box-label">项目摘要</p>
          <p class="box-text">{{ review.ideaSummary }}</p>
        </div>
        <div class="box-grid">
          <div class="detail-box">
            <p class="box-label">入驻周期</p>
            <p class="box-text num">{{ fmtDate(review.startDate) }} → {{ fmtDate(review.endDate) }}</p>
          </div>
          <div class="detail-box">
            <p class="box-label">阶段目标</p>
            <p class="box-text">{{ review.milestone || '暂无' }}</p>
          </div>
        </div>
        <div class="detail-box">
          <p class="box-label">项目摘要哈希</p>
          <p class="box-text hash">{{ review.ideaHash }}</p>
        </div>
        <div class="field">
          <label for="review-note">评审意见</label>
          <textarea id="review-note" v-model="reviewNote" name="reviewNote" autocomplete="off" placeholder="通过或驳回的理由，会一并写入链上…" />
        </div>
      </div>
      <template #foot>
        <button class="btn btn-danger" :disabled="approving" @click="doReview(false)">驳回</button>
        <button class="btn btn-primary" :disabled="approving" @click="doReview(true)">通过评审</button>
      </template>
    </AppModal>

    <!-- 借用审批弹窗 -->
    <AppModal v-if="borrow" :title="`审批借用 · ${borrow.equipmentName}`" :subtitle="borrow.id" @close="borrow = null">
      <div class="modal-stack">
        <div class="kv">
          <div><span>借用人</span><b>{{ borrow.borrower }}</b></div>
          <div><span>关联项目</span><b>{{ ledger.projectMap[borrow.projectId]?.name || borrow.projectId || '个人借用' }}</b></div>
          <div><span>用途</span><b>{{ borrow.purpose }}</b></div>
          <div><span>应还日期</span><b class="num">{{ fmtDate(borrow.dueDate) }}</b></div>
        </div>
        <div class="field">
          <label for="borrow-review-note">审批意见</label>
          <textarea id="borrow-review-note" v-model="borrowNote" name="borrowReviewNote" autocomplete="off" placeholder="选填…" />
        </div>
      </div>
      <template #foot>
        <button class="btn btn-danger" :disabled="borrowBusy" @click="doReviewBorrow(false)">驳回</button>
        <button class="btn btn-primary" :disabled="borrowBusy" @click="doReviewBorrow(true)">批准</button>
      </template>
    </AppModal>

    <!-- 领取确认弹窗 -->
    <AppModal v-if="pickup" title="领取交接确认" :subtitle="`${pickup.id} · ${pickup.equipmentName}`" @close="pickup = null">
      <div class="modal-stack">
        <div class="detail-box">
          <p class="box-label">请核对设备与附件</p>
          <ul class="checklist">
            <li>设备编号与外观状态是否与台账一致</li>
            <li>随附配件是否齐全</li>
            <li>借用期限与用途已确认</li>
          </ul>
        </div>
        <div class="field">
          <label for="pickup-condition">设备成色</label>
          <select id="pickup-condition" v-model="pickupCond" name="pickupCondition" autocomplete="off">
            <option value="GOOD">良好</option>
            <option value="FAIR">一般</option>
          </select>
        </div>
      </div>
      <template #foot>
        <button class="btn btn-ghost" @click="pickup = null">取消</button>
        <button class="btn btn-primary" :disabled="pickupBusy" @click="doPickup">确认已领取</button>
      </template>
    </AppModal>

    <!-- 归还验收弹窗 -->
    <AppModal v-if="ret" title="归还验收" :subtitle="`${ret.id} · ${ret.equipmentName}`" @close="ret = null">
      <div class="modal-stack">
        <div class="field">
          <label for="return-condition">验收结论</label>
          <select id="return-condition" v-model="retCond" name="returnCondition" autocomplete="off">
            <option value="NORMAL">正常归还（设备恢复可借）</option>
            <option value="DAMAGED">损坏（转入维护）</option>
            <option value="MISSING">缺件（转入维护）</option>
          </select>
        </div>
        <div class="field">
          <label for="return-note">备注</label>
          <textarea id="return-note" v-model="retNote" name="returnNote" autocomplete="off" placeholder="记录外观、功能或附件异常…" />
        </div>
        <div class="note-box" v-if="retCond !== 'NORMAL'">
          选择损坏或缺件后，设备状态会变为「维护中」，需另行处理完毕才能再次出借。
        </div>
      </div>
      <template #foot>
        <button class="btn btn-ghost" @click="ret = null">取消</button>
        <button class="btn btn-primary" :disabled="retBusy" @click="doReturn">提交验收</button>
      </template>
    </AppModal>

    <!-- 空间分配弹窗 -->
    <AppModal
      v-if="allocate"
      :title="`分配空间 · ${allocate.projectName}`"
      :subtitle="`${allocate.id} · ${fmtDate(allocate.startDate)} → ${fmtDate(allocate.endDate)}`"
      width="720px"
      @close="allocate = null"
    >
      <p class="modal-lede">选择要分配给该项目的空间。已被其他项目在同期占用的空间会标记为冲突。</p>
      <div class="space-pick">
        <button
          v-for="sp in ledger.spaces"
          :key="sp.id"
          type="button"
          class="pick"
          :class="{
            on: chosenSpaces.includes(sp.id),
            blocked: !!spaceBlocked(sp) || sp.status !== 'OPEN',
          }"
          :disabled="!!spaceBlocked(sp) || sp.status !== 'OPEN'"
          :aria-pressed="chosenSpaces.includes(sp.id)"
          @click="toggleAlloc(sp.id)"
        >
          <span class="pick-id">{{ sp.id }}</span>
          <strong>{{ sp.name }}</strong>
          <span class="pick-meta">{{ sp.floor }} · {{ sp.zone }} · {{ sp.capacity }} 人</span>
          <span v-if="spaceBlocked(sp)" class="pick-warn">占用：{{ spaceBlocked(sp) }}</span>
          <span v-else-if="sp.status !== 'OPEN'" class="pick-warn">维护中</span>
        </button>
      </div>
      <template #foot>
        <button class="btn btn-ghost" @click="allocate = null">取消</button>
        <button class="btn btn-primary" :disabled="activateBusy" @click="doActivate">确认分配并入驻</button>
      </template>
    </AppModal>
  </div>
</template>

<style scoped>
/* ---------- 首屏 ---------- */
.cs-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.24fr) minmax(0, 0.72fr);
  gap: 3px;
  margin-bottom: 3px;
}

.cs-copy {
  background: var(--surface);
  padding: 34px 38px 38px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.cs-copy h1 {
  font-family: var(--serif);
  font-size: clamp(34px, 4.4vw, 66px);
  font-weight: 500;
  letter-spacing: -0.02em;
  line-height: 1.05;
  color: var(--ink);
  max-width: 12em;
}

.cs-sub {
  margin-top: 22px;
  font-size: 16px;
  line-height: 1.72;
  color: var(--ink-3);
  max-width: 46ch;
}

.cs-tally {
  list-style: none;
  margin: 30px 0 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 22px;
}

.cs-tally li {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 0;
  background: transparent;
}
.cs-tally li.warn b,
.cs-tally li.warn span { color: var(--warn); }

.cs-tally b {
  font-family: var(--serif);
  font-size: 22px;
  font-weight: 500;
  letter-spacing: -0.015em;
  line-height: 1;
  color: var(--ink);
  font-variant-numeric: tabular-nums;
}

.cs-tally span { font-size: 14px; color: var(--ink-2); white-space: nowrap; }

.cs-photo { margin: 0; position: relative; overflow: hidden; min-height: 320px; }
.cs-photo img { width: 100%; height: 100%; object-fit: cover; display: block; transition: transform 0.8s cubic-bezier(0.16, 1, 0.3, 1); }
.cs-photo:hover img { transform: scale(1.04); }
.cs-photo::after,
.cs-strip figure::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(0deg, rgba(5, 18, 31, 0.7), transparent 42%); /* taste-skill-ignore: image caption scrim */
}
.cs-photo figcaption {
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

/* ---------- 任务切换 ---------- */
.cs-tabs {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 3px;
  margin-bottom: 3px;
}

.cs-tabs button {
  border: 0;
  font-family: var(--sans);
  text-align: left;
  background: var(--surface);
  padding: 15px 16px 16px;
  cursor: pointer;
  transition: background 0.18s ease;
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 10px;
}

.cs-tabs button span { font-size: 14px; color: var(--ink-2); }
.cs-tabs button b {
  font-size: 14px;
  font-weight: 620;
  color: var(--ink-4);
  font-variant-numeric: tabular-nums;
}

.cs-tabs button:hover { background: var(--bg); }
.cs-tabs button.has b { color: var(--blue-deep); }
.cs-tabs button.on { background: var(--blue); }
.cs-tabs button.on span { color: #fff; font-weight: 600; }
.cs-tabs button.on b { color: rgba(255, 255, 255, 0.86); }

/* ---------- 任务面板 ---------- */
.board { background: var(--surface); padding: 24px 28px 28px; }

.board-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 6px;
}

.board-head h2 {
  font-family: var(--serif);
  font-size: 24px;
  font-weight: 500;
  letter-spacing: -0.015em;
  color: var(--ink);
}

.board-head span { font-size: 14px; color: var(--ink-3); }

.board-lede { font-size: 16px; color: var(--ink-2); line-height: 1.66; margin-bottom: 14px; }

.board-empty { font-size: 16px; color: var(--ink-3); padding: 12px 0 6px; }

/* 表格：整棵子树豁免细线 */
.book { width: 100%; border-collapse: collapse; font-size: 16px; }

.book th {
  text-align: left;
  padding: 12px 14px 12px 0;
  font-size: 14px;
  font-weight: 550;
  color: var(--ink-3);
  background: var(--bg);
  white-space: nowrap;
}
.book th:first-child { padding-left: 14px; }
.book td:first-child { padding-left: 14px; }

.book td {
  padding: 15px 14px 15px 0;
  border-top: 1px solid var(--surface-2);
  color: var(--ink-2);
  vertical-align: top;
}

.cell-id { color: var(--ink); font-weight: 550; font-variant-numeric: tabular-nums; white-space: nowrap; }
.cell-strong { color: var(--ink); font-weight: 600; font-size: 16px; }
.cell-note {
  margin-top: 3px;
  font-size: 14px;
  color: var(--ink-3);
  line-height: 1.55;
  max-width: 320px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.num { font-variant-numeric: tabular-nums; white-space: nowrap; }
.cell-act { text-align: right; padding-right: 0; white-space: nowrap; }

/* ---------- 弹窗内部 ---------- */
.modal-stack { display: grid; gap: 12px; }
.modal-lede { font-size: 16px; color: var(--ink-2); line-height: 1.66; margin-bottom: 14px; }

.detail-box { background: var(--surface-2); padding: 14px 16px; }
.box-label { font-size: 14px; color: var(--ink-3); margin-bottom: 5px; }
.box-text { font-size: 16px; line-height: 1.66; color: var(--ink); }
.box-text.hash { font-variant-numeric: tabular-nums; word-break: break-all; color: var(--blue-deep); font-size: 14px; }
.box-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }

.checklist {
  margin: 6px 0 0 18px;
  font-size: 14px;
  line-height: 1.9;
  color: var(--ink-2);
}

.kv { display: grid; gap: 2px; }
.kv div {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr);
  gap: 14px;
  padding: 11px 13px;
  background: var(--surface-2);
  font-size: 16px;
}
.kv span { color: var(--ink-3); }
.kv b { font-weight: 550; color: var(--ink); text-align: right; }

.note-box {
  padding: 14px 16px;
  background: var(--sun-soft);
  font-size: 14px;
  line-height: 1.66;
  color: var(--ink);
}

/* 空间选择 */
.space-pick { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 2px; }

.pick {
  border: 0;
  font-family: var(--sans);
  text-align: left;
  background: var(--surface-2);
  padding: 13px 15px 15px;
  cursor: pointer;
  transition: background 0.16s ease;
}

.pick:hover { background: var(--blue-soft); }
.pick.on { background: var(--blue); }
.pick.on .pick-id,
.pick.on strong,
.pick.on .pick-meta { color: #fff; }
.pick.blocked { background: var(--surface-3); cursor: not-allowed; }
.pick.blocked strong { color: var(--ink-3); }

.pick-id { display: block; font-size: 14px; color: var(--blue-deep); font-variant-numeric: tabular-nums; }
.pick strong { display: block; margin-top: 5px; font-size: 16px; font-weight: 600; color: var(--ink); }
.pick-meta { display: block; margin-top: 4px; font-size: 14px; color: var(--ink-3); }
.pick-warn { display: block; margin-top: 6px; font-size: 14px; color: var(--bad); }

/* ---------- 满幅实景带 ---------- */
.cs-strip {
  margin: 3px calc(var(--gutter) * -1) 0;
  width: auto;
  display: grid;
  grid-template-columns: 1.18fr 1fr 1.08fr;
  gap: 3px;
  height: 32vh;
  min-height: 240px;
}

.cs-strip figure { margin: 0; position: relative; overflow: hidden; }
.cs-strip img,
.cs-strip video { width: 100%; height: 100%; object-fit: cover; display: block; }
.cs-strip figcaption {
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

@media (max-width: 980px) {
  .book th:nth-child(5), .book td:nth-child(5),
  .book th:nth-child(6), .book td:nth-child(6) { display: none; }
}

@media (max-width: 1180px) {
  .cs-strip { grid-template-columns: 1fr 1fr; height: 28vh; }
  .cs-strip figure:last-child { display: none; }
  .cs-hero { grid-template-columns: 1fr; }
  .cs-photo { min-height: 240px; max-height: 320px; }
  .cs-copy { padding: 26px 24px 30px; }
  .cs-tabs { grid-template-columns: repeat(3, 1fr); }
  .space-pick { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

@media (max-width: 720px) {
  .cs-copy h1 { font-size: 32px; }
  .cs-tally { gap: 16px; }
  .cs-tally b { font-size: 19px; }
  .cs-tally span { font-size: 14px; }
  .cs-tabs { grid-template-columns: repeat(2, 1fr); }
  .board { padding: 18px 16px 22px; }
  .box-grid { grid-template-columns: 1fr; }
  .space-pick { grid-template-columns: 1fr; }
  .book, .book tbody { display: block; }
  .book thead { display: none; }
  .book tbody { display: grid; gap: 6px; }
  .book tr {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 5px 14px;
    padding: 15px 14px;
    background: var(--bg);
  }
  .book td,
  .book td:first-child,
  .book td:nth-child(3),
  .book td:nth-child(5),
  .book td:nth-child(6) {
    display: block;
    min-width: 0;
    padding: 0;
    border: 0;
    grid-column: 1;
    white-space: normal;
  }
  .book .cell-act {
    grid-column: 2;
    grid-row: 1 / span 8;
    align-self: start;
    justify-self: end;
  }
  .cell-note { max-width: none; }
}
</style>
