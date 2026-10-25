<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useLedger } from '../lib/store'
import { useAuth } from '../lib/auth'
import {
  APPLICATION_STATUS, PROJECT_STATUS, SPACE_KIND,
  label, tone, fmtDate, toInputDate,
} from '../lib/labels'
import { useReveal } from '../lib/reveal'
import { usePointerTilt } from '../lib/pointer'
import { useParallax } from '../lib/parallax'
import CountUp from '../components/CountUp.vue'
import StatusChip from '../components/StatusChip.vue'
import AppModal from '../components/AppModal.vue'

const ledger = useLedger()
const auth = useAuth()
const router = useRouter()
const route = useRoute()

useReveal()
usePointerTilt()
useParallax('[data-parallax]')

const floor = ref(typeof route.query.floor === 'string' ? route.query.floor : '3F')
const hoverSpace = ref(null)
const selectedSpaces = ref([])

const floors = computed(() => [...new Set(ledger.spaces.map((s) => s.floor))].sort())
const floorSpaces = computed(() => ledger.spaces.filter((s) => s.floor === floor.value))

watch(floor, (nextFloor) => {
  const query = { ...route.query }
  if (nextFloor !== '3F') query.floor = nextFloor
  else delete query.floor
  router.replace({ query })
})

watch(floors, (available) => {
  if (available.length && !available.includes(floor.value)) floor.value = available[0]
}, { immediate: true })

// 12 列不等宽：真实的平面图里没有两间一样大的房，等宽格子会读成一排卡片
const PLAN_COLS = '1.35fr 1fr 1.18fr 0.92fr 1.26fr 1.05fr 1.42fr 0.88fr 1.12fr 1.28fr 0.96fr 1.2fr'

// mapY / mapH 就是平面图坐标。不能把不同起始行压成连续编号：
// C/D 区跨两行时会与 E 区重叠，视觉上便像少了一条分隔线。
// 没有房间覆盖的行按窄走廊渲染，其余行保留房间高度。
const rowPlan = computed(() => {
  const rows = floorSpaces.value.reduce((max, s) => Math.max(max, s.mapY + s.mapH - 1), 1)
  const template = Array.from({ length: rows }, (_, index) => {
    const row = index + 1
    const occupied = floorSpaces.value.some((s) => row >= s.mapY && row < s.mapY + s.mapH)
    return occupied ? 'minmax(86px, 1fr)' : '18px'
  }).join(' ')
  return { rows, template }
})

function spaceStyle(sp) {
  return {
    gridColumn: `${sp.mapX} / span ${sp.mapW}`,
    gridRow: `${sp.mapY} / span ${sp.mapH}`,
  }
}

function spaceOccupant(sp) {
  const alloc = ledger.allocations.find((a) => a.spaceId === sp.id && a.status === 'ACTIVE')
  if (!alloc) return null
  return ledger.projectMap[alloc.projectId] || null
}

function spaceState(sp) {
  if (sp.status === 'MAINTENANCE') return 'maintenance'
  if (spaceOccupant(sp)) return 'occupied'
  if (selectedSpaces.value.includes(sp.id)) return 'selected'
  return 'free'
}

function toggleSpace(sp) {
  if (spaceOccupant(sp) || sp.status === 'MAINTENANCE') return
  const i = selectedSpaces.value.indexOf(sp.id)
  if (i >= 0) selectedSpaces.value.splice(i, 1)
  else selectedSpaces.value.push(sp.id)
}

const pendingApps = computed(() => ledger.pendingApplications)
const activeProjects = computed(() => ledger.activeProjects)

const showApply = ref(false)
const submitting = ref(false)
const form = ref({
  projectName: '',
  leadName: '',
  membersText: '',
  requestedSeats: 2,
  startDate: '',
  endDate: '',
  ideaSummary: '',
  milestone: '',
})

function openApply() {
  const start = new Date()
  start.setDate(start.getDate() + 3)
  const end = new Date(start)
  end.setDate(end.getDate() + 28)
  form.value.startDate = toInputDate(start)
  form.value.endDate = toInputDate(end)
  form.value.leadName = auth.user?.subjectName || ''
  showApply.value = true
}

