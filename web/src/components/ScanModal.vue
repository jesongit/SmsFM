<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { probe, reloadDevices, selectDevice, store, type DeviceEntry } from '../stores/devices'
import * as real from '../api/real'
import type { ScanProgress } from '../api/real'
import ErrorBanner from './ErrorBanner.vue'
import IconClose from './IconClose.vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const isReal = computed(() => store.mode === 'real')
const devices = computed(() => store.devices)

/* real：真实扫描（后端执行） */
const cidr = ref('')
const port = ref(5000)
const progress = ref<ScanProgress | null>(null)
const error = ref<unknown>(null)
const busy = ref(false)
let timer: ReturnType<typeof setInterval> | null = null

const found = computed(() => progress.value?.found ?? [])
const percent = computed(() => {
  const p = progress.value
  if (!p || p.total === 0) return 0
  if (p.phase === 'done') return 100
  return Math.min(100, Math.round((p.scanned / p.total) * 100))
})

async function initDefaults(): Promise<void> {
  if (!isReal.value || cidr.value) return
  try {
    const { cidrs } = await real.networks()
    if (cidrs.length) cidr.value = cidrs[0]!
  } catch { /* 保持手填 */ }
}

async function start(): Promise<void> {
  error.value = null
  busy.value = true
  try {
    progress.value = await real.startScan(cidr.value.trim(), Number(port.value) || 5000)
    startPolling()
  } catch (e) {
    error.value = e
  } finally {
    busy.value = false
  }
}

function startPolling(): void {
  stopPolling()
  timer = setInterval(async () => {
    try {
      const p = await real.scanStatus()
      progress.value = p
      if (p.phase === 'done' || p.phase === 'cancelled' || p.phase === 'error') stopPolling()
    } catch {
      stopPolling()
    }
  }, 600)
}

function stopPolling(): void {
  if (timer) { clearInterval(timer); timer = null }
}

async function cancel(): Promise<void> {
  try { await real.cancelScan() } catch { /* 忽略 */ }
}

/** 已识别设备保存为连接（同地址已保存则复用现有条目，不重复添加） */
async function saveFound(f: { host: string; port: number }): Promise<void> {
  error.value = null
  busy.value = true
  try {
    await real.createConnection({ host: f.host, port: f.port, safetyMode: 0, secret: '', remark: '' })
    await reloadDevices()
    const entry = store.devices.find((d) => d.host === f.host && d.port === f.port)
    if (entry) {
      selectDevice(entry.id)
      void probe(entry as DeviceEntry)
      emit('close')
    }
  } catch (e) {
    error.value = e
  } finally {
    busy.value = false
  }
}

/** 该地址是否已保存为连接 */
function isSaved(f: { host: string; port: number }): boolean {
  return devices.value.some((d) => d.host === f.host && d.port === f.port)
}

onBeforeUnmount(stopPolling)

/* mock：模拟扫描动画（演示模式） */
const mockPhase = ref<'idle' | 'fast' | 'recheck' | 'done' | 'cancelled'>('idle')
const mockProgress = ref(0)
let mockTimer: ReturnType<typeof setInterval> | null = null

function startMock(): void {
  stopMock()
  mockPhase.value = 'fast'
  mockProgress.value = 0
  mockTimer = setInterval(() => {
    mockProgress.value += 7
    if (mockProgress.value >= 70 && mockPhase.value === 'fast') mockPhase.value = 'recheck'
    if (mockProgress.value >= 100) {
      stopMock()
      mockPhase.value = 'done'
    }
  }, 90)
}
function stopMock(): void {
  if (mockTimer) { clearInterval(mockTimer); mockTimer = null }
}
onBeforeUnmount(stopMock)

