<script setup lang="ts">
import { computed, ref } from 'vue'
import { deviceApi, currentDevice, deviceDisplayName } from '../stores/devices'
import ErrorBanner from './ErrorBanner.vue'
import { simDisplay } from '../api/phone'
import IconClose from './IconClose.vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const phone = ref('')
const content = ref('')
const simSlot = ref<1 | 2>(1)
const busy = ref(false)
const error = ref<unknown>(null)
const result = ref<null | { kind: 'sent' | 'unknown'; text: string }>(null)

const device = computed(() => currentDevice())
const sims = computed(() => {
  const cfg = device.value?.config
  const list = cfg?.sim_info_list ?? {}
  // 卡位来源：sim_info_list；该 ROM 读不出时回退用手机端自填的 SIM 备注（extra_sim1/2，通常含号码）
  const keys = Object.keys(list).length ? Object.keys(list) : (cfg?.extra_sim1 ? ['0', ...(cfg?.extra_sim2 ? ['1'] : [])] : [])
  return keys.map((k) => {
    const idx = Number(k) + 1
    const remark = cfg?.[`extra_sim${idx}` as 'extra_sim1' | 'extra_sim2'] ?? ''
    const raw = list[k]
    const disp = remark ? simDisplay(remark, '') : simDisplay(raw?.mCarrierName, raw?.mNumber)
    return {
      slot: idx as 1 | 2,
      label: `SIM${idx} · ${disp.name}${disp.number ? ' · ' + disp.number : ''}`,
    }
  })
})
const canConfirmSim = computed(() => sims.value.length > 0)

/* 短信分条：70 字/条（中文字符按上游计费口径，长短信 67 字/条起拆） */
const segments = computed(() => {
  const len = content.value.length
  if (!len) return 1
  return len <= 70 ? 1 : Math.ceil(len / 67)
})

async function send(): Promise<void> {
  const d = device.value
  if (!d || !canConfirmSim.value) return
  busy.value = true
  error.value = null
  result.value = null
  try {
    await deviceApi.smsSend(d, {
      sim_slot: simSlot.value,
      phone_numbers: phone.value,
      msg_content: content.value,
    })
    result.value = { kind: 'sent', text: '已提交手机发送。发送成功不代表对方已经收到。' }
    content.value = ''
  } catch (e) {
    // 超时语义：显示结果未知，不自动重发（开发计划 §6）
    const msg = e instanceof Error ? e.message : String(e)
    if (/timeout|超时/i.test(msg)) {
      result.value = { kind: 'unknown', text: '请求超时，发送结果未知。为避免重复发送，不会自动重发；请稍后在「查短信-发送」中核对。' }
    } else {
      error.value = e
    }
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <div v-if="open" class="mask" @click.self="emit('close')">
      <aside class="drawer">
        <div class="head">
          <div>
            <div class="title">写短信</div>
            <div class="sub">{{ deviceDisplayName(device) }} · {{ device?.host }}:{{ device?.port }}</div>
          </div>
          <button class="x" aria-label="关闭" @click="emit('close')"><IconClose /></button>
        </div>

        <div v-if="!canConfirmSim" class="nosim">
          无法确认 SIM 卡槽（设备未上报 sim_info_list），已阻止提交。请先在手机端完成 SIM 配置。
        </div>
        <template v-else>
          <div class="field">
            <span class="lb">当前设备</span>
            <span class="val">{{ deviceDisplayName(device) }}</span>
          </div>
          <div class="field">
            <span class="lb">选择 SIM</span>
            <div class="seg">
              <span
                v-for="s in sims"
                :key="s.slot"
                :class="{ on: simSlot === s.slot }"
                @click="simSlot = s.slot"
              >{{ s.label }}</span>
            </div>
          </div>
          <label class="field col">
            <span class="lb">接收人号码</span>
            <input v-model="phone" required placeholder="手机号码，多个用逗号分隔" class="mono" />
          </label>
          <label class="field col">
            <span class="lb">短信内容</span>
            <textarea v-model="content" rows="4" placeholder="短信内容…"></textarea>
          </label>
          <div class="meta">
            <span>{{ content.length }} 字 · 第 1 条 / 共 {{ segments }} 条（70 字/条）</span>
            <span>无法确认 SIM 时阻止提交</span>
          </div>
          <button class="send" :disabled="busy || !phone || !content" @click="send">
            {{ busy ? '发送中…' : '发送' }}
          </button>
        </template>

        <div v-if="result?.kind === 'sent'" class="ok-banner">{{ result.text }}</div>
        <div v-if="result?.kind === 'unknown'" class="unknown-banner">{{ result.text }}</div>
        <ErrorBanner v-if="error" :error="error" style="margin: 12px 20px 0" />
        <p class="foot-note">发送成功 ≠ 对方已收到 · 超时显示「结果未知」，不自动重发</p>
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
  width: 430px;
  max-width: 92vw;
  background: var(--card);
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
.head {
  display: flex;
  align-items: center;
  padding: 2px 20px 12px;
  border-bottom: 1px solid var(--line);
}
.title { font-size: 15px; font-weight: 700; }
.sub { font-size: 11px; color: var(--faint); margin-top: 2px; }
.x {
  margin-left: auto;
  width: 26px; height: 26px;
  border-radius: 6px;
  border: 1px solid var(--line);
  background: var(--card);
  color: var(--faint);
}
.nosim {
  margin: 14px 20px 0;
  background: #fffbeb;
  border: 1px solid #fde68a;
  color: #92400e;
  border-radius: 8px;
  padding: 12px 14px;
  font-size: 12.5px;
  line-height: 1.6;
}
.field {
  display: flex;
  gap: 10px;
  align-items: center;
  padding: 12px 20px 0;
  font-size: 12.5px;
}
.field.col { flex-direction: column; align-items: stretch; }
.lb { color: var(--faint); font-size: 11px; width: 60px; flex: none; }
.field.col .lb { width: auto; }
.val { color: var(--ink); font-weight: 600; }
.seg { display: flex; flex-wrap: wrap; gap: 6px; }
.seg span {
  padding: 6px 12px;
  border: 1px solid var(--line);
  border-radius: 7px;
  font-size: 12px;
  color: var(--dim);
  cursor: pointer;
}
.seg span.on { background: var(--blue-soft); color: var(--blue); border-color: var(--blue); font-weight: 600; }
input, textarea {
  border: 1px solid var(--line);
  border-radius: 8px;
  padding: 9px 12px;
  font-size: 13px;
  font-family: inherit;
  resize: vertical;
}
input:focus, textarea:focus { outline: 2px solid rgba(37, 99, 235, 0.25); border-color: var(--blue); }
.meta {
  display: flex;
  justify-content: space-between;
  padding: 10px 20px 0;
  font-size: 10.5px;
  color: var(--faint);
}
.send {
  margin: 14px 20px 0;
  background: var(--blue);
  border: none;
  color: #fff;
  border-radius: 8px;
  padding: 10px;
  font-size: 13.5px;
  font-weight: 600;
}
.send:disabled { opacity: .55; }
.ok-banner {
  margin: 12px 20px 0;
  background: var(--green-soft);
  border: 1px solid #bbf7d0;
  color: var(--green);
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 12.5px;
  line-height: 1.6;
}
.unknown-banner {
  margin: 12px 20px 0;
  background: #fffbeb;
  border: 1px solid #fde68a;
  color: #92400e;
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 12.5px;
  line-height: 1.6;
}
.foot-note { padding: 10px 20px 0; font-size: 10.5px; color: var(--faint); }
</style>
