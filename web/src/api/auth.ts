/**
 * 认证：token 管理（记住我 → localStorage 持久；否则 sessionStorage 关浏览器失效）
 * 与登录/账号 API。
 */

const TOKEN_KEY = 'smsfm_token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY) ?? sessionStorage.getItem(TOKEN_KEY)
}

export function saveToken(token: string, remember: boolean): void {
  clearToken()
  ;(remember ? localStorage : sessionStorage).setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
  sessionStorage.removeItem(TOKEN_KEY)
}

/** 未授权（401）时的回调，由 App 注册（清 token 并回到登录页） */
let onUnauthorized: (() => void) | null = null
export function setUnauthorizedHandler(fn: () => void): void {
  onUnauthorized = fn
}
export function unauthorized(): void {
  clearToken()
  onUnauthorized?.()
}

export interface Me {
  username: string
  totpEnabled: boolean
  isDefaultPassword: boolean
}

async function authCall<T>(path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' }
  const token = getToken()
  if (token) headers.Authorization = `Bearer ${token}`
  const resp = await fetch(path, {
    method: 'POST',
    headers,
    body: body === undefined ? '{}' : JSON.stringify(body),
  })
  const text = await resp.text()
  let data: unknown = null
  if (text) {
    try {
      data = JSON.parse(text)
    } catch {
      throw new Error(`响应异常（HTTP ${resp.status}）`)
    }
  }
  if (!resp.ok) {
    if (resp.status === 401) {
      if (path !== '/api/auth/login' && path !== '/api/auth/login-2fa') unauthorized()
      throw new Error((data as { error?: string })?.error ?? `HTTP ${resp.status}`)
    }
    throw new Error((data as { error?: string })?.error ?? `HTTP ${resp.status}`)
  }
  return data as T
}

export async function login(username: string, password: string, remember: boolean): Promise<{ token?: string; need2fa?: boolean; ticket?: string }> {
  return authCall('/api/auth/login', { username, password, remember })
}

export async function login2fa(ticket: string, code: string, remember: boolean): Promise<{ token: string }> {
  return authCall('/api/auth/login-2fa', { ticket, code, remember })
}

export async function logout(): Promise<void> {
  try {
    await authCall('/api/auth/logout')
  } catch { /* token 已失效也继续退出 */ }
  clearToken()
}

export async function fetchMe(): Promise<Me> {
  const resp = await fetch('/api/auth/me', { headers: { Authorization: `Bearer ${getToken() ?? ''}` } })
  if (!resp.ok) {
    unauthorized()
    throw new Error('未登录')
  }
  return resp.json()
}

export async function changePassword(oldPassword: string, newPassword: string): Promise<void> {
  await authCall('/api/auth/change-password', { oldPassword, newPassword })
}

export async function changeUsername(password: string, newUsername: string): Promise<void> {
  await authCall('/api/auth/change-username', { password, newUsername })
}

export async function totpSetup(): Promise<{ secret: string; otpauthUrl: string }> {
  return authCall('/api/auth/totp/setup')
}

export async function totpEnable(secret: string, code: string): Promise<void> {
  await authCall('/api/auth/totp/enable', { secret, code })
}

export async function totpDisable(password: string): Promise<void> {
  return authCall('/api/auth/totp/disable', { password })
}
