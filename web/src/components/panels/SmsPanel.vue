<script setup lang="ts">
import { computed, onActivated, onDeactivated, onMounted, onUnmounted, ref, watch } from 'vue'
import { deviceApi, currentDevice, store } from '../../stores/devices'
import type { SmsInfo } from '../../api/types'
import { simDisplay } from '../../api/phone'
import ErrorBanner from '../ErrorBanner.vue'

const smsType = ref<1 | 2>(1)
const keyword = ref('')
const simFilter = ref<'all' | number>('all') // 上游无 SIM 筛选参数：本地在拉取范围内过滤
const items = ref<SmsInfo[]>([])
const loading = ref(false)
const error = ref<unknown>(null)

const PAGE_SIZE = 20
/** SIM 筛选时的拉取窗口（上游无 sim 参数，一次拉 100 条内过滤后本地分页） */
const FILTER_WINDOW = 100
const pageNum = ref(1)
const hasMore = ref(false)
/** SIM 筛选激活时在窗口内本地分页 */
const localPageNum = ref(1)

/* 本机卡号（config 的 SIM 备注拆出，如 13800138000）；sim_id 0/1 → 号码 */
const ownNumbers = computed(() => {
  const cfg = currentDevice()?.config
  const out: Record<number, string> = {}
  for (const idx of [1, 2] as const) {
    const remark = cfg?.[`extra_sim${idx}` as 'extra_sim1' | 'extra_sim2'] ?? ''
    const list = cfg?.sim_info_list ?? {}
    const raw = list[String(idx - 1)]
    const disp = remark ? simDisplay(remark, '') : simDisplay(raw?.mCarrierName, raw?.mNumber)
    if (disp.number) out[idx - 1] = disp.number
  }
  return out
})

const simOptions = computed(() => {
  const opts: { key: 'all' | number; label: string }[] = [{ key: 'all', label: '全部 SIM' }]
  for (const [k, num] of Object.entries(ownNumbers.value)) {
    opts.push({ key: Number(k), label: `SIM${Number(k) + 1} · ${num}` })
  }
  return opts
})

/** 请求序号：快速切换/翻页时丢弃过期响应 */
let querySeq = 0

async function querySms(page = 1): Promise<void> {
  const d = currentDevice()
  if (!d || d.state !== 'online') { items.value = []; hasMore.value = false; return }
  const seq = ++querySeq
  loading.value = true
  error.value = null
  const simOn = simFilter.value !== 'all'
  try {
    const list = await deviceApi.smsQuery(d, {
      type: smsType.value,
      page_num: simOn ? 1 : page,
      page_size: simOn ? FILTER_WINDOW : PAGE_SIZE,
      keyword: keyword.value,
    })
    if (seq !== querySeq) return
    items.value = list
    if (simOn) {
      localPageNum.value = page
      hasMore.value = filtered.value.length >= page * PAGE_SIZE
    } else {
      pageNum.value = page
      hasMore.value = list.length >= PAGE_SIZE
    }
    trackNewest()
    if (page === 1) hasNew.value = false
  } catch (e) {
    if (seq !== querySeq) return
    items.value = []
    hasMore.value = false
    error.value = e
  } finally {
    if (seq === querySeq) loading.value = false
  }
}

/** SIM 筛选后的本地分页视图 */
const filtered = computed(() =>
  simFilter.value === 'all' ? items.value : items.value.filter((s) => (s.sim_id ?? -1) === simFilter.value),
)
const rows = computed(() => {
  if (simFilter.value === 'all') return items.value
  const start = (localPageNum.value - 1) * PAGE_SIZE
  return filtered.value.slice(start, start + PAGE_SIZE)
})

function switchType(t: 1 | 2): void {
  if (smsType.value === t) return
  smsType.value = t
  void querySms(1)
}

function switchSim(v: 'all' | number): void {
  simFilter.value = v
  void querySms(1)
}

/* ── 展示 ── */

/** 对方号码单行：号码（姓名或"未知号码"） */
function party(s: SmsInfo): string {
  const name = s.name?.trim() || '未知号码'
  return `${s.number}（${name}）`
}

/** 发送方：接收短信=对方号码；发送短信=本机卡号 */
function senderOf(s: SmsInfo): string {
  return s.type === 1 ? party(s) : ownNumbers.value[s.sim_id ?? 0] ?? `SIM${(s.sim_id ?? 0) + 1}`
}
/** 接收方：接收短信=本机卡号；发送短信=对方号码 */
function receiverOf(s: SmsInfo): string {
  return s.type === 1 ? ownNumbers.value[s.sim_id ?? 0] ?? `SIM${(s.sim_id ?? 0) + 1}` : party(s)
}