async function submitApply() {
  const checks = [
    [form.value.projectName.trim(), '请填写项目名称', '#apply-project-name'],
    [form.value.leadName.trim(), '请填写项目负责人', '#apply-lead'],
    [Number(form.value.requestedSeats) > 0, '申请工位数必须大于 0', '#apply-seats'],
    [form.value.startDate, '请选择开始日期', '#apply-start'],
    [form.value.endDate && form.value.endDate >= form.value.startDate, '结束日期不能早于开始日期', '#apply-end'],
    [form.value.ideaSummary.trim(), '请填写项目说明', '#apply-idea'],
  ]
  const invalid = checks.find(([valid]) => !valid)
  if (invalid) {
    window.__xlabToast?.(invalid[1], 'err')
    requestAnimationFrame(() => document.querySelector(invalid[2])?.focus())
    return
  }
  submitting.value = true
  try {
    const members = form.value.membersText
      .split(/[,，\s]+/)
      .map((x) => x.trim())
      .filter(Boolean)
    await ledger.createApplication({
      projectName: form.value.projectName,
      leadName: form.value.leadName,
      members: members.length ? members : [form.value.leadName],
      requestedSeats: Number(form.value.requestedSeats),
      preferredSpace: selectedSpaces.value,
      startDate: form.value.startDate,
      endDate: form.value.endDate,
      ideaSummary: form.value.ideaSummary,
      milestone: form.value.milestone,
    })
    showApply.value = false
    selectedSpaces.value = []
    window.__xlabToast?.('入驻申请已提交，等待管理方评审')
  } catch (e) {
    window.__xlabToast?.(e.message, 'err')
  } finally {
    submitting.value = false
  }
}

const activeSpaceHelp = computed(() => {
  const n = selectedSpaces.value.length
  const cap = selectedSpaces.value.reduce((s, id) => s + (ledger.spaceMap[id]?.capacity || 0), 0)
  return { n, cap }
})
</script>

