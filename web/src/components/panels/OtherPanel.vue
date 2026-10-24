<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { currentDevice, deviceDisplayName, errorOf, refreshBattery, store } from '../../stores/devices'
import { deviceApi } from '../../stores/devices'
import { simDisplay } from '../../api/phone'
import type { LocationInfo } from '../../api/types'

const device = computed(() => currentDevice())
const name = computed(() => deviceDisplayName(device.value))

const stateText = computed(() => {
  const s = device.value?.state
  return s === 'online' ? '已连接' : s === 'pending-auth' ? '待凭据' : s === 'scanning' ? '连接中…' : '离线'
})

/* SIM 概要 */
const sims = computed(() => {
  const cfg = device.value?.config
  const out: { slot: number; carrier: string; number: string }[] = []
  for (const idx of [1, 2] as const) {
    const remark = cfg?.[`extra_sim${idx}` as 'extra_sim1' | 'extra_sim2'] ?? ''
    const raw = cfg?.sim_info_list?.[String(idx - 1)]
    const disp = remark ? simDisplay(remark, '') : simDisplay(raw?.mCarrierName, raw?.mNumber)
    if (disp.name || disp.number) out.push({ slot: idx, carrier: disp.name, number: disp.number })
  }
  return out
})

const flags = computed(() => {
  const c = device.value?.config
  if (!c) return null
  const all = [
    { label: '短信查询', on: c.enable_api_sms_query },
    { label: '短信发送', on: c.enable_api_sms_send },
    { label: '通话', on: c.enable_api_call_query },
    { label: '联系人', on: c.enable_api_contact_query },
    { label: '加联系人', on: c.enable_api_contact_add },
    { label: '电量', on: c.enable_api_battery_query },
    { label: '定位', on: c.enable_api_location },
    { label: 'WOL', on: c.enable_api_wol },
    { label: '克隆', on: c.enable_api_clone },
  ]
  return { on: all.filter((f) => f.on).length, total: all.length, list: all }
})

/* ── 电量（融入横幅）── */
const batt = computed(() => device.value?.battery)
function numeric(v: string | undefined): number {
  const m = String(v ?? '').match(/[-\d.]+/)
  return m ? parseFloat(m[0]!) : NaN
}
const battPct = computed(() => {
  const lv = numeric(batt.value?.level)
  return isNaN(lv) ? 0 : Math.round(lv)
})
const battStatus = computed(() => {
  const s = batt.value?.status ?? ''
  return { '2': '充电中', '3': '放电中', '4': '未充电', '5': '已充满' }[s] ?? s ?? '—'
})
const battPlugged = computed(() => {
  const p = batt.value?.plugged ?? ''
  if (!p || p === '未知') return ''
  return { '0': '未插电', '1': '交流电源', '2': 'USB', '4': '无线' }[p] ?? p
})
const battVolt = computed(() => {
  const raw = String(batt.value?.voltage ?? '').trim()
  if (!raw) return '—'
  if (/v/i.test(raw)) return raw
  const v = numeric(raw)
  return isNaN(v) ? '—' : `${(v / 1000).toFixed(2)} V`
})
const battTemp = computed(() => {
  const t = String(batt.value?.temperature ?? '')
  return t.includes('℃') || t.includes('°') ? t : `${t} ℃`
})
const battHealth = computed(() => batt.value?.health ?? '—')

async function refreshBatt(): Promise<void> {
  const d = device.value
  if (d) await refreshBattery(d)
}

/* ── 定位（缓存）── */
const locEnabled = computed(() => Boolean(device.value?.config?.enable_api_location))
const loc = ref<LocationInfo | null>(null)
const locLoading = ref(false)
const locError = ref<unknown>(null)

async function readLocation(): Promise<void> {
  const d = device.value
  if (!d || !locEnabled.value || locLoading.value) return
  locLoading.value = true
  locError.value = null
  try {
    loc.value = await deviceApi.locationQuery(d)
  } catch (e) {
    loc.value = null
    locError.value = e
  } finally {
    locLoading.value = false
  }
}

/* ── WOL ── */
const wolEnabled = computed(() => Boolean(device.value?.config?.enable_api_wol))
const wolMac = ref('')
const wolIp = ref('')
const wolPort = ref('9')
const wolBusy = ref(false)
const wolResult = ref<null | { ok: boolean; text: string }>(null)
const wolError = ref<unknown>(null)

