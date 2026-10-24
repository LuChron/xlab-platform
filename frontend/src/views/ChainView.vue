<script setup>
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useLedger } from '../lib/store'
import { useReveal } from '../lib/reveal'
import { usePointerTilt } from '../lib/pointer'
import { useParallax } from '../lib/parallax'
import CountUp from '../components/CountUp.vue'
import { ACTION_LABEL, fmtTime, shortHash, whoIs, orgName, fnLabel } from '../lib/labels'
import AppModal from '../components/AppModal.vue'
import { scrollElementIntoView } from '../lib/motion'

const ledger = useLedger()
const route = useRoute()
const router = useRouter()

useReveal()
usePointerTilt()
useParallax('[data-parallax]')

const validTypes = new Set(['ALL', 'application', 'project', 'borrow', 'equipment'])
const q = ref(typeof route.query.q === 'string' ? route.query.q : '')
const filter = ref(validTypes.has(route.query.type) ? route.query.type : 'ALL')

watch([q, filter], ([nextQ, nextFilter]) => {
  const query = { ...route.query }
  if (nextQ) query.q = nextQ
  else delete query.q
  if (nextFilter !== 'ALL') query.type = nextFilter
  else delete query.type
  router.replace({ query })
})

const types = [
  { key: 'ALL', label: '全部' },
  { key: 'application', label: '入驻申请' },
  { key: 'project', label: '项目' },
  { key: 'borrow', label: '设备交接' },
  { key: 'equipment', label: '设备台账' },
]

const filtered = computed(() =>
  ledger.activities.filter((a) => {
    const okT = filter.value === 'ALL' || a.objectType === filter.value
    const text = `${a.txId} ${a.action} ${a.objectId} ${a.detail} ${a.actor}`.toLowerCase()
    return okT && text.includes(q.value.toLowerCase())
  })
)

// 流水默认只放一屏（12 笔），其余按需展开：整页滚动更短，也避免 data-ui 包裹超过一屏半
const PAGE = 12
const shown = ref(PAGE)
watch([q, filter], () => { shown.value = PAGE })
const visible = computed(() => filtered.value.slice(0, shown.value))

const counts = computed(() => {
  const c = {}
  for (const a of ledger.activities) c[a.objectType] = (c[a.objectType] || 0) + 1
  return c
})

const actionCount = computed(() => {
  const m = {}
  for (const a of ledger.activities) m[a.action] = (m[a.action] || 0) + 1
  const total = ledger.activities.length || 1
  return Object.entries(m)
    .sort((a, b) => b[1] - a[1])
    .slice(0, 6)
    .map(([act, n]) => ({ act, n, pct: Math.round((n / total) * 100) }))
})

// 真实区块：qscc GetChainInfo + GetBlockByNumber 解码得到，倒序展示（最新在前）
const chainBlocks = computed(() => (ledger.chain?.blocks || []).slice().reverse())
const chainHeight = computed(() => ledger.chain?.height || 0)

// 校验哈希链是否首尾相接：区块 N 的 hash 必须等于区块 N+1 的 prevHash
const chainSealed = computed(() => {
  const blocks = ledger.chain?.blocks || []
  if (blocks.length < 2) return false
  for (let i = 1; i < blocks.length; i++) {
    if (blocks[i].prevHash !== blocks[i - 1].hash) return false
  }
  return true
})

// 首屏的主行动：跳到流水表本身，而不是停在标题上
function jumpToJournal() {
  scrollElementIntoView(document.querySelector('.journal'))
}

// 业务对象：编号本身是时间戳，前缀相同无从区分，右侧补上人能读的名字
function objectName(a) {
  if (a.objectType === 'project') return ledger.projectMap[a.objectId]?.name || ''
  if (a.objectType === 'borrow') return ledger.borrowMap[a.objectId]?.equipmentName || ''
  if (a.objectType === 'equipment') return ledger.equipmentMap[a.objectId]?.name || ''
  if (a.objectType === 'application') {
    const app = ledger.applications.find((x) => x.id === a.objectId)
    return app?.projectName || ''
  }
  return ''
}