<template>
  <div class="resview">
    <header class="rv-hero">
      <div class="rv-copy" data-reveal>
        <h1>项目入驻与空间</h1>
        <p class="rv-sub">
          填写团队信息、使用期限和阶段目标。评审通过后，管理方在资源地图中分配工位。
        </p>
        <ul class="rv-tally">
          <li data-reveal data-delay="40">
            <b><CountUp :to="ledger.stats.spaceFree || 0" /></b>
            <span>处可分配，共 {{ ledger.stats.spaceTotal || 0 }} 处</span>
          </li>
          <li data-reveal data-delay="90">
            <b><CountUp :to="ledger.stats.projectActive || 0" /></b>
            <span>个在驻项目</span>
          </li>
          <li class="warn" data-reveal data-delay="140">
            <b><CountUp :to="ledger.stats.applicationPending || 0" /></b>
            <span>份待评审申请</span>
          </li>
        </ul>
        <button class="btn btn-primary rv-cta" @click="openApply">提交入驻申请</button>
      </div>

      <figure data-parallax class="rv-photo" data-reveal data-delay="120">
        <img src="/media/xlab-official-opening.png" alt="浙江大学图书馆 X-Lab 的开放体验空间" width="960" height="720" />
        <figcaption data-ui>浙江大学图书馆 X-Lab · 开放空间</figcaption>
      </figure>
    </header>

    <!-- 空间的三种用法 -->
    <section data-parallax class="rv-strip" aria-label="空间实景">
      <figure data-reveal>
        <img src="/media/xlab-official-message-wall.png" alt="同学在 X-Lab 留言墙上记录想法" width="960" height="640" loading="lazy" />
        <figcaption data-ui>协作空间 · 创意留言墙</figcaption>
      </figure>
      <figure data-reveal data-delay="90">
        <img src="/media/xlab-official-xr-space.png" alt="学生在 X-Lab 混合现实空间进行交互体验" width="960" height="640" loading="lazy" />
        <figcaption data-ui>体验空间 · 混合现实</figcaption>
      </figure>
      <figure data-reveal data-delay="180">
        <img src="/media/xlab-official-vr-demo.png" alt="X-Lab 工作人员指导学生使用 VR 设备" width="960" height="720" loading="lazy" />
        <figcaption data-ui>教学空间 · VR 设备指导</figcaption>
      </figure>
    </section>

    <!-- 平面图：整幅横向，是这一页的主体 -->
    <section class="map-block" data-reveal>
      <div class="map-bar">
        <h2>未来学习中心资源地图</h2>
        <div class="seg" role="group" aria-label="楼层筛选">
          <button v-for="f in floors" :key="f" type="button" :class="{ on: floor === f }" :aria-pressed="floor === f" @click="floor = f">{{ f }}</button>
        </div>
        <p class="map-hint">点空闲空间加入申请偏好</p>
      </div>

      <div
        class="map"
        data-ui
        :style="{ gridTemplateColumns: PLAN_COLS, gridTemplateRows: rowPlan.template }"
      >
        <button
          v-for="sp in floorSpaces"
          :key="sp.id"
          type="button"
          class="space"
          :class="spaceState(sp)"
          :style="spaceStyle(sp)"
          :disabled="!!spaceOccupant(sp) || sp.status === 'MAINTENANCE'"
          :aria-pressed="selectedSpaces.includes(sp.id)"
          :aria-label="`${sp.name}，${spaceOccupant(sp)?.name || (sp.status === 'MAINTENANCE' ? '维护中' : '可选')}`"
          @click="toggleSpace(sp)"
          @mouseenter="hoverSpace = sp"
          @mouseleave="hoverSpace = null"
        >
          <span class="space-id">{{ sp.id }}</span>
          <strong class="space-name">{{ sp.name }}</strong>
          <span class="space-kind">{{ SPACE_KIND[sp.kind] }} · {{ sp.capacity }} 人</span>
          <span v-if="spaceOccupant(sp)" class="space-owner">{{ spaceOccupant(sp).name }}</span>
          <span v-else-if="spaceState(sp) === 'selected'" class="space-owner">已选为偏好</span>
        </button>
      </div>

      <div class="map-foot">
        <span class="lg"><i class="d free" />可分配</span>
        <span class="lg"><i class="d occupied" />已入驻</span>
        <span class="lg"><i class="d selected" />已选为偏好</span>
        <span class="lg"><i class="d maintenance" />维护中</span>
        <span v-if="hoverSpace" class="map-note">
          {{ hoverSpace.id }} · {{ hoverSpace.zone }} · {{ hoverSpace.description }}
        </span>
        <span v-else class="map-note">
          {{ selectedSpaces.length ? `已选 ${activeSpaceHelp.n} 处 · 可容纳 ${activeSpaceHelp.cap} 人` : '空间偏好不等于自动占用，管理方分配后生效' }}
        </span>
      </div>
    </section>

    <!-- 下半：在驻项目 + 待评审申请 -->
    <div class="split">
      <section class="col" data-reveal>
        <div class="col-head">
          <h2>在驻项目</h2>
          <span>{{ activeProjects.length }}</span>
        </div>
        <ul class="list">
          <li v-for="(p, i) in activeProjects" :key="p.id" data-reveal :data-delay="i * 60">
            <router-link class="item" :to="`/projects/${p.id}`">
              <div class="item-main">
                <b>{{ p.name }}</b>
                <div class="item-top">
                  <span class="item-id">{{ p.id }}</span>
                  <StatusChip :label="label(PROJECT_STATUS, p.status)" :tone="tone(PROJECT_STATUS, p.status)" />
                </div>
                <p>{{ p.summary }}</p>
                <div class="item-tags">
                  <span v-for="sid in p.spaceIds" :key="sid">{{ ledger.spaceMap[sid]?.name || sid }}</span>
                </div>
              </div>
              <div class="item-side">
                <span>{{ fmtDate(p.startDate) }} → {{ fmtDate(p.endDate) }}</span>
                <span>{{ p.memberCount }} 人</span>
              </div>
            </router-link>
          </li>
          <li v-if="!activeProjects.length" class="empty-row">暂无在驻项目</li>
        </ul>
      </section>

      <section class="col" data-reveal data-delay="100">
        <div class="col-head">
          <h2>待评审申请</h2>
          <span>{{ pendingApps.length }}</span>
        </div>
        <ul class="list">
          <li v-for="(a, i) in pendingApps" :key="a.id" class="item" data-reveal :data-delay="i * 60">
            <div class="item-main">
              <div class="item-top">
                <StatusChip :label="label(APPLICATION_STATUS, a.status)" :tone="tone(APPLICATION_STATUS, a.status)" />
                <span class="item-id">{{ a.id }}</span>
              </div>
              <b>{{ a.projectName }}</b>
              <p>{{ a.ideaSummary }}</p>
              <div class="item-tags">
                <span>{{ a.memberCount }} 人</span>
                <span>申请 {{ a.requestedSeats }} 工位</span>
                <span>{{ fmtDate(a.startDate) }} 起</span>
              </div>
            </div>
          </li>
          <li v-if="!pendingApps.length" class="empty-row">没有待评审申请</li>
        </ul>
        <router-link v-if="pendingApps.length" to="/console" class="col-link">
          前往管理台评审 →
        </router-link>
      </section>
    </div>


    <AppModal
      v-if="showApply"
      title="提交项目入驻申请"
      subtitle="说明项目计划以及需要使用哪些空间"
      width="700px"
      @close="showApply = false"
    >
      <div class="modal-grid">
        <div class="field span-2">
          <label for="apply-project-name">项目名称</label>
          <input id="apply-project-name" v-model="form.projectName" name="projectName" autocomplete="off" required placeholder="例如：无障碍校园地图…" />
        </div>
        <div class="field">
          <label for="apply-lead">项目负责人</label>
          <input id="apply-lead" v-model="form.leadName" name="leadName" autocomplete="name" required readonly />
        </div>
        <div class="field">
          <label for="apply-members">团队成员（逗号分隔）</label>
          <input id="apply-members" v-model="form.membersText" name="members" autocomplete="off" placeholder="例如：林同学、周同学、沈同学…" />
        </div>
        <div class="field">
          <label for="apply-seats">申请工位数</label>
          <input id="apply-seats" v-model.number="form.requestedSeats" name="requestedSeats" type="number" inputmode="numeric" min="1" autocomplete="off" required />
        </div>
        <div class="field">
          <label for="apply-spaces">偏好空间</label>
          <input id="apply-spaces" name="preferredSpaces" :value="selectedSpaces.length ? selectedSpaces.join(', ') : '在地图上点选'" readonly style="cursor: pointer" />
        </div>
        <div class="field">
          <label for="apply-start">开始日期</label>
          <input id="apply-start" v-model="form.startDate" name="startDate" type="date" autocomplete="off" required />
        </div>
        <div class="field">
          <label for="apply-end">结束日期</label>
          <input id="apply-end" v-model="form.endDate" name="endDate" type="date" autocomplete="off" required />
        </div>
        <div class="field span-2">
          <label for="apply-idea">项目说明</label>
          <textarea id="apply-idea" v-model="form.ideaSummary" name="ideaSummary" autocomplete="off" required placeholder="解决什么问题、准备做出什么、如何验收…" />
        </div>
        <div class="field span-2">
          <label for="apply-milestone">阶段目标</label>
          <input id="apply-milestone" v-model="form.milestone" name="milestone" autocomplete="off" placeholder="例如：完成可操作原型与一次现场测试…" />
        </div>
      </div>

      <div class="note-box">
        提交后，管理方先评审项目，再分配工位并检查时间冲突。
        空间偏好不等于自动占用，以管理方分配结果为准。
      </div>

      <template #foot>
        <button class="btn btn-ghost" @click="showApply = false">取消</button>
        <button class="btn btn-primary" :disabled="submitting" @click="submitApply">
          {{ submitting ? '提交中…' : '提交评审' }}
        </button>
      </template>
    </AppModal>
  </div>
