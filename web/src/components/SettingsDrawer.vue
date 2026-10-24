<script setup lang="ts">
import { computed, ref } from 'vue'
import { probe, reloadDevices, selectDevice, store, type DeviceEntry } from '../stores/devices'
import { SAFETY_MODE_LABELS, type SafetyMode } from '../api/types'
import * as real from '../api/real'
import ErrorBanner from './ErrorBanner.vue'
import IconClose from './IconClose.vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const devices = computed(() => store.devices)
const isReal = computed(() => store.mode === 'real')
const editing = ref<string | null>(null)
const error = ref<unknown>(null)
const busy = ref(false)
const manual = ref(false)
const notice = ref<string | null>(null)

const draft = ref<{ host: string; port: number; safetyMode: SafetyMode; secret: string; remark: string }>({
  host: '', port: 5000, safetyMode: 0, secret: '', remark: '',
})

function startEdit(d: DeviceEntry): void {
  notice.value = null
  editing.value = d.id
  manual.value = false
  draft.value = { host: d.host, port: d.port, safetyMode: d.safetyMode, secret: d.secret ?? '', remark: d.remark ?? '' }
}

function startManual(): void {
  notice.value = null
  manual.value = true
  editing.value = 'new'
  draft.value = { host: '', port: 5000, safetyMode: 0, secret: '', remark: '' }
}

/** 保存（real=后端连接库；mock=本地条目）并重连测试 */
async function save(): Promise<void> {
  error.value = null
  busy.value = true
  try {
    if (isReal.value) {
      const input = {
        host: draft.value.host.trim(),
        port: Number(draft.value.port) || 5000,
        safetyMode: draft.value.safetyMode,
        secret: draft.value.secret,
        remark: draft.value.remark,
      }
      if (manual.value) {
        const res = await real.createConnection(input) // 同地址已存在时后端幂等返回现有条目
        if (res.existed) {
          notice.value = `该地址已保存（${res.entry.host}:${res.entry.port}），已为你选中现有连接`
        }
      } else if (editing.value) {
        await real.updateConnection(editing.value, input)
      }
      await reloadDevices()
    } else {
      const d = devices.value.find((x) => x.id === editing.value)
      if (d) {
        d.host = draft.value.host.trim()
        d.port = Number(draft.value.port) || 5000
        d.safetyMode = draft.value.safetyMode
        d.secret = draft.value.secret || undefined
        d.remark = draft.value.remark
        d.state = 'scanning'
        await probe(d)
      }
    }
    editing.value = null
    manual.value = false
    if (!notice.value) notice.value = null
  } catch (e) {
    error.value = e
  } finally {
    busy.value = false
  }
}

/** real：删除连接 */
async function remove(d: DeviceEntry): Promise<void> {
  error.value = null
  if (!isReal.value) return
  busy.value = true
  try {
    await real.deleteConnection(d.id)
    await reloadDevices()
  } catch (e) {
    error.value = e
  } finally {
    busy.value = false
  }
}

async function retest(d: DeviceEntry): Promise<void> {
  d.state = 'scanning'
  await probe(d)
}

