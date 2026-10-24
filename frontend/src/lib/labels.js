// 状态与枚举的中文标签、语义色映射，全站统一使用

export const EQUIPMENT_STATUS = {
  AVAILABLE: { label: '可借用', tone: 'ok' },
  REQUESTED: { label: '申请中', tone: 'warn' },
  APPROVED: { label: '待领取', tone: 'warn' },
  BORROWED: { label: '借用中', tone: 'info' },
  MAINTENANCE: { label: '维护中', tone: 'bad' },
  RETIRED: { label: '已报废', tone: 'idle' },
}

export const EQUIPMENT_CONDITION = {
  GOOD: '良好',
  FAIR: '一般',
  DAMAGED: '损坏',
}

export const BORROW_STATUS = {
  PENDING: { label: '待审批', tone: 'warn' },
  APPROVED: { label: '待领取', tone: 'warn' },
  REJECTED: { label: '已驳回', tone: 'bad' },
  BORROWED: { label: '已借出', tone: 'info' },
  RETURN_REQUESTED: { label: '待验收', tone: 'warn' },
  RETURNED: { label: '已归还', tone: 'ok' },
  EXCEPTION: { label: '异常待处理', tone: 'bad' },
  RESOLVED: { label: '已处理', tone: 'ok' },
}

export const APPLICATION_STATUS = {
  SUBMITTED: { label: '待评审', tone: 'warn' },
  APPROVED: { label: '已通过·待分配', tone: 'ok' },
  REJECTED: { label: '已驳回', tone: 'bad' },
  ACTIVATED: { label: '已入驻', tone: 'ok' },
}

export const PROJECT_STATUS = {
  ACTIVE: { label: '入驻中', tone: 'ok' },
  EXTENDED: { label: '已续期', tone: 'info' },
  CLOSED: { label: '已结项', tone: 'idle' },
}

export const SPACE_STATUS = {
  OPEN: { label: '可分配', tone: 'ok' },
  MAINTENANCE: { label: '维护中', tone: 'bad' },
}

export const SPACE_KIND = {
  DESK: '工位',
  BENCH: '工程台',
  ZONE: '功能区',
}

export const ACTION_LABEL = {
  INIT_LEDGER: '初始化账本',
  CREATE_EQUIPMENT: '新增设备',
  UPDATE_EQUIPMENT_STATUS: '调整设备状态',
  CREATE_SPACE: '新增空间',
  SUBMIT_APPLICATION: '提交入驻申请',
  REVIEW_APPLICATION: '评审入驻申请',
  ACTIVATE_PROJECT: '分配空间并入驻',
  EXTEND_PROJECT: '项目续期',
  CLOSE_PROJECT: '项目结项',
  SUBMIT_MILESTONE: '提交阶段成果',
  REQUEST_BORROW: '提交借用申请',
  REVIEW_BORROW: '审批借用申请',
  PICKUP_EQUIPMENT: '领取交接',
  REQUEST_RETURN: '发起归还',
  CONFIRM_RETURN: '归还验收',
  RESOLVE_EXCEPTION: '异常处理完毕',
}

// 活动流水里的 detail 是链码写入的自由文本，其中有几种是枚举原文（approved=true / NORMAL）。
// 直接显示英文会破坏中文界面的一致性，这里把它们翻成人话。
const DETAIL_FIXUPS = [
  [/^approved=true$/, '评审通过'],
  [/^approved=false$/, '评审驳回'],
]
// 归还验收的结论用另一套词：链码只接受 NORMAL / DAMAGED / MISSING
const RETURN_VERDICT = { NORMAL: '正常', DAMAGED: '损坏', MISSING: '遗失' }

export function detailText(raw) {
  const s = (raw || '').trim()
  if (!s) return ''
  for (const [re, text] of DETAIL_FIXUPS) if (re.test(s)) return text
  if (RETURN_VERDICT[s]) return `验收结论 ${RETURN_VERDICT[s]}`
  if (EQUIPMENT_CONDITION[s]) return `成色 ${EQUIPMENT_CONDITION[s]}`
  return s
}

export function tone(statusMap, key) {
  return statusMap[key]?.tone || 'idle'
}

export function label(statusMap, key) {
  return statusMap[key]?.label || key || '暂无'
}

// 截断哈希便于展示
export function shortHash(h, head = 10, tail = 6) {
  if (!h) return '暂无'
  if (h.length <= head + tail + 1) return h
  return `${h.slice(0, head)}…${h.slice(-tail)}`
}

export function fmtDate(s) {
  if (!s) return '暂无'
  const raw = String(s)
  const date = /^\d{4}-\d{2}-\d{2}$/.test(raw) ? new Date(`${raw}T00:00:00`) : new Date(raw)
  if (Number.isNaN(date.getTime())) return raw
  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  }).format(date)
}

export function toInputDate(value = new Date()) {
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  const local = new Date(date.getTime() - date.getTimezoneOffset() * 60000)
  return local.toISOString().slice(0, 10)
}

export function fmtTime(s) {
  if (!s) return '暂无'
  const d = new Date(s)
  if (Number.isNaN(d.getTime())) return s
  return d.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export function daysBetween(a, b) {
  if (!a || !b) return 0
  const d1 = new Date(a)
  const d2 = new Date(b)
  return Math.max(0, Math.round((d2 - d1) / 86400000))
}

// 链上记的是 Fabric 的序列化身份（base64 过的 X.509 主体），界面不能直接摊给人看。
// 解出来只取 CN 与 O：C=US, O=org1.example.com, CN=User1@org1.example.com
export function whoIs(raw) {
  if (!raw) return '暂无'
  let s = String(raw)
  // 只认「还没解出来」的那种：序列化身份里必有 CN=，base64 里没有
  if (!/CN=/.test(s)) {
    try {
      s = atob(s)
    } catch {
      return String(raw).slice(0, 12)
    }
  }
  const cn = /CN=([^,]+)/.exec(s)
  const org = /O=([^,]+)/.exec(s)
  const msp = /org(\d)\.example\.com/i.exec(org ? org[1] : '')
  if (cn) {
    const name = cn[1].replace(/@.*$/, '')
    return msp ? `${name} · Org${msp[1]}` : name
  }
  return msp ? `Org${msp[1]}` : String(raw).slice(0, 12)
}

// 链码函数名（CamelCase，从真实区块交易解码得到）→ 中文操作名
const FN_LABEL = {
  InitLedger: '初始化账本',
  CreateEquipment: '新增设备',
  UpdateEquipmentStatus: '调整设备状态',
  CreateSpace: '新增空间',
  SubmitProjectApplication: '提交入驻申请',
  ReviewProjectApplication: '评审入驻申请',
  ActivateProject: '分配空间并入驻',
  ExtendProject: '项目续期',
  CloseProject: '项目结项',
  SubmitMilestone: '提交阶段成果',
  RequestEquipmentBorrow: '提交借用申请',
  ReviewEquipmentBorrow: '审批借用申请',
  ConfirmEquipmentPickup: '领取交接',
  RequestEquipmentReturn: '发起归还',
  ConfirmEquipmentReturn: '归还验收',
  ResolveException: '异常处理完毕',
}

export function fnLabel(fn) {
  return FN_LABEL[fn] || fn || '暂无'
}

// 组织标识在界面上统一换成中文角色
const MSP_LABEL = { Org1MSP: '委员会', Org2MSP: '项目团队' }

export function orgName(msp) {
  return MSP_LABEL[msp] || msp || '暂无'
}