const detail = ref(null)
const orgCount = computed(() => new Set(ledger.activities.map((a) => a.actorMsp)).size)

// 区块里某笔交易对应的中文操作名与组织
function blockTxLabel(tx) {
  return fnLabel(tx.fn)
}
</script>

<template>
  <div class="chainview">
    <!-- 首屏 -->
    <header class="ch-hero">
      <div class="ch-copy" data-reveal>
        <h1>链上记录</h1>
        <p class="ch-sub">
          这里显示申请、审批、交接和验收记录。链上保存编号、状态、摘要和确认人；正文与照片仍存放在链下。
        </p>
        <ul class="ch-tally">
          <li data-reveal data-delay="40"><b><CountUp :to="ledger.activities.length" /></b><span>笔链上记录</span></li>
          <li data-reveal data-delay="90"><b><CountUp :to="orgCount" /></b><span>个参与组织</span></li>
          <li data-reveal data-delay="140"><b><CountUp :to="chainHeight" /></b><span>个区块高度</span></li>
        </ul>

        <div class="ch-actions">
          <button class="ch-cta" type="button" @click="jumpToJournal()">直接看流水</button>
          <router-link class="ch-alt" to="/equipment">回到设备目录</router-link>
        </div>
      </div>

      <figure data-parallax class="ch-photo" data-reveal data-delay="120">
        <img src="/media/pittsboro-merchant-business-ledger.jpg" alt="1832 年商号的手写账本，竖排逐笔记录" width="1920" height="2881" />
        <figcaption data-ui>1832 年商号手写流水账</figcaption>
      </figure>
    </header>

    <!-- 通道信息：一行三格，事实摆在正文上 -->
    <section class="ch-meta-row" data-reveal>
      <div class="m">
        <span>通道</span>
        <b>{{ ledger.channel || 'xlabchannel' }}</b>
      </div>
      <div class="m">
        <span>链码</span>
        <b>{{ ledger.chaincode || 'xlab' }} 1.0</b>
      </div>
      <div class="m">
        <span>数据来源</span>
        <b>{{ ledger.mode === 'fabric' ? 'Hyperledger Fabric' : '演示账本' }}</b>
      </div>
      <div class="m">
        <span>当前区块</span>
        <b v-if="ledger.mode === 'fabric'">#{{ chainHeight }} · {{ shortHash(ledger.chain?.currentHash, 8, 6) }}</b>
        <b v-else>无（演示模式）</b>
      </div>
    </section>

    <!-- 满幅：账本里的东西从哪来 -->
    <section data-parallax class="ch-strip" aria-label="被记录的工作现场">
      <figure data-reveal>
        <img src="/media/xlab-official-opening.png" alt="浙江大学图书馆 X-Lab 的开放体验空间" width="960" height="720" loading="lazy" />
        <figcaption data-ui>真实场景 · X-Lab 开放体验</figcaption>
      </figure>
      <figure data-reveal data-delay="90">
        <img src="/media/xlab-official-experience.png" alt="学生在 X-Lab 进行沉浸式绘画创作" width="960" height="640" loading="lazy" />
        <figcaption data-ui>真实使用 · 沉浸式创作</figcaption>
      </figure>
      <figure data-reveal data-delay="180">
        <img src="/media/xlab-official-mr-demo.png" alt="学生在 X-Lab 使用混合现实设备" width="960" height="640" loading="lazy" />
        <figcaption data-ui>真实设备 · MR 交互</figcaption>
      </figure>
    </section>

    <div class="ch-body">
      <!-- 流水：一页真正的账 -->
      <section class="journal" data-ui>
        <div class="j-bar">
          <div class="seg" role="group" aria-label="链上记录类型筛选">
            <button v-for="t in types" :key="t.key" type="button" :class="{ on: filter === t.key }" :aria-pressed="filter === t.key" @click="filter = t.key">
              {{ t.label }}<i>{{ t.key === 'ALL' ? ledger.activities.length : counts[t.key] || 0 }}</i>
            </button>
          </div>
          <div class="search">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="11" cy="11" r="7" /><path d="M21 21l-4-4" />
            </svg>
            <input v-model="q" name="ledgerSearch" autocomplete="off" spellcheck="false" aria-label="搜索链上记录" placeholder="搜索哈希、对象或操作…" />
          </div>
        </div>

        <table class="j-table">
          <colgroup>
            <col style="width: 18%" />
            <col style="width: 18%" />
            <col style="width: 27%" />
            <col style="width: 19%" />
            <col style="width: 18%" />
          </colgroup>
          <thead>
            <tr>
              <th scope="col">时间</th>
              <th scope="col">操作</th>
              <th scope="col">业务对象</th>
              <th scope="col">交易哈希</th>
              <th scope="col">调用方</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="(a, i) in visible"
              :key="a.txId"
              data-reveal
              :data-delay="Math.min(i, 8) * 40"
            >
              <td class="c-time">{{ fmtTime(a.timestamp) }}</td>
              <td class="c-act">{{ ACTION_LABEL[a.action] || a.action }}</td>
              <td class="c-obj">
                <template v-if="objectName(a)">
                  <b class="obj-name">{{ objectName(a) }}</b>
                  <span class="obj-id">{{ a.objectId }}</span>
                </template>
                <b v-else class="obj-name obj-plain">{{ a.objectId }}</b>
              </td>
              <td class="c-hash">
                <button type="button" @click="detail = a" :aria-label="`查看交易 ${a.txId}`">
                  {{ shortHash(a.txId, 6, 4) }}
                </button>
              </td>
              <td class="c-actor">
                <i class="org-dot" :class="a.actorMsp === 'Org1MSP' ? 'org-1' : 'org-2'" aria-hidden="true"></i>
                {{ orgName(a.actorMsp) }}
              </td>
            </tr>
          </tbody>
        </table>

        <div v-if="!filtered.length" class="j-empty">没有匹配的交易</div>

        <div class="j-foot">
          <span>共 {{ filtered.length }} 笔 · 点击交易哈希查看原始记录</span>
          <button v-if="filtered.length > visible.length" class="j-more" @click="shown = filtered.length">
            展开全部 {{ filtered.length }} 笔
          </button>
        </div>
      </section>

      <!-- 侧栏 -->
      <aside class="side">
        <div class="side-block" data-reveal>
          <h2>操作分布</h2>
          <div v-for="a in actionCount" :key="a.act" class="bar">
            <div class="bar-top">
              <span>{{ ACTION_LABEL[a.act] || a.act }}</span>
              <b>{{ a.n }}</b>
            </div>
            <div class="bar-track"><i :style="{ width: a.pct + '%' }" /></div>
          </div>
          <p v-if="!actionCount.length" class="side-empty">暂无数据</p>
        </div>

        <div class="side-block" data-reveal data-delay="90">
          <h2>区块</h2>
          <p class="side-note" v-if="ledger.mode === 'fabric'">
            每个区块都包含前一区块的哈希，因此可以检查记录是否被改动。
            <b v-if="chainSealed" class="seal seal-ok">哈希链闭合</b>
            <b v-else class="seal">链上暂无足够区块</b>
          </p>
          <p class="side-note" v-else>演示模式没有真实区块，连接 Fabric 后这里显示真实出块与哈希链。</p>
          <div v-for="b in chainBlocks" :key="b.number" class="blk">
            <div class="blk-top">
              <b>#{{ b.number }}</b>
              <span>{{ b.txCount }} 笔</span>
            </div>
            <span class="blk-hash">{{ shortHash(b.hash, 10, 6) }}</span>
            <span class="blk-tx" v-if="b.txs.length">{{ fnLabel(b.txs[0].fn) }}</span>
          </div>
          <p v-if="ledger.mode === 'fabric' && !chainBlocks.length" class="side-empty">暂无区块</p>
        </div>
      </aside>
    </div>


    <AppModal v-if="detail" title="交易详情" :subtitle="ACTION_LABEL[detail.action] || detail.action" @close="detail = null">
      <dl class="kv-list">
        <div class="kv"><dt>交易哈希</dt><dd class="break">{{ detail.txId }}</dd></div>
        <div class="kv"><dt>业务操作</dt><dd>{{ ACTION_LABEL[detail.action] || detail.action }}</dd></div>
        <div class="kv"><dt>对象类型</dt><dd>{{ detail.objectType }}</dd></div>
        <div class="kv"><dt>对象编号</dt><dd>{{ detail.objectId }}</dd></div>
        <div class="kv"><dt>操作详情</dt><dd>{{ detail.detail || '暂无' }}</dd></div>
        <div class="kv"><dt>调用方身份</dt><dd>{{ whoIs(detail.actor) }}</dd></div>
        <div class="kv"><dt>所属组织</dt><dd>{{ orgName(detail.actorMsp) }}</dd></div>
        <div class="kv"><dt>时间戳</dt><dd>{{ fmtTime(detail.timestamp) }}</dd></div>
      </dl>
    </AppModal>
  </div>