function pick(d: DeviceEntry): void {
  selectDevice(d.id)
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="mask" @click.self="emit('close')">
      <aside class="drawer">
        <div class="head">
          <div>
            <div class="title">连接设置</div>
            <div class="sub">
              {{ isReal ? `已保存 ${devices.length} 个连接 · 凭据保存在后端数据目录` : `演示模式（后端未启动）· ${devices.length} 个本地条目` }}
            </div>
          </div>
          <button class="x" aria-label="关闭" @click="emit('close')"><IconClose /></button>
        </div>

        <div v-if="notice" class="notice-banner">{{ notice }}</div>
        <ErrorBanner v-if="error" :error="error" style="margin: 12px 20px 0" />

        <div class="actions-row">
          <button class="btn primary" @click="startManual">手动连接…</button>
          <span class="hint">输入 IP、端口、安全模式与密钥；连接测试成功后保存</span>
        </div>

        <div v-for="d in devices" :key="d.id" class="conn">
          <div class="conn-head">
            <b class="mono">{{ d.host }}:{{ d.port }}</b>
            <span class="mode">{{ SAFETY_MODE_LABELS[d.safetyMode] }}</span>
            <span v-if="d.remark" class="remark">{{ d.remark }}</span>
            <span class="state" :class="d.state">
              {{ d.state === 'online' ? '在线' : d.state === 'pending-auth' ? '待凭据' : d.state === 'scanning' ? '探测中' : '离线' }}
            </span>
          </div>
          <div v-if="d.lastError" class="conn-err">{{ d.lastError.kind }} · {{ d.lastError.message }}</div>
          <template v-if="editing === d.id">
            <div class="edit-grid">
              <label>主机<input v-model="draft.host" class="mono" /></label>
              <label>端口<input v-model.number="draft.port" type="number" class="mono" /></label>
              <label>安全模式
                <select v-model.number="draft.safetyMode">
                  <option :value="0">无安全措施</option>
                  <option :value="1">时间戳签名</option>
                  <option :value="2">RSA 加密</option>
                  <option :value="3">SM4 加密</option>
                </select>
              </label>
              <label>备注<input v-model="draft.remark" placeholder="设备备注（可选）" /></label>
              <label class="wide">密钥<input v-model="draft.secret" type="password" class="mono" :placeholder="draft.safetyMode === 0 ? '无安全模式无需密钥' : '按模式填写密钥'" /></label>
            </div>
            <div class="row-btns">
              <button class="btn primary" :disabled="busy" @click="save">{{ busy ? '测试中…' : '保存并重连' }}</button>
              <button class="btn" @click="editing = null; manual = false">取消</button>
            </div>
          </template>
          <div v-else class="row-btns">
            <button class="btn" @click="pick(d)">选择</button>
            <button class="btn" @click="retest(d)">重连测试</button>
            <button class="btn" @click="startEdit(d)">编辑</button>
            <button v-if="isReal" class="btn danger" :disabled="busy" @click="remove(d)">删除</button>
          </div>
        </div>

        <div class="note-block">
          <b>关于安全模式</b>
          <span>时间戳签名不是完整的防篡改或防重放机制；RSA / SM4 为加密模式，密钥按上游协议填写。连接条目使用控制台内部标识，IP 与设备备注变化不会自动合并设备。</span>
        </div>
      </aside>
    </div>
  </Teleport>
</template>

<style scoped>
.mask {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.32);
  display: flex;
  justify-content: flex-end;
  z-index: 40;
  animation: mask-in 0.16s ease-out;
}
@keyframes mask-in {
  from { opacity: 0; }
  to { opacity: 1; }
}
.drawer {
  width: 460px;
  max-width: 92vw;
  background: var(--bg);
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 16px 0 12px;
  box-shadow: -12px 0 28px rgba(15, 23, 42, 0.08);
  overflow-y: auto;
  animation: drawer-in 0.2s cubic-bezier(0.2, 0.7, 0.3, 1);
}
@keyframes drawer-in {
  from { transform: translateX(28px); opacity: 0.6; }
  to { transform: translateX(0); opacity: 1; }
}
.head { display: flex; align-items: center; padding: 2px 20px 12px; background: var(--card); border-bottom: 1px solid var(--line); }
.title { font-size: 15px; font-weight: 700; }
.sub { font-size: 11px; color: var(--faint); margin-top: 2px; }
.x { margin-left: auto; width: 26px; height: 26px; border-radius: 6px; border: 1px solid var(--line); background: var(--card); color: var(--faint); display: flex; align-items: center; justify-content: center; }
.actions-row { display: flex; align-items: center; gap: 10px; padding: 14px 20px 4px; }
.hint { font-size: 11px; color: var(--faint); }
.conn { background: var(--card); border-bottom: 1px solid var(--line-soft); padding: 12px 20px 14px; }
.conn-head { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.conn-head b { font-size: 13.5px; }
.mode { font-size: 11px; padding: 2px 9px; border-radius: 4px; background: var(--blue-soft); color: var(--blue); }
.remark { font-size: 11px; color: var(--dim); }
.state { margin-left: auto; font-size: 11.5px; font-weight: 600; }
.state.online { color: var(--green); }
.state.pending-auth { color: var(--amber); }
.state.offline, .state.scanning { color: var(--faint); }
.conn-err { margin-top: 6px; font-size: 11.5px; color: var(--red); }
.edit-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 10px; margin-top: 10px; }
.edit-grid label { display: flex; flex-direction: column; gap: 4px; font-size: 11px; color: var(--dim); }
.edit-grid label.wide { grid-column: 1 / -1; }
.edit-grid input, .edit-grid select {
  border: 1px solid var(--line);
  border-radius: 7px;
  padding: 7px 10px;
  font-size: 12.5px;
  background: var(--card);
}
.row-btns { display: flex; gap: 8px; margin-top: 10px; flex-wrap: wrap; }
.btn { border: 1px solid var(--line); background: var(--card); border-radius: 7px; padding: 7px 14px; font-size: 12.5px; }
.btn.primary { background: var(--blue); border-color: var(--blue); color: #fff; font-weight: 600; }
.btn.danger { color: var(--red); border-color: #fecaca; }
.btn:disabled { opacity: 0.55; cursor: not-allowed; }
.notice-banner {
  margin: 12px 20px 0;
  background: var(--blue-soft);
  border: 1px solid #bfdbfe;
  color: var(--blue);
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 12.5px;
}
.note-block {
  margin: 14px 20px;
  background: var(--card);
  border: 1px dashed var(--line);
  border-radius: 10px;
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.note-block b { font-size: 12.5px; }
.note-block span { font-size: 11.5px; color: var(--dim); line-height: 1.65; }
</style>