const stateLabel: Record<string, string> = {
  identified: '配置识别成功',
  'need-credentials': '待配置凭据',
  unverified: '未验证候选',
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="mask" @click.self="emit('close')">
      <div class="modal">
        <div class="head">
          <div>
            <div class="title">扫描局域网设备</div>
            <div class="sub">{{ isReal ? '后端在可达网段内并发快扫，发现即识别；不把 Ping 作为前提' : '演示模式：后端未启动，以下为流程模拟' }}</div>
          </div>
          <button class="x" aria-label="关闭" @click="emit('close')"><IconClose /></button>
        </div>

        <!-- real：真实扫描 -->
        <template v-if="isReal">
          <div class="form">
            <label>扫描网段<input v-model="cidr" class="mono" placeholder="如 192.168.1.0/24" @focus="initDefaults" /></label>
            <label>端口<input v-model.number="port" type="number" class="mono" /></label>
            <button class="btn primary" :disabled="busy || !cidr" @click="start">
              {{ busy ? '启动中…' : progress && (progress.phase === 'fast' || progress.phase === 'recheck') ? '重新扫描' : '开始扫描' }}
            </button>
          </div>

          <ErrorBanner v-if="error" :error="error" style="margin: 12px 22px 0" />

          <div v-if="progress" class="progress-block">
            <div class="bar"><i :style="{ transform: 'scaleX(' + percent / 100 + ')' }"></i></div>
            <div class="meta">
              <span v-if="progress.phase === 'fast'">快扫中 · 128 并发 · 350ms 连接超时 · {{ progress.scanned }}/{{ progress.total }}</span>
              <span v-else-if="progress.phase === 'recheck'">快扫完成 · 复查超时地址（64 并发 · 1000ms）</span>
              <span v-else-if="progress.phase === 'done'">扫描完成 · 用时 {{ Math.round(progress.elapsedMs / 100) / 10 }}s</span>
              <span v-else-if="progress.phase === 'cancelled'">已取消（保留已发现结果）</span>
              <span v-else-if="progress.phase === 'error'">扫描失败：{{ progress.error }}</span>
              <span>{{ percent }}%</span>
            </div>
            <button
              v-if="progress.phase === 'fast' || progress.phase === 'recheck'"
              class="btn cancel"
              @click="cancel"
            >取消扫描</button>
          </div>

          <div class="results">
            <div class="sec-title">发现结果（{{ found.length }}）</div>
            <div v-for="f in found" :key="f.host + ':' + f.port" class="res-row">
              <span class="dot" :class="{ amber: f.state !== 'identified' }"></span>
              <span class="mono host">{{ f.host }}:{{ f.port }}</span>
              <span class="tag" :class="{ amber: f.state !== 'identified' }">{{ stateLabel[f.state] || f.state }}</span>
              <span v-if="f.config" class="cfg">{{ f.config.extra_device_mark || f.config.version_name }}</span>
              <span v-if="f.error" class="err-line">{{ f.error.slice(0, 40) }}</span>
              <button v-if="f.state === 'identified' && isSaved(f)" class="mini" disabled>已保存</button>
              <button v-else-if="f.state === 'identified'" class="mini" :disabled="busy" @click="saveFound(f)">保存并选择</button>
              <span v-else class="mini-hint">在连接设置中补凭据</span>
            </div>
            <div v-if="progress && progress.phase === 'done' && !found.length" class="empty">未发现设备——可尝试手动连接或调整网段</div>
          </div>

          <p class="foot-note">端口开放仅表示候选服务；无凭据请求返回明确签名错误时标记为待配置凭据。扫描范围以后端可达的网段为准。</p>
        </template>

        <!-- mock：模拟扫描 -->
        <template v-else>
          <div class="form">
            <label>扫描网段<input v-model="cidr" class="mono" placeholder="192.168.1.0/24" /></label>
            <label>端口<input v-model.number="port" type="number" class="mono" /></label>
            <button class="btn primary" @click="startMock">{{ mockPhase === 'idle' || mockPhase === 'done' || mockPhase === 'cancelled' ? '开始扫描' : '重新扫描' }}</button>
          </div>
          <div v-if="mockPhase !== 'idle'" class="progress-block">
            <div class="bar"><i :style="{ transform: 'scaleX(' + Math.min(mockProgress, 100) / 100 + ')' }"></i></div>
            <div class="meta">
              <span v-if="mockPhase === 'fast'">快扫中 · 128 并发 · 350ms</span>
              <span v-else-if="mockPhase === 'recheck'">复查超时地址（64 并发 · 1000ms）</span>
              <span v-else-if="mockPhase === 'done'">扫描完成</span>
              <span v-else>已取消</span>
              <span>{{ Math.min(mockProgress, 100) }}%</span>
            </div>
            <button v-if="mockPhase === 'fast' || mockPhase === 'recheck'" class="btn cancel" @click="stopMock(); mockPhase = 'cancelled'">取消扫描</button>
          </div>
          <div class="results">
            <div class="sec-title">发现结果</div>
            <div v-for="d in devices.filter(x => x.state !== 'offline')" :key="d.id" class="res-row">
              <span class="dot"></span>
              <span class="mono host">{{ d.host }}:{{ d.port }}</span>
              <span class="tag">演示识别</span>
            </div>
          </div>
          <p class="foot-note">演示数据为合成；启动 Go 后端（server/）后自动切换为真实扫描。</p>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.32);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 40;
  animation: mask-in 0.16s ease-out;
}
@keyframes mask-in {
  from { opacity: 0; }
  to { opacity: 1; }
}
.modal {
  width: 640px;
  max-width: 94vw;
  max-height: 86vh;
  overflow-y: auto;
  background: var(--bg);
  border-radius: 14px;
  padding: 16px 0 12px;
  box-shadow: 0 24px 60px rgba(15, 23, 42, 0.2);
  animation: modal-in 0.2s cubic-bezier(0.2, 0.7, 0.3, 1);
}
@keyframes modal-in {
  from { transform: translateY(14px) scale(0.985); opacity: 0.6; }
  to { transform: translateY(0) scale(1); opacity: 1; }
}
.head { display: flex; align-items: center; padding: 2px 22px 12px; }
.title { font-size: 16px; font-weight: 700; }
.sub { font-size: 11.5px; color: var(--faint); margin-top: 2px; }
.x { margin-left: auto; width: 26px; height: 26px; border-radius: 6px; border: 1px solid var(--line); background: var(--card); color: var(--faint); display: flex; align-items: center; justify-content: center; }
.form { display: flex; gap: 12px; align-items: end; padding: 6px 22px 14px; border-bottom: 1px solid var(--line); }
.form label { display: flex; flex-direction: column; gap: 4px; font-size: 11px; color: var(--dim); }
.form input { border: 1px solid var(--line); border-radius: 7px; padding: 8px 12px; font-size: 13px; width: 220px; background: var(--card); }
.form input:focus-visible { outline: 2px solid var(--blue); outline-offset: 1px; }
.btn { border: 1px solid var(--line); background: var(--card); border-radius: 8px; padding: 8px 16px; font-size: 13px; }
.btn.primary { background: var(--blue); border-color: var(--blue); color: #fff; font-weight: 600; }
.btn.cancel { padding: 6px 12px; font-size: 12px; }
.btn:disabled { opacity: 0.55; cursor: not-allowed; }
.progress-block { padding: 14px 22px 4px; }
.bar { height: 6px; background: var(--line-soft); border-radius: 3px; overflow: hidden; }
.bar i { display: block; height: 100%; width: 100%; background: var(--blue); transform-origin: left; transition: transform 0.4s ease; }
.meta { display: flex; justify-content: space-between; gap: 10px; font-size: 11.5px; color: var(--dim); margin-top: 8px; }
.results { padding: 6px 22px; min-height: 120px; }
.sec-title { font-size: 11px; letter-spacing: 2px; color: var(--faint); padding: 8px 0 4px; }
.res-row { display: flex; align-items: center; gap: 10px; padding: 8px 0; border-bottom: 1px solid var(--line-soft); font-size: 13px; flex-wrap: wrap; }
.dot { width: 8px; height: 8px; border-radius: 50%; background: var(--green); flex: none; }
.dot.amber { background: var(--amber); }
.host { font-weight: 600; }
.tag { font-size: 11px; padding: 2px 9px; border-radius: 4px; background: var(--green-soft); color: var(--green); }
.tag.amber { background: #fffbeb; color: var(--amber); }
.cfg { font-size: 11.5px; color: var(--dim); }
.err-line { font-size: 11px; color: var(--faint); max-width: 200px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.mini { margin-left: auto; border: 1px solid var(--line); background: var(--card); border-radius: 6px; padding: 5px 12px; font-size: 12px; }
.mini:disabled { opacity: 0.55; }
.mini-hint { margin-left: auto; font-size: 10.5px; color: var(--faint); }
.empty { color: var(--faint); font-size: 12.5px; padding: 12px 0; }
.foot-note { padding: 8px 22px 4px; font-size: 10.5px; color: var(--faint); line-height: 1.6; }
</style>
