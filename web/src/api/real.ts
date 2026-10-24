/**
 * 真实传输层：调 Go 后端（server/）的 HTTP API。
 * 后端负责：局域网扫描、连接保存、按连接的安全模式与密钥代调手机接口。
 * 前端 dev 模式经 Vite proxy /api → localhost:8080；生产与后端同源。
 */

import type {
  BaseRequest,
  BatteryInfo,
  CallInfo,
  ClonePayload,
  ConfigData,
  ContactInfo,
  LocationInfo,
  SafetyMode,
  SmsInfo,
} from './types'

/* 后端数据结构 */

export interface ConnEntry {
  id: string
  host: string
  port: number
  safetyMode: SafetyMode
  secret?: string
  remark?: string
  createdAt: string
  updatedAt: string
}

export interface ScanFound {
  host: string
  port: number
  state: 'identified' | 'need-credentials' | 'unverified'
  error?: string
  config?: ConfigData
}

export interface ScanProgress {
  phase: 'idle' | 'fast' | 'recheck' | 'done' | 'cancelled' | 'error'
  cidr: string
  port: number
  scanned: number
  total: number
  found: ScanFound[]
  elapsedMs: number
  error?: string
}

/** 后端错误体：kind 为错误分级（connection/auth/feature-off/business/protocol） */
export class BackendError extends Error {
  readonly kind: string
  readonly kindLabel: string
  readonly code?: number

  constructor(message: string, kind = 'connection', kindLabel = '连接失败', code?: number) {
    super(message)
    this.kind = kind
    this.kindLabel = kindLabel
    this.code = code
  }
}

async function call<T>(path: string, init?: RequestInit): Promise<T> {
  let resp: Response
  try {
    resp = await fetch(path, {
      headers: { 'Content-Type': 'application/json' },
      ...init,
    })
  } catch (e) {
    throw new BackendError(
      e instanceof Error ? `无法连接控制台后端：${e.message}` : '无法连接控制台后端',
      'connection',
      '连接失败',
    )
  }
  const text = await resp.text()
  let body: unknown = null
  if (text) {
    try {
      body = JSON.parse(text)
    } catch {
      throw new BackendError(`后端响应异常（HTTP ${resp.status}）`, 'protocol', '协议不符')
    }
  }
  if (!resp.ok) {
    const errObj = body as { error?: string } | null
    throw new BackendError(errObj?.error ?? `HTTP ${resp.status}`, 'business', '业务错误', resp.status)
  }
  if (body && typeof body === 'object' && 'error' in (body as Record<string, unknown>)) {
    const errObj = body as { error: string; kind?: string; kindLabel?: string; code?: number }
    throw new BackendError(errObj.error, errObj.kind ?? 'business', errObj.kindLabel ?? '业务错误', errObj.code)
  }
  return body as T
}

/* ── 健康与网络 ── */

export async function health(): Promise<boolean> {
  try {
    await call<{ ok: boolean }>('/api/health')
    return true
  } catch {
    return false
  }
}

export function networks(): Promise<{ cidrs: string[] }> {
  return call('/api/networks')
}

/* ── 连接管理 ── */

export function listConnections(): Promise<{ items: ConnEntry[] }> {
  return call('/api/connections')
}

export interface ConnInput {
  host: string
  port: number
  safetyMode: SafetyMode
  secret: string
  remark: string
}

/** 手动连接：后端先测试配置查询，成功才保存 */
export function createConnection(input: ConnInput): Promise<{ entry: ConnEntry; config: ConfigData; existed: boolean }> {
  return call('/api/connections', { method: 'POST', body: JSON.stringify(input) })
}

export function updateConnection(id: string, input: ConnInput): Promise<{ entry: ConnEntry }> {
  return call(`/api/connections/${id}`, { method: 'PUT', body: JSON.stringify(input) })
}

export function deleteConnection(id: string): Promise<{ ok: boolean }> {
  return call(`/api/connections/${id}`, { method: 'DELETE' })
}

export function probeConnection(id: string): Promise<{ config: ConfigData }> {
  return call(`/api/connections/${id}/probe`, { method: 'POST' })
}

/* ── 扫描 ── */

export function startScan(cidr: string, port: number): Promise<ScanProgress> {
  return call('/api/scan/start', { method: 'POST', body: JSON.stringify({ cidr, port }) })
}

export function scanStatus(): Promise<ScanProgress> {
  return call('/api/scan/status')
}

export function cancelScan(): Promise<{ ok: boolean }> {
  return call('/api/scan/cancel', { method: 'POST' })
}

/* ── 业务调用（后端按连接凭据代调手机） ── */

function proxy<T>(id: string, phonePath: string, data?: unknown): Promise<T> {
  return call<T>(`/api/connections/${id}${phonePath}`, {
    method: 'POST',
    body: data === undefined ? '{}' : JSON.stringify(data),
  })
}

export const realApi = {
  configQuery: (id: string) => proxy<ConfigData>(id, '/config/query'),
  smsQuery: (id: string, q: { type: 1 | 2; page_num: number; page_size: number; keyword: string }) =>
    proxy<SmsInfo[]>(id, '/sms/query', q),
  smsSend: (id: string, d: { sim_slot: number; phone_numbers: string; msg_content: string }) =>
    proxy<null>(id, '/sms/send', d),
  callQuery: (id: string, q: { type: number; page_num: number; page_size: number; phone_number?: string }) =>
    proxy<CallInfo[]>(id, '/call/query', q),
  contactQuery: (id: string, q: { page_num: number; page_size: number; phone_number?: string; name?: string }) =>
    proxy<ContactInfo[]>(id, '/contact/query', q),
  contactAdd: (id: string, d: { name: string; phone_number: string }) => proxy<null>(id, '/contact/add', d),
  batteryQuery: (id: string) => proxy<BatteryInfo>(id, '/battery/query'),
  locationQuery: (id: string) => proxy<LocationInfo>(id, '/location/query'),
  wolSend: (id: string, d: { mac: string; ip?: string; port?: number }) => proxy<null>(id, '/wol/send', d),
  clonePull: (id: string, body: { version_code: number }) => proxy<ClonePayload>(id, '/clone/pull', body),
  clonePush: (id: string, payload: ClonePayload) => proxy<null>(id, '/clone/push', payload),
}

export type { BaseRequest }
