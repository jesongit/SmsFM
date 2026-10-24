<script setup lang="ts">
import { ref } from 'vue'
import { login, login2fa, saveToken } from '../api/auth'

const emit = defineEmits<{ (e: 'logged'): void }>()

const username = ref('')
const password = ref('')
const remember = ref(false)
const busy = ref(false)
const error = ref<string | null>(null)

/* 2FA 第二步 */
const need2fa = ref(false)
const ticket = ref('')
const totpCode = ref('')

async function submit(): Promise<void> {
  if (busy.value) return
  error.value = null
  busy.value = true
  try {
    if (!need2fa.value) {
      if (!username.value || !password.value) {
        error.value = '请输入用户名和密码'
        return
      }
      const res = await login(username.value, password.value, remember.value)
      if (res.need2fa && res.ticket) {
        need2fa.value = true
        ticket.value = res.ticket
        return
      }
      if (res.token) {
        saveToken(res.token, remember.value)
        emit('logged')
      }
    } else {
      if (!/^\d{6}$/.test(totpCode.value.trim())) {
        error.value = '请输入 6 位验证码'
        return
      }
      const res = await login2fa(ticket.value, totpCode.value.trim(), remember.value)
      saveToken(res.token, remember.value)
      emit('logged')
    }
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div class="login-wrap">
    <form class="login-card" @submit.prevent="submit">
      <div class="brand"><span class="logo">S</span>SmsFM 控制台</div>

      <template v-if="!need2fa">
        <label class="field">
          <span>用户名</span>
          <input v-model="username" autocomplete="username" autofocus />
        </label>
        <label class="field">
          <span>密码</span>
          <input v-model="password" type="password" autocomplete="current-password" />
        </label>
        <label class="remember">
          <input v-model="remember" type="checkbox" />
          记住我（30 天内免登录）
        </label>
      </template>

      <template v-else>
        <div class="twofa-hint">已开启两步验证，请输入验证器中的 6 位码</div>
        <label class="field">
          <span>验证码</span>
          <input v-model="totpCode" class="mono code" inputmode="numeric" maxlength="6" placeholder="000000" autofocus />
        </label>
        <label class="remember">
          <input v-model="remember" type="checkbox" />
          记住我（30 天内免登录）
        </label>
        <button type="button" class="back" @click="need2fa = false; error = null">← 返回重新输入密码</button>
      </template>

      <div v-if="error" class="err" role="alert">{{ error }}</div>

      <button class="submit" type="submit" :disabled="busy">
        {{ busy ? '验证中…' : need2fa ? '验证并登录' : '登录' }}
      </button>

      <div class="tip">默认账号与密码均为 admin，首次登录后请到「账号」页修改</div>
    </form>
  </div>
</template>

<style scoped>
.login-wrap {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background:
    radial-gradient(700px 400px at 20% 15%, rgba(37, 99, 235, 0.06), transparent 60%),
    radial-gradient(600px 400px at 85% 85%, rgba(22, 163, 74, 0.05), transparent 60%),
    var(--bg, #f8fafc);
}
.login-card {
  width: 380px;
  max-width: 92vw;
  background: var(--card, #fff);
  border: 1px solid var(--line, #e5e7eb);
  border-radius: 14px;
  padding: 30px 30px 24px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  box-shadow: 0 18px 50px rgba(15, 23, 42, 0.08);
}
.brand {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 19px;
  font-weight: 800;
  justify-content: center;
  margin-bottom: 4px;
}
.logo {
  width: 30px;
  height: 30px;
  border-radius: 8px;
  background: var(--blue, #2563eb);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
}
.field {
  display: flex;
  flex-direction: column;
  gap: 5px;
  font-size: 12px;
  color: var(--dim, #6b7280);
}
.field input {
  border: 1px solid var(--line, #e5e7eb);
  border-radius: 8px;
  padding: 10px 12px;
  font-size: 14px;
  outline: none;
}
.field input:focus-visible {
  border-color: var(--blue, #2563eb);
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.15);
}
.code {
  letter-spacing: 8px;
  font-size: 18px !important;
  text-align: center;
}
.twofa-hint {
  font-size: 12.5px;
  color: var(--dim, #6b7280);
  background: var(--blue-soft, #eff6ff);
  border: 1px solid #bfdbfe;
  border-radius: 8px;
  padding: 8px 12px;
}
.remember {
  display: flex;
  align-items: center;
  gap: 7px;
  font-size: 12.5px;
  color: var(--dim, #6b7280);
  cursor: pointer;
}
.back {
  border: none;
  background: none;
  color: var(--blue, #2563eb);
  font-size: 12px;
  cursor: pointer;
  padding: 0;
  align-self: flex-start;
}
.err {
  background: #fef2f2;
  border: 1px solid #fecaca;
  color: var(--red, #dc2626);
  border-radius: 8px;
  padding: 8px 12px;
  font-size: 12.5px;
}
.submit {
  background: var(--blue, #2563eb);
  border: none;
  color: #fff;
  border-radius: 9px;
  padding: 11px;
  font-size: 14.5px;
  font-weight: 700;
  cursor: pointer;
}
.submit:disabled { opacity: 0.6; cursor: default; }
.tip {
  font-size: 11px;
  color: var(--faint, #9ca3af);
  text-align: center;
}
</style>