function fmtTime(ms: number): string {
  const dt = new Date(ms)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${p(dt.getMonth() + 1)}-${p(dt.getDate())} ${p(dt.getHours())}:${p(dt.getMinutes())}`
}

const summary = computed(() => {
  if (!rows.value.length) return '本页无记录'
  const where = simFilter.value === 'all' ? '' : `（最近 ${filtered.value.length} 条内）`
  const page = simFilter.value === 'all' ? pageNum.value : localPageNum.value
  return `第 ${page} 页 · 本页 ${rows.value.length} 条${where}`
})
const currentPage = computed(() => (simFilter.value === 'all' ? pageNum.value : localPageNum.value))
function gotoPage(p: number): void {
  void querySms(p)
}

watch(() => store.currentId, () => { void querySms(1) })
watch(() => currentDevice()?.state, (s) => { if (s === 'online') void querySms(1) })

/* ── 新短信自动检测（上游无推送；15s 轮询最新 1 条比对） ── */
const AUTO_INTERVAL = 15_000
const autoRefresh = ref(true)
const hasNew = ref(false)
let newestDate = 0
let autoTimer: ReturnType<typeof setInterval> | null = null

function trackNewest(): void {
  newestDate = items.value.reduce((m, s) => Math.max(m, s.date), 0)
}

async function pollNewest(): Promise<void> {
  if (!autoRefresh.value || loading.value || hasNew.value) return
  if (typeof document !== 'undefined' && document.visibilityState !== 'visible') return
  const d = currentDevice()
  if (!d || d.state !== 'online' || smsType.value !== 1 || simFilter.value !== 'all') return
  try {
    const top = await deviceApi.smsQuery(d, { type: 1, page_num: 1, page_size: 1, keyword: keyword.value })
    if (top.length && top[0]!.date > newestDate) {
      if (currentPage.value === 1) await querySms(1)
      else hasNew.value = true
    }
  } catch { /* 手机瞬时不可达：静默 */ }
}

function startAuto(): void { if (!autoTimer) autoTimer = setInterval(() => void pollNewest(), AUTO_INTERVAL) }
function stopAuto(): void { if (autoTimer) { clearInterval(autoTimer); autoTimer = null } }
function viewNew(): void { hasNew.value = false; void querySms(1) }

onActivated(startAuto)
onDeactivated(stopAuto)
onMounted(async () => {
  startAuto()
  for (let i = 0; i < 30; i++) {
    if (currentDevice()?.state === 'online') break
    await new Promise((r) => setTimeout(r, 100))
  }
  await querySms()
})
onUnmounted(stopAuto)

defineExpose({ querySms })
</script>

<template>
  <div>
    <div class="tools">
      <div class="seg">
        <span :class="{ on: smsType === 1 }" @click="switchType(1)">接收</span>
        <span :class="{ on: smsType === 2 }" @click="switchType(2)">发送</span>
      </div>
      <input v-model="keyword" class="search" placeholder="按号码或内容筛选（匹配发送方 / 接收方号码）" @keydown.enter="querySms(1)" />
      <select class="sim-select" :value="simFilter" @change="switchSim(($event.target as HTMLSelectElement).value === 'all' ? 'all' : Number(($event.target as HTMLSelectElement).value))">
        <option v-for="o in simOptions" :key="o.key" :value="o.key">{{ o.label }}</option>
      </select>
      <button class="btn primary" @click="querySms(1)">查询</button>
      <button
        class="auto-toggle"
        :class="{ on: autoRefresh }"
        :title="autoRefresh ? '每 15 秒自动检测新短信（接收模式）' : '自动检测已关闭'"
        @click="autoRefresh = !autoRefresh"
      >{{ autoRefresh ? '自动刷新中' : '自动刷新' }}</button>
    </div>
    <div v-if="hasNew" class="new-banner" @click="viewNew">检测到新短信 · 点击查看</div>
    <ErrorBanner v-if="error" :error="error" class="err" />
    <div v-if="!error && loading" class="hint">查询中…</div>
    <div v-else-if="!error && !loading && rows.length === 0" class="hint empty">
      未查询到短信——空列表可能与手机端权限或查询异常有关，不一定是没有记录
    </div>
    <div class="table-card">
      <table>
        <thead>
          <tr><th>时间</th><th>发送方</th><th>接收方</th><th>内容</th><th>SIM</th></tr>
        </thead>
        <tbody>
          <tr v-for="s in rows" :key="s.date + s.number">
            <td class="t mono">{{ fmtTime(s.date) }}</td>
            <td class="party mono">{{ senderOf(s) }}</td>
            <td class="party mono">{{ receiverOf(s) }}</td>
            <td class="c">{{ s.content }}</td>
            <td><span class="simtag">SIM{{ (s.sim_id ?? 0) + 1 }}</span></td>
          </tr>
        </tbody>
      </table>
      <div class="pager">
        <span>{{ summary }}<template v-if="hasMore"> · 可能还有更多</template></span>
        <div class="pg">
          <i :class="{ dim: currentPage <= 1 }" @click="currentPage > 1 && gotoPage(currentPage - 1)">‹</i>
          <i class="on">{{ currentPage }}</i>
          <i :class="{ dim: !hasMore }" @click="hasMore && gotoPage(currentPage + 1)">›</i>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tools {
  display: flex;
  gap: 10px;
  padding: 16px 26px 0;
  align-items: center;
}
.seg {
  display: flex;
  border: 1px solid var(--line);
  border-radius: 8px;
  overflow: hidden;
  background: var(--card);
  flex: none;
}
.seg span {
  padding: 7px 16px;
  font-size: 13px;
  color: var(--dim);
  cursor: pointer;
}
.seg span.on {
  color: var(--blue);
  background: var(--blue-soft);
  font-weight: 600;
}
.search {
  flex: 1;
  max-width: 360px;
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: 8px 12px;
  font-size: 13px;
  background: var(--card);
  color: var(--ink);
}
.sim-select {
  flex: none;
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: 7px 10px;
  font-size: 12.5px;
  background: var(--card);
  color: var(--dim);
}
.btn.primary {
  background: var(--blue);
  border: 1px solid var(--blue);
  border-radius: 8px;
  padding: 8px 16px;
  font-size: 13px;
  color: #fff;
  font-weight: 600;
  flex: none;
}
.auto-toggle {
  border: 1px solid var(--line);
  background: var(--card);
  border-radius: 8px;
  padding: 7px 12px;
  font-size: 11.5px;
  color: var(--dim);
  flex: none;
}
.auto-toggle.on {
  color: var(--green);
  border-color: #bbf7d0;
  background: var(--green-soft);
}
.new-banner {
  margin: 10px 26px 0;
  background: var(--green-soft);
  border: 1px solid #bbf7d0;
  color: var(--green);
  border-radius: 8px;
  padding: 8px 14px;
  font-size: 12.5px;
  cursor: pointer;
  animation: pulse-in 0.25s ease-out;
}
@keyframes pulse-in {
  from { transform: translateY(-4px); opacity: 0; }
  to { transform: translateY(0); opacity: 1; }
}
.err { margin: 12px 26px 0; }
.hint {
  padding: 10px 28px 0;
  font-size: 12.5px;
  color: var(--faint);
}
.hint.empty { color: var(--dim); }
.table-card {
  margin: 16px 26px;
  background: var(--card);
  border: 1px solid var(--line);
  border-radius: 10px;
  overflow: hidden;
}
table { width: 100%; border-collapse: collapse; }
th {
  text-align: left;
  font-size: 12px;
  color: var(--dim);
  font-weight: 600;
  padding: 12px 18px;
  background: #fafbfc;
  border-bottom: 1px solid var(--line);
  white-space: nowrap;
}
td {
  padding: 13px 18px;
  font-size: 13px;
  border-bottom: 1px solid var(--line-soft);
  vertical-align: top;
}
tr:last-child td { border-bottom: none; }
td .t { color: var(--dim); font-size: 12px; white-space: nowrap; }
td .party { font-size: 12.5px; white-space: nowrap; }
td .c { color: #374151; }
.simtag {
  font-size: 10.5px;
  padding: 2px 8px;
  border-radius: 4px;
  background: var(--blue-soft);
  color: var(--blue);
  white-space: nowrap;
}
.pager {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 18px;
  font-size: 11px;
  color: var(--dim);
}
.pager .pg { margin-left: auto; display: flex; gap: 6px; }
.pager .pg i {
  width: 21px;
  height: 21px;
  border: 1px solid var(--line);
  border-radius: 5px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-style: normal;
  font-size: 10.5px;
  background: var(--card);
}
.pager .pg i.on {
  background: var(--blue);
  color: #fff;
  border-color: var(--blue);
  min-width: 28px;
}
.pager .pg i.dim { opacity: 0.4; cursor: default; }
.pager .pg i:not(.dim):not(.on) { cursor: pointer; }
.pager .pg i:not(.dim):not(.on):hover { border-color: var(--blue); color: var(--blue); }
</style>