</template>

<style scoped>
/* ---------- 首屏 ---------- */
.rv-hero {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 0.86fr);
  gap: 56px;
  align-items: stretch;
}

.rv-copy { display: flex; flex-direction: column; justify-content: center; padding: 18px 0 8px; }

.rv-copy h1 {
  font-family: var(--serif);
  font-size: clamp(40px, 5.4vw, 84px);
  font-weight: 500;
  line-height: 1.06;
  letter-spacing: -0.02em;
  color: var(--ink);
}

.rv-sub {
  margin-top: 26px;
  font-size: 16px;
  line-height: 1.72;
  color: var(--ink-3);
  max-width: 42ch;
}

/* 账目：数字与说明同行，标签不压数字 */
.rv-tally {
  list-style: none;
  margin: 38px 0 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 26px;
}

.rv-tally li {
  display: flex;
  align-items: baseline;
  gap: 9px;
  padding: 0;
  background: transparent;
}

.rv-tally li.warn b,
.rv-tally li.warn span { color: var(--warn); }

.rv-tally b {
  font-family: var(--serif);
  font-size: 32px;
  font-weight: 500;
  letter-spacing: -0.02em;
  line-height: 1;
  color: var(--ink);
  font-variant-numeric: tabular-nums;
}

.rv-tally span { font-size: 14px; color: var(--ink-2); white-space: nowrap; }

