<script setup lang="ts">
import { onMounted, ref } from 'vue'
import {
  changePassword, changeUsername, fetchMe, totpDisable, totpEnable, totpSetup, unauthorized,
  type Me,
} from '../../api/auth'
import QRCode from 'qrcode'

const me = ref<Me | null>(null)
const busy = ref(false)
const msg = ref<{ ok: boolean; text: string } | null>(null)
const err = ref<string | null>(null)

async function loadMe(): Promise<void> {
  try {
    me.value = await fetchMe()
  } catch { /* 401 已由 unauthorized 处理 */ }
}
onMounted(loadMe)

/* 修改用户名 */
const newName = ref('')
const namePwd = ref('')
async function submitUsername(): Promise<void> {
  err.value = null; msg.value = null; busy.value = true
  try {
    await changeUsername(namePwd.value, newName.value.trim())
    msg.value = { ok: true, text: '用户名已修改，下次登录使用新用户名' }
    newName.value = ''; namePwd.value = ''
    await loadMe()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally { busy.value = false }
}

/* 修改密码 */
const oldPwd = ref('')
const newPwd = ref('')
const newPwd2 = ref('')
async function submitPassword(): Promise<void> {
  err.value = null; msg.value = null
  if (newPwd.value !== newPwd2.value) { err.value = '两次输入的新密码不一致'; return }
  busy.value = true
  try {
    await changePassword(oldPwd.value, newPwd.value)
    msg.value = { ok: true, text: '密码已修改' }
    oldPwd.value = ''; newPwd.value = ''; newPwd2.value = ''
    await loadMe()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally { busy.value = false }
}

/* 2FA：绑定流程（扫码/手动输密钥 → 输 6 位码确认） */
const totpStep = ref<'idle' | 'binding'>('idle')
const totpSecret = ref('')
const otpauthUrl = ref('')
const qrDataUrl = ref('')
const bindCode = ref('')
const disablePwd = ref('')

async function startBind(): Promise<void> {
  err.value = null; msg.value = null; busy.value = true
  try {
    const r = await totpSetup()
    totpSecret.value = r.secret
    otpauthUrl.value = r.otpauthUrl
    qrDataUrl.value = await QRCode.toDataURL(r.otpauthUrl, { width: 180, margin: 1 })
    totpStep.value = 'binding'
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally { busy.value = false }
}

async function confirmBind(): Promise<void> {
  err.value = null; msg.value = null
  if (!/^\d{6}$/.test(bindCode.value.trim())) { err.value = '请输入 6 位验证码'; return }
  busy.value = true
  try {
    await totpEnable(totpSecret.value, bindCode.value.trim())
    msg.value = { ok: true, text: '两步验证已开启，此后登录需要输入验证器中的 6 位码' }
    totpStep.value = 'idle'
    bindCode.value = ''
    await loadMe()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally { busy.value = false }
}

async function disable2FA(): Promise<void> {
  err.value = null; msg.value = null; busy.value = true
  try {
    await totpDisable(disablePwd.value)
    msg.value = { ok: true, text: '两步验证已关闭' }
    disablePwd.value = ''
    await loadMe()
  } catch (e) {
    err.value = e instanceof Error ? e.message : String(e)
  } finally { busy.value = false }
}

function exit(): void {
  unauthorized()
}
</script>

<template>
  <div class="account">
    <div v-if="me?.isDefaultPassword" class="default-warn">
      当前仍在使用默认密码（admin/admin），请立即修改
    </div>

    <div v-if="msg" class="banner ok">{{ msg.text }}</div>
    <div v-if="err" class="banner err">{{ err }}</div>

    <div class="sheet">
      <!-- 账号信息 -->
      <div class="row">
        <div class="row-label">当前账号</div>
        <div class="row-body">
          <b class="username">{{ me?.username ?? '…' }}</b>
          <span class="kv"><i>两步验证</i>{{ me?.totpEnabled ? '已开启' : '未开启' }}</span>
          <button class="btn ghost" @click="exit">退出登录</button>
        </div>
      </div>

      <!-- 修改用户名 -->
      <div class="row">
        <div class="row-label">修改用户名</div>
        <div class="row-body">
          <input v-model="newName" class="in" placeholder="新用户名" />
          <input v-model="namePwd" class="in short" type="password" placeholder="当前密码" />
          <button class="btn" :disabled="busy || !newName || !namePwd" @click="submitUsername">保存</button>
        </div>
      </div>

      <!-- 修改密码 -->
      <div class="row">
        <div class="row-label">修改密码</div>
        <div class="row-body">
          <input v-model="oldPwd" class="in short" type="password" placeholder="当前密码" />
          <input v-model="newPwd" class="in short" type="password" placeholder="新密码（≥6位）" />
          <input v-model="newPwd2" class="in short" type="password" placeholder="确认新密码" />
          <button class="btn" :disabled="busy || !oldPwd || !newPwd || !newPwd2" @click="submitPassword">保存</button>
        </div>
      </div>

      <!-- 2FA -->
      <div class="row column">
        <div class="row-head">
          <div class="row-label">两步验证</div>
          <div class="row-body">
            <template v-if="!me?.totpEnabled && totpStep === 'idle'">
              <button class="btn" :disabled="busy" @click="startBind">开启两步验证</button>
              <span class="kv-note">使用 Google Authenticator 等验证器扫码绑定，登录时需额外输入 6 位动态码</span>
            </template>
            <template v-else-if="totpStep === 'binding'">
              <span class="kv-note">用验证器扫描二维码（或手动输入密钥），然后输入 6 位码确认绑定</span>
              <button class="btn ghost" :disabled="busy" @click="totpStep = 'idle'">取消</button>
            </template>
            <template v-else>
              <span class="ok-inline">已开启</span>
              <input v-model="disablePwd" class="in short" type="password" placeholder="当前密码" />
              <button class="btn danger" :disabled="busy || !disablePwd" @click="disable2FA">关闭</button>
            </template>
          </div>
        </div>

        <!-- 绑定：二维码 + 密钥 + 确认码 -->
        <div v-if="totpStep === 'binding'" class="bind-area">
          <div class="bind-qr">
            <img v-if="qrDataUrl" :src="qrDataUrl" alt="TOTP 绑定二维码" width="180" height="180" />
          </div>
          <div class="bind-right">
            <div class="secret-label">手动输入密钥</div>
            <code class="secret mono">{{ totpSecret }}</code>
            <div class="bind-confirm">
              <input v-model="bindCode" class="in mono code" inputmode="numeric" maxlength="6" placeholder="6 位验证码" />
              <button class="btn primary" :disabled="busy" @click="confirmBind">确认绑定</button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.account {
  padding: 16px 26px 24px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.default-warn {
  background: #fffbeb;
  border: 1px solid #fde68a;
  color: #92400e;
  border-radius: 8px;
  padding: 10px 14px;
  font-size: 12.5px;
}
.banner { border-radius: 8px; padding: 10px 14px; font-size: 12.5px; }
.banner.ok { background: var(--green-soft, #dcfce7); border: 1px solid #bbf7d0; color: var(--green, #16a34a); }
.banner.err { background: #fef2f2; border: 1px solid #fecaca; color: var(--red, #dc2626); }

.sheet { background: var(--card, #fff); border: 1px solid var(--line, #e5e7eb); border-radius: 10px; overflow: hidden; }
.row { display: flex; padding: 14px 18px; border-bottom: 1px solid var(--line-soft, #f1f3f5); align-items: baseline; }
.row:last-child { border-bottom: none; }
.row.column { flex-direction: column; gap: 10px; }
.row-head { display: flex; align-items: baseline; }
.row-label { flex: none; width: 92px; font-size: 13px; font-weight: 700; }
.row-body { flex: 1; min-width: 0; display: flex; flex-wrap: wrap; align-items: baseline; gap: 8px 12px; }
.username { font-size: 14px; }
.kv { font-size: 12.5px; color: var(--dim, #6b7280); display: inline-flex; gap: 6px; }
.kv i { font-style: normal; font-size: 11px; color: var(--faint, #9ca3af); }
.kv-note { font-size: 11px; color: var(--faint, #9ca3af); width: 100%; }
.ok-inline { font-size: 12px; color: var(--green, #16a34a); background: var(--green-soft, #dcfce7); border-radius: 6px; padding: 3px 10px; }

.btn {
  border: 1px solid var(--line, #e5e7eb);
  background: var(--card, #fff);
  border-radius: 8px;
  padding: 8px 14px;
  font-size: 12.5px;
  color: var(--ink, #111827);
}
.btn.primary { background: var(--blue, #2563eb); border-color: var(--blue, #2563eb); color: #fff; font-weight: 600; }
.btn.danger { color: var(--red, #dc2626); border-color: #fecaca; }
.btn.ghost { border-style: dashed; color: var(--dim, #6b7280); }
.btn:disabled { opacity: 0.55; cursor: not-allowed; }

.in {
  border: 1px solid var(--line, #e5e7eb);
  border-radius: 7px;
  padding: 8px 11px;
  font-size: 12.5px;
  background: var(--card, #fff);
  color: var(--ink, #111827);
}
.in.short { width: 150px; }
.code { letter-spacing: 6px; text-align: center; width: 130px; }

.bind-area {
  display: flex;
  gap: 18px;
  padding: 4px 0 4px 92px;
  align-items: flex-start;
}
.bind-qr {
  border: 1px solid var(--line, #e5e7eb);
  border-radius: 10px;
  padding: 8px;
  background: #fff;
  flex: none;
}
.bind-right { display: flex; flex-direction: column; gap: 8px; min-width: 0; }
.secret-label { font-size: 11px; color: var(--faint, #9ca3af); }
.secret {
  font-size: 13px;
  background: #f6f7f9;
  border: 1px dashed var(--line, #e5e7eb);
  border-radius: 7px;
  padding: 7px 10px;
  word-break: break-all;
}
.bind-confirm { display: flex; gap: 8px; align-items: center; }

@media (max-width: 768px) {
  .account { padding: 12px 14px 20px; }
  .row { flex-direction: column; gap: 8px; padding: 12px 14px; }
  .row-head { flex-direction: column; }
  .bind-area { padding-left: 0; flex-direction: column; }
}
</style>