async function sendWol(): Promise<void> {
  const d = device.value
  if (!d || wolBusy.value) return
  wolBusy.value = true
  wolResult.value = null
  wolError.value = null
  try {
    await deviceApi.wolSend(d, { mac: wolMac.value, ip: wolIp.value || undefined, port: Number(wolPort.value) || 9 })
    wolResult.value = { ok: true, text: '唤醒指令已发送（接口成功不代表目标电脑已开机）' }
  } catch (e) {
    wolError.value = e
  } finally {
    wolBusy.value = false
  }
}

/* ── 配置克隆 / 恢复 ── */
const cloneEnabled = computed(() => Boolean(device.value?.config?.enable_api_clone))
const cloneBusy = ref(false)
const cloneSnapshot = ref<Record<string, unknown> | null>(null)
const cloneConfirmed = ref(false)
const cloneError = ref<unknown>(null)

async function pullClone(): Promise<void> {
  const d = device.value
  if (!d || cloneBusy.value) return
  cloneBusy.value = true
  cloneError.value = null
  try {
    cloneSnapshot.value = (await deviceApi.clonePull(d)) as Record<string, unknown>
  } catch (e) {
    cloneError.value = e
  } finally {
    cloneBusy.value = false
  }
}

async function pushClone(): Promise<void> {
  const d = device.value
  if (!d || cloneBusy.value) return
  cloneBusy.value = true
  cloneError.value = null
  try {
    await deviceApi.clonePush(d, cloneSnapshot.value ?? {})
    cloneSnapshot.value = null
    cloneConfirmed.value = false
  } catch (e) {
    cloneError.value = e
  } finally {
    cloneBusy.value = false
  }
}

/* 设备切换时重置动态区 */
watch(() => store.currentId, () => {
  loc.value = null
  wolResult.value = null
  cloneSnapshot.value = null
  cloneConfirmed.value = false
})
watch(() => device.value?.state, (s) => { if (s === 'online') void refreshBatt() })
onMounted(() => { void refreshBatt() })
</script>