.rv-cta { align-self: flex-start; margin-top: 38px; }

.rv-photo { margin: 0; position: relative; overflow: hidden; min-height: 360px; border-radius: 18px; }
.rv-photo img { width: 100%; height: 100%; object-fit: cover; display: block; }

.rv-photo::after,
.rv-strip figure::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(0deg, rgba(5, 18, 31, 0.7), transparent 42%); /* taste-skill-ignore: image caption scrim */
}

.rv-photo figcaption,
.rv-strip figcaption {
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

/* ---------- 空间实景 ---------- */
.rv-strip {
  margin: 60px calc(var(--gutter) * -1) 0;
  width: auto;
  display: grid;
  grid-template-columns: 1.2fr 1fr 1fr;
  gap: 3px;
  height: 38vh;
  min-height: 270px;
}

.rv-strip figure { margin: 0; position: relative; overflow: hidden; border-radius: 14px; }

.rv-strip img,
.rv-strip video {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
  transition: transform 0.9s cubic-bezier(0.16, 1, 0.3, 1);
}

.rv-strip figure:hover img,
.rv-strip figure:hover video { transform: scale(1.05); }

/* 地图块：满幅白色账页 */
.map-block {
  margin: 3px calc(var(--gutter) * -1) 0;
  width: auto;
  background: var(--surface);
  border-radius: 18px;
  overflow: hidden;
  padding: 40px var(--gutter) 44px;
}

.map-bar {
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 22px 26px 18px;
}

.map-bar h2 { font-family: var(--serif); font-size: 28px; font-weight: 500; letter-spacing: -0.015em; }
.map-bar .seg { margin-left: auto; }

.map-hint { font-size: 14px; color: var(--ink-3); }

.map {
  display: grid;
  gap: 10px;
  background: #d7e0e7;
  padding: 18px;
  min-height: 430px;
}

.space {
  position: relative;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
  gap: 3px;
  padding: 15px 17px;
  border: 0;
  background: var(--surface);
  border-radius: 12px;
  cursor: pointer;
  text-align: left;
  font-family: var(--sans);
  transition: background 0.18s ease, color 0.18s ease;
  overflow: hidden;
}

.space:hover { background: var(--blue-soft); }

.space-id { font-size: 14px; color: var(--ink-4); font-variant-numeric: tabular-nums; }

.space-name {
  font-size: 16px;
  font-weight: 620;
  letter-spacing: -0.015em;
  color: var(--ink);
  line-height: 1.3;
}

.space-kind { font-size: 14px; color: var(--ink-3); }

.space-owner {
  margin-top: auto;
  font-size: 14px;
  color: var(--ink-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 已入驻：整块实心蓝，白字 7.33:1 */
.space.occupied {
  background: var(--blue-field);
  cursor: not-allowed;
}
.space.occupied:hover { background: var(--blue-field); }
.space.occupied .space-id,
.space.occupied .space-kind,
.space.occupied .space-name,
.space.occupied .space-owner { color: #fff; }
.space.occupied .space-id,
.space.occupied .space-kind { color: rgba(255, 255, 255, 0.82); }

/* 已选为偏好：整块实心青绿，墨字 8.4:1 */
.space.selected { background: var(--mint); }
.space.selected .space-kind { color: var(--ink-2); }
.space.selected .space-owner { color: var(--ink); font-weight: 600; }

.space.maintenance { background: var(--surface-3); cursor: not-allowed; }
.space.maintenance:hover { background: var(--surface-3); }
.space.maintenance .space-name { color: var(--ink-3); }

.map-foot {
  display: flex;
  align-items: center;
  gap: 22px;
  flex-wrap: wrap;
  padding: 18px 26px 22px;
}

.lg {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  color: var(--ink-2);
}

.d { width: 12px; height: 12px; display: inline-block; border-radius: 3px; background: var(--surface); }
.d.free { background: var(--surface-3); }
.d.occupied { background: var(--blue-field); }
.d.selected { background: var(--mint); }
.d.maintenance { background: var(--surface-2); }

.map-note {
  margin-left: auto;
  font-size: 14px;
  color: var(--ink-3);
  max-width: 46ch;
  text-align: right;
}

/* 下半双栏 */
.split {
  display: grid;
  grid-template-columns: 1.28fr 1fr;
  gap: 3px;
  margin-top: 3px;
}

.col { background: var(--surface); padding: 26px 26px 8px; border-radius: 16px; }

.col-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 14px;
}

.col-head h2 { font-family: var(--serif); font-size: 28px; font-weight: 500; letter-spacing: -0.015em; }
.col-head span { font-size: 14px; color: var(--ink-3); font-variant-numeric: tabular-nums; }

.list { list-style: none; margin: 0; padding: 0; display: grid; gap: 2px; }

.item {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 24px;
  padding: 20px 22px;
  background: var(--surface-2);
  border-radius: 10px;
  cursor: pointer;
  color: inherit;
  text-decoration: none;
  transition: background 0.16s ease, transform 0.18s cubic-bezier(0.16, 1, 0.3, 1);
}

.item:hover { background: var(--bg); transform: translateX(2px); }

.list > li:last-child .item { margin-bottom: 26px; }

.item-top { display: flex; align-items: center; gap: 10px; margin-top: 7px; }

.item-id { font-size: 14px; color: var(--ink-3); font-variant-numeric: tabular-nums; }

.item b {
  display: block;
  font-size: 16px;
  font-weight: 620;
  letter-spacing: -0.02em;
  color: var(--ink);
}

.item p {
  margin-top: 9px;
  font-size: 14px;
  line-height: 1.65;
  color: var(--ink-2);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.item-tags { display: flex; gap: 8px; flex-wrap: wrap; margin-top: 12px; }

.item-tags span {
  font-size: 14px;
  padding: 0;
  background: transparent;
  color: var(--blue-deep);
}

.item-tags span + span::before { content: '·'; margin-right: 8px; color: var(--ink-4); }

.item-side {
  display: grid;
  gap: 5px;
  justify-items: end;
  align-content: start;
  font-size: 14px;
  color: var(--ink-3);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.empty-row {
  padding: 30px 22px;
  background: var(--surface-2);
  border-radius: 10px;
  font-size: 14px;
  color: var(--ink-3);
  margin-bottom: 22px;
}

.col-link {
  display: inline-block;
  margin: 8px 0 22px;
  font-size: 16px;
  font-weight: 600;
  color: var(--blue-deep);
  text-decoration: none;
}

/* 表单 */
.modal-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.span-2 { grid-column: span 2; }

.note-box {
  margin-top: 18px;
  padding: 14px 16px;
  background: var(--blue-soft);
  border-radius: 10px;
  font-size: 14px;
  color: var(--ink-2);
  line-height: 1.7;
}

@media (max-width: 1180px) {
  .rv-hero { grid-template-columns: 1fr; gap: 32px; }
  .rv-photo { min-height: 300px; max-height: 380px; }
  /* 单列时列表项不再各自成块，交给留白分隔 */
  .item { background: transparent; padding-left: 0; }
  .item:hover { background: transparent; transform: translateX(2px); }
  .split { grid-template-columns: 1fr; }
  .map-bar { flex-wrap: wrap; gap: 12px; }
  .map-bar .seg { margin-left: 0; }
  .map-note { margin-left: 0; text-align: left; }
}

@media (max-width: 720px) {
  .rv-copy h1 { font-size: 36px; }
  .rv-tally { gap: 18px; }
  .rv-tally b { font-size: 24px; }
  .rv-tally span { font-size: 14px; }
  .rv-strip { grid-template-columns: 1fr; height: auto; }
  .rv-strip figure { height: 180px; }
  .map-block { padding: 24px 16px 26px; }
  /* 手机端平面图退化为单列清单：两列同宽色块会被读成一排卡片 */
  .map { min-height: 0; grid-template-columns: 1fr !important; grid-template-rows: none !important; gap: 2px; padding: 8px; }
  .space { grid-column: auto !important; grid-row: auto !important; min-height: 0; padding: 13px 14px; }
  .item { grid-template-columns: 1fr; gap: 10px; }
  .item-side { justify-items: start; }
  .modal-grid { grid-template-columns: 1fr; }
  .span-2 { grid-column: span 1; }
}
</style>