</template>

<style scoped>
/* ---------- 首屏 ---------- */
.ch-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.06fr) minmax(0, 0.72fr);
  gap: 56px;
  align-items: stretch;
}

.ch-copy { display: flex; flex-direction: column; justify-content: center; padding: 18px 0 8px; }

.ch-copy h1 {
  font-family: var(--serif);
  font-size: clamp(40px, 5.4vw, 84px);
  font-weight: 500;
  line-height: 1.06;
  letter-spacing: -0.02em;
  color: var(--ink);
}

.ch-sub {
  margin-top: 26px;
  font-size: 16px;
  line-height: 1.72;
  color: var(--ink-3);
  max-width: 46ch;
}

.ch-actions {
  margin-top: 30px;
  display: flex;
  align-items: center;
  gap: 26px;
  flex-wrap: wrap;
}

.ch-cta {
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

.ch-cta:hover { background: var(--blue); }

.ch-alt {
  font-size: 16px;
  font-weight: 550;
  color: var(--ink-2);
  text-decoration: none;
  transition: color 0.16s ease;
}

.ch-alt:hover { color: var(--blue-deep); }

.ch-tally {
  list-style: none;
  margin: 38px 0 0;
  padding: 0;
  display: flex;
  flex-wrap: wrap;
  gap: 26px;
}

.ch-tally li {
  display: flex;
  align-items: baseline;
  gap: 9px;
  padding: 0;
  background: transparent;
}

.ch-tally b {
  font-family: var(--serif);
  font-size: 32px;
  font-weight: 500;
  letter-spacing: -0.02em;
  line-height: 1;
  color: var(--ink);
  font-variant-numeric: tabular-nums;
}

.ch-tally span { font-size: 14px; color: var(--ink-2); white-space: nowrap; }

.ch-photo { margin: 0; position: relative; overflow: hidden; min-height: 380px; max-height: 520px; border-radius: 18px; }

.ch-photo img { width: 100%; height: 100%; object-fit: cover; object-position: center 22%; display: block; }

.ch-photo::after,
.ch-strip figure::after {
  content: '';
  position: absolute;
  inset: 0;
  pointer-events: none;
  background: linear-gradient(0deg, rgba(5, 18, 31, 0.7), transparent 42%); /* taste-skill-ignore: image caption scrim */
}

.ch-photo figcaption {
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

/* ---------- 通道信息 ---------- */
.ch-meta-row {
  margin: 56px calc(var(--gutter) * -1) 3px;
  width: auto;
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 3px;
  background: #0B1526;
  border-radius: 14px;
  overflow: hidden;
}

.ch-meta-row .m {
  display: flex;
  align-items: baseline;
  gap: 12px;
  padding: 22px var(--gutter) 24px;
  background: rgba(255, 255, 255, 0.05);
}

.ch-meta-row .m:first-child { padding-left: var(--gutter); }

.ch-meta-row span { font-size: 14px; color: rgba(255, 255, 255, 0.74); white-space: nowrap; }

.ch-meta-row b {
  font-size: 16px;
  font-weight: 620;
  color: #fff;
  letter-spacing: -0.015em;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ---------- 满幅图带 ---------- */
.ch-strip {
  margin: 0 calc(var(--gutter) * -1) 3px;
  width: auto;
  display: grid;
  grid-template-columns: 1.1fr 1.28fr 0.98fr;
  gap: 3px;
  height: 40vh;
  min-height: 300px;
}

.ch-strip figure { margin: 0; position: relative; overflow: hidden; border-radius: 14px; }
.ch-strip img,
.ch-strip video { width: 100%; height: 100%; object-fit: cover; display: block; }
.ch-strip figure:hover img,
.ch-strip figure:hover video { transform: scale(1.04); }
.ch-strip img,
.ch-strip video { transition: transform 0.7s cubic-bezier(0.16, 1, 0.3, 1); }

.ch-strip figcaption {
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
.ch-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  gap: 3px;
  align-items: start;
}

.journal { background: var(--surface); min-width: 0; border-radius: 16px; overflow: hidden; }

.j-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px 24px 18px;
  flex-wrap: wrap;
}

.j-bar .search { flex: 1; min-width: 220px; }

.seg button i {
  font-style: normal;
  margin-left: 6px;
  color: var(--ink-4);
  font-variant-numeric: tabular-nums;
}
.seg button.on i { color: #c8d8e7; }

/* 表格账本 */
.j-table {
  width: 100%;
  border-collapse: collapse;
  table-layout: fixed;
}

.j-table th {
  text-align: left;
  font-size: 14px;
  font-weight: 500;
  color: var(--ink-3);
  background: var(--bg);
  padding: 12px 14px;
}

.j-table td {
  padding: 13px 14px;
  border-top: 1px solid var(--surface-2);
  font-size: 14px;
  color: var(--ink-2);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.j-table tbody tr { transition: background 0.15s ease; }

.j-table tbody tr:hover { background: var(--blue-soft); }

.j-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  padding: 14px 24px 16px;
  font-size: 14px;
  color: var(--ink-3);
}

.j-more {
  border: 0;
  font-family: var(--sans);
  font-size: 14px;
  font-weight: 600;
  color: var(--blue-deep);
  background: var(--blue-soft);
  padding: 9px 16px;
  cursor: pointer;
  transition: background 0.16s ease;
}
.j-more:hover { background: var(--surface-3); }

.c-time { color: var(--ink-3); font-variant-numeric: tabular-nums; white-space: nowrap; }
.c-act { color: var(--ink); font-weight: 600; }
/* 业务对象：名字在上、编号在下。编号是时间戳，截断后彼此无法区分，必须给全文 */
.c-obj { white-space: normal; }
.obj-name {
  display: block;
  font-weight: 550;
  color: var(--ink);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.obj-plain { font-variant-numeric: tabular-nums; font-weight: 400; color: var(--ink-2); }
.obj-id {
  display: block;
  margin-top: 2px;
  font-size: 13px;
  color: var(--ink-3);
  font-variant-numeric: tabular-nums;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.c-hash { font-variant-numeric: tabular-nums; font-size: 14px; }
.c-hash button {
  border: 0;
  padding: 5px 0;
  background: transparent;
  color: var(--blue-deep);
  font: inherit;
  font-variant-numeric: inherit;
  cursor: pointer;
}
.c-hash button:hover { color: var(--blue); text-decoration: underline; text-underline-offset: 3px; }
.c-actor {
  color: var(--ink-3);
  font-size: 14px;
  white-space: nowrap;
  overflow: visible;
  text-overflow: clip;
}
.org-dot {
  display: inline-block;
  width: 7px; height: 7px;
  border-radius: 999px;
  margin-right: 7px;
  vertical-align: 1px;
}
.org-1 { background: var(--blue); }
.org-2 { background: var(--mint-deep); }

.j-empty { padding: 54px 24px; color: var(--ink-3); font-size: 16px; }

/* 侧栏 */
.side { display: grid; gap: 3px; }

.side-block { background: var(--surface); padding: 22px 24px 26px; border-radius: 14px; }

.side-block h2 {
  font-size: 22px;
  font-weight: 620;
  letter-spacing: -0.025em;
  margin-bottom: 18px;
}

.bar { margin-bottom: 16px; }

.bar-top {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 14px;
  color: var(--ink-2);
  margin-bottom: 7px;
}
.bar-top b { color: var(--ink); font-variant-numeric: tabular-nums; }

.bar-track { height: 6px; background: var(--surface-2); }
.bar-track i { display: block; height: 100%; background: var(--blue); transition: width 0.5s cubic-bezier(0.16, 1, 0.3, 1); }

.side-note {
  font-size: 14px;
  line-height: 1.65;
  color: var(--ink-3);
  margin: -8px 0 16px;
}

.blk { padding: 12px 14px; background: var(--surface-2); margin-bottom: 4px; border-radius: 9px; transition: background 0.16s ease; }
.blk:hover { background: var(--bg); }
.blk-top { display: flex; justify-content: space-between; align-items: center; gap: 10px; }
.blk-top b { font-size: 16px; color: var(--ink); font-variant-numeric: tabular-nums; }
.blk-top span { font-size: 14px; color: var(--ink-3); }
.blk-hash {
  display: block;
  margin-top: 5px;
  font-size: 14px;
  color: var(--ink-3);
  font-variant-numeric: tabular-nums;
}

.blk-tx {
  display: block;
  margin-top: 3px;
  font-size: 14px;
  color: var(--ink-2);
  font-weight: 550;
}

.seal {
  display: block;
  margin-top: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--ink-3);
}
.seal-ok { color: var(--ok); }

.side-empty { font-size: 14px; color: var(--ink-3); }

/* 详情 */
.kv-list { margin: 0; display: grid; }
.kv {
  display: grid;
  grid-template-columns: 108px minmax(0, 1fr);
  gap: 18px;
  padding: 12px 0;
  font-size: 14px;
}
.kv dt { color: var(--ink-3); }
.kv dd { margin: 0; color: var(--ink); text-align: right; font-variant-numeric: tabular-nums; }
.break { word-break: break-all; }

@media (max-width: 1180px) {
  .ch-hero { grid-template-columns: 1fr; gap: 32px; }
  .ch-photo { min-height: 300px; max-height: 380px; }
  /* 深色元信息条在手机上改成两块一行，四列会撑出 145px */
  .ch-meta-row { grid-template-columns: 1fr 1fr; overflow: hidden; }
  .ch-strip { grid-template-columns: 1fr 1fr; height: 34vh; }
  .ch-strip figure:last-child { display: none; }
  .ch-body { grid-template-columns: 1fr; }
  .j-table th:nth-child(4), .j-table td:nth-child(4) { display: none; }
}

@media (max-width: 720px) {
  .ch-copy h1 { font-size: 36px; }
  .ch-tally { gap: 18px; }
  .ch-tally b { font-size: 24px; }
  .ch-tally span { font-size: 14px; }
  /* 深色元信息条在手机上改成两块一行，四列会撑出 145px */
  .ch-meta-row { grid-template-columns: 1fr 1fr; overflow: hidden; }
  .ch-meta-row .m { padding: 16px 20px 18px; }
  .ch-meta-row span { font-size: 14px; }
  .ch-meta-row b { font-size: 14px; }
  .j-table th:nth-child(5), .j-table td:nth-child(5) { display: none; }
  .j-table th, .j-table td { padding: 12px 12px; }
  .j-bar { padding: 16px 14px 14px; }
  .j-bar .search { min-width: 0; }
  /* 筛选段里的计数在窄屏会顶出容器 */
  .seg { flex-wrap: wrap; }
  .seg button { padding: 7px 11px; font-size: 14px; }
  .seg button i { display: none; }
}
</style>