<template>
  <div class="other">
    <!-- 设备概要 + 电量（融合横幅） -->
    <header class="hero">
      <div class="hero-left">
        <div class="hero-main">
          <span class="state-dot" :class="device?.state"></span>
          <b class="hero-name">{{ name }}</b>
          <span class="badge" :class="device?.state">{{ stateText }}</span>
        </div>
        <div class="hero-meta mono">
          <span>{{ device ? `${device.host}:${device.port}` : '—' }}</span>
          <span class="dot-sep">·</span>
          <span>v{{ device?.config?.version_name || '—' }}</span>
          <template v-for="s in sims" :key="s.slot">
            <span class="dot-sep">·</span>
            <span>SIM{{ s.slot }} {{ s.carrier }}<i v-if="s.number" class="num">{{ s.number }}</i></span>
          </template>
        </div>
        <div v-if="flags" class="hero-flags">
          <span v-for="f in flags.list" :key="f.label" class="flag" :class="{ on: f.on }">{{ f.label }}</span>
          <span class="flag-sum">{{ flags.on }}/{{ flags.total }}</span>
        </div>
      </div>
      <div class="hero-batt" :class="{ low: battPct < 20 }" title="点击刷新">
        <div class="batt-num">{{ batt ? battPct : '—' }}<small v-if="batt">%</small></div>
        <div class="batt-sub">
          {{ battStatus }}<template v-if="battPlugged"> · {{ battPlugged }}</template>
          <button class="link" @click="refreshBatt">刷新</button>
        </div>
      </div>
    </header>

    <!-- 设置行：标签左 · 内容右，行式融合 -->
    <div class="sheet">
      <div class="row">
        <div class="row-label">电池</div>
        <div class="row-body">
          <template v-if="batt">
            <span class="kv"><i>电压</i>{{ battVolt }}</span>
            <span class="kv"><i>温度</i>{{ battTemp }}</span>
            <span class="kv"><i>健康</i>{{ battHealth }}</span>
            <span class="kv-note">读自系统广播缓存，非实时采样</span>
          </template>
          <span v-else class="muted">暂无电量数据</span>
        </div>
      </div>

      <div class="row">
        <div class="row-label">定位</div>
        <div class="row-body">
          <template v-if="!locEnabled">
            <span class="muted">未开启——这台手机未启用 enable_api_location，请在手机端开启后使用</span>
          </template>
          <template v-else>
            <button class="btn" :disabled="locLoading" @click="readLocation">{{ locLoading ? '读取中…' : '读取缓存' }}</button>
            <template v-if="loc">
              <span class="kv"><i>时间</i>{{ loc.time || '未知' }}</span>
              <span class="kv wide"><i>地址</i>{{ loc.address || '无地址信息' }}</span>
              <span class="kv mono"><i>坐标</i>{{ loc.latitude.toFixed(5) }}, {{ loc.longitude.toFixed(5) }}（{{ loc.provider }}）</span>
              <span class="kv-note">缓存时间，不代表当前时刻；不触发新定位</span>
            </template>
            <span v-if="locError" class="err-inline">{{ errorOf(locError).kind }}：{{ errorOf(locError).message }}</span>
          </template>
        </div>
      </div>

      <div class="row">
        <div class="row-label">唤醒</div>
        <div class="row-body">
          <template v-if="!wolEnabled">
            <span class="muted">未开启——手机端未启用 enable_api_wol</span>
          </template>
          <template v-else>
            <input v-model="wolMac" class="in mono" placeholder="目标 MAC，如 1A:2B:3C:4D:5E:6F" />
            <input v-model="wolIp" class="in mono short" placeholder="IP（默认广播）" />
            <input v-model="wolPort" class="in mono tiny" placeholder="9" />
            <button class="btn primary" :disabled="wolBusy || !wolMac" @click="sendWol">{{ wolBusy ? '发送中…' : '发送唤醒' }}</button>
            <span v-if="wolResult?.ok" class="ok-inline">{{ wolResult.text }}</span>
            <span v-if="wolError" class="err-inline">{{ errorOf(wolError).kind }}：{{ errorOf(wolError).message }}</span>
          </template>
        </div>
      </div>

      <div class="row column">
        <div class="row-head">
          <div class="row-label">配置</div>
          <div class="row-body">
            <template v-if="!cloneEnabled">
              <span class="muted">未开启——手机端未启用 enable_api_clone</span>
            </template>
            <template v-else-if="!cloneSnapshot">
              <button class="btn" :disabled="cloneBusy" @click="pullClone">{{ cloneBusy ? '读取中…' : '从手机拉取配置快照' }}</button>
              <span class="kv-note">拉取只读，不改动手机设置</span>
            </template>
          </div>
        </div>
        <template v-if="cloneSnapshot">
          <div class="clone-area">
            <pre class="snapshot">{{ JSON.stringify(cloneSnapshot, null, 2) }}</pre>
            <div class="clone-warn">
              <b>恢复的影响范围</b>
              <span>恢复会用快照<b>替换目标手机的现有设置</b>，并可能清理部分已有数据（转发规则等）。不可自动撤销。</span>
              <label class="confirm"><input v-model="cloneConfirmed" type="checkbox" /> 我已了解恢复会替换设置并清理部分数据</label>
              <div class="clone-btns">
                <button class="btn" :disabled="cloneBusy" @click="cloneSnapshot = null">仅留档，关闭</button>
                <button class="btn danger" :disabled="cloneBusy || !cloneConfirmed" @click="pushClone">{{ cloneBusy ? '恢复中…' : '恢复到其他手机' }}</button>
              </div>
            </div>
          </div>
        </template>
        <span v-if="cloneError" class="err-inline" style="margin: 8px 0 0 96px">{{ errorOf(cloneError).kind }}：{{ errorOf(cloneError).message }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.other {
  padding: 16px 26px 24px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* ── 设备概要横幅（含电量）── */
.hero {
  background: var(--card);
  border: 1px solid var(--line);
  border-radius: 10px;
  padding: 14px 18px;
  display: flex;
  gap: 16px;
  align-items: stretch;
}
.hero-left { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 7px; }
.hero-main { display: flex; align-items: center; gap: 10px; }
.state-dot { width: 9px; height: 9px; border-radius: 50%; background: #d1d5db; flex: none; }
.state-dot.online { background: var(--green); box-shadow: 0 0 0 3px rgba(22, 163, 74, 0.15); }
.state-dot.pending-auth { background: var(--amber); box-shadow: 0 0 0 3px rgba(217, 119, 6, 0.15); }
.hero-name { font-size: 17px; font-weight: 800; }
.badge { font-size: 11px; padding: 2px 10px; border-radius: 12px; background: var(--green-soft); color: var(--green); font-weight: 600; }
.badge.pending-auth { background: #fffbeb; color: var(--amber); }
.badge.offline, .badge.scanning { background: #f2f4f7; color: var(--faint); }
.hero-meta { font-size: 12px; color: var(--dim); display: flex; flex-wrap: wrap; align-items: baseline; gap: 6px; }
.hero-meta .num { font-style: normal; margin-left: 5px; color: var(--ink); }
.dot-sep { color: #d5d9e0; }
.hero-flags { display: flex; flex-wrap: wrap; gap: 4px; align-items: center; padding-top: 5px; border-top: 1px dashed var(--line-soft); }
.flag { font-size: 10px; line-height: 1; padding: 4px 8px; border-radius: 999px; color: #b6bcc7; background: #f6f7f9; }
.flag.on { color: var(--green); background: var(--green-soft); }
.flag-sum { font-size: 10.5px; color: var(--faint); margin-left: auto; }

/* 电量（横幅右翼） */
.hero-batt {
  flex: none;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  justify-content: center;
  padding-left: 20px;
  border-left: 1px solid var(--line-soft);
  min-width: 120px;
  cursor: pointer;
}
.batt-num { font-size: 34px; font-weight: 800; line-height: 1; color: var(--green); }
.hero-batt.low .batt-num { color: var(--red); }
.batt-num small { font-size: 15px; font-weight: 700; margin-left: 1px; }
.batt-sub { font-size: 11.5px; color: var(--dim); margin-top: 6px; display: flex; align-items: center; gap: 8px; }
.link { border: none; background: none; color: var(--blue); font-size: 11.5px; cursor: pointer; padding: 0; }

/* ── 设置行（融合表面）── */
.sheet {
  background: var(--card);
  border: 1px solid var(--line);
  border-radius: 10px;
  overflow: hidden;
}
.row {
  display: flex;
  gap: 0;
  padding: 14px 18px;
  border-bottom: 1px solid var(--line-soft);
  align-items: baseline;
}
.row:last-child { border-bottom: none; }
.row.column { flex-direction: column; gap: 10px; }
.row-head { display: flex; gap: 0; align-items: baseline; }
.row-label {
  flex: none;
  width: 78px;
  font-size: 13px;
  font-weight: 700;
  color: var(--ink);
}
.row-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 8px 14px;
}
.kv { font-size: 12.5px; color: var(--ink); display: inline-flex; gap: 6px; align-items: baseline; }
.kv i { font-style: normal; font-size: 11px; color: var(--faint); }
.kv.wide i { flex: none; }
.kv.mono { font-family: Consolas, monospace; font-size: 12px; }
.kv-note { font-size: 10.5px; color: var(--faint); width: 100%; }
.muted { font-size: 12.5px; color: var(--faint); }

.btn {
  border: 1px solid var(--line);
  background: var(--card);
  border-radius: 8px;
  padding: 8px 14px;
  font-size: 12.5px;
  color: var(--ink);
}
.btn.primary { background: var(--blue); border-color: var(--blue); color: #fff; font-weight: 600; }
.btn.danger { background: var(--red); border-color: var(--red); color: #fff; font-weight: 600; }
.btn:disabled { opacity: 0.55; cursor: not-allowed; }

.in {
  border: 1px solid var(--line);
  border-radius: 7px;
  padding: 7px 10px;
  font-size: 12.5px;
  background: var(--card);
  color: var(--ink);
}
.in.short { width: 130px; }
.in.tiny { width: 52px; }

.ok-inline { font-size: 11.5px; color: var(--green); background: var(--green-soft); border-radius: 6px; padding: 3px 8px; }
.err-inline { font-size: 11.5px; color: var(--red); }

/* 配置展开区 */
.clone-area { display: flex; flex-direction: column; gap: 10px; padding-left: 78px; }
.snapshot {
  background: #0f172a;
  color: #e2e8f0;
  border-radius: 8px;
  padding: 12px 14px;
  font-size: 11.5px;
  line-height: 1.55;
  overflow: auto;
  max-height: 300px;
  margin: 0;
}
.clone-warn {
  background: #fffbeb;
  border: 1px solid #fde68a;
  border-radius: 8px;
  padding: 10px 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 12px;
}
.clone-warn b { color: #92400e; }
.clone-warn span { color: #78350f; line-height: 1.6; }
.confirm { display: flex; gap: 8px; align-items: center; font-size: 12px; color: var(--ink); }
.clone-btns { display: flex; gap: 8px; }

@media (max-width: 768px) {
  .other { padding: 12px 14px 20px; }
  .hero { flex-direction: column; gap: 10px; }
  .hero-batt { border-left: none; padding-left: 0; border-top: 1px solid var(--line-soft); padding-top: 10px; flex-direction: row; align-items: baseline; justify-content: space-between; min-width: 0; }
  .row { flex-direction: column; gap: 8px; padding: 12px 14px; }
  .row-head { flex-direction: column; }
  .clone-area { padding-left: 0; }
}
</style>
