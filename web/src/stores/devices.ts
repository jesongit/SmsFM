/**
 * 设备与连接状态。
 * 双模式：后端可用（/api/health）→ real 模式（设备来自后端连接库与扫描）；
 * 后端不可用 → mock 模式（合成演示数据，界面右上角标注）。
 * 连接条目使用控制台内部标识；IP 与备注均可变化，不凭备注合并设备（开发计划 §3.2）。
 */

import { reactive, readonly } from 'vue'
import { createMockApi, MOCK_DEVICES } from '../api/mock'
import { ApiError, FAILURE_KIND_LABELS, type ConnectionParams, type SmsFmApi } from '../api/client'
import * as real from '../api/real'
import type { BatteryInfo, ConfigData, SafetyMode, SmsInfo } from '../api/types'

export type DeviceState = 'online' | 'offline' | 'pending-auth' | 'scanning'

export interface DeviceEntry {
  id: string // 控制台内部标识（real=后端连接 id；mock=演示 id）
  host: string
  port: number
  safetyMode: SafetyMode
  secret?: string
  remark?: string
  state: DeviceState
  config?: ConfigData
  battery?: BatteryInfo
  lastSeen?: number
  lastError?: { kind: string; message: string }
}

interface ConsoleStore {
  mode: 'booting' | 'real' | 'mock'
  devices: DeviceEntry[]
  currentId: string | null
}

const state = reactive<ConsoleStore>({
  mode: 'booting',
  devices: [],
  currentId: null,
})

/* ── 传输层选择 ── */

const mockApi: SmsFmApi = createMockApi()

export const store = readonly(state) as Readonly<ConsoleStore>

export function connOf(d: DeviceEntry): ConnectionParams {
  return { host: d.host, port: d.port, safetyMode: d.safetyMode, secret: d.secret }
}

/** 面向设备的统一业务调用：按模式分发（组件只面对 DeviceEntry） */
export const deviceApi = {
  configQuery: (d: DeviceEntry): Promise<ConfigData> =>
    state.mode === 'real' ? real.realApi.configQuery(d.id) : mockApi.configQuery(connOf(d)),
  smsQuery: (d: DeviceEntry, q: Parameters<SmsFmApi['smsQuery']>[1]): Promise<SmsInfo[]> =>
    state.mode === 'real' ? real.realApi.smsQuery(d.id, q) : mockApi.smsQuery(connOf(d), q),
  smsSend: (d: DeviceEntry, data: Parameters<SmsFmApi['smsSend']>[1]) =>
    state.mode === 'real' ? real.realApi.smsSend(d.id, data) : mockApi.smsSend(connOf(d), data),
  callQuery: (d: DeviceEntry, q: Parameters<SmsFmApi['callQuery']>[1]) =>
    state.mode === 'real' ? real.realApi.callQuery(d.id, q) : mockApi.callQuery(connOf(d), q),
  contactQuery: (d: DeviceEntry, q: Parameters<SmsFmApi['contactQuery']>[1]) =>
    state.mode === 'real' ? real.realApi.contactQuery(d.id, q) : mockApi.contactQuery(connOf(d), q),
  contactAdd: (d: DeviceEntry, data: Parameters<SmsFmApi['contactAdd']>[1]) =>
    state.mode === 'real' ? real.realApi.contactAdd(d.id, data) : mockApi.contactAdd(connOf(d), data),
  batteryQuery: (d: DeviceEntry) =>
    state.mode === 'real' ? real.realApi.batteryQuery(d.id) : mockApi.batteryQuery(connOf(d)),
  locationQuery: (d: DeviceEntry) =>
    state.mode === 'real' ? real.realApi.locationQuery(d.id) : mockApi.locationQuery(connOf(d)),
  wolSend: (d: DeviceEntry, data: Parameters<SmsFmApi['wolSend']>[1]) =>
    state.mode === 'real' ? real.realApi.wolSend(d.id, data) : mockApi.wolSend(connOf(d), data),
  clonePull: (d: DeviceEntry) =>
    state.mode === 'real'
      ? real.realApi.clonePull(d.id, { version_code: d.config?.version_code ?? 0 })
      : mockApi.clonePull(connOf(d)),
  clonePush: (d: DeviceEntry, payload: Parameters<SmsFmApi['clonePush']>[1]) =>
    state.mode === 'real' ? real.realApi.clonePush(d.id, payload) : mockApi.clonePush(connOf(d), payload),
}

/* ── 设备操作 ── */

export function selectDevice(id: string): void {
  state.currentId = id
}

export function currentDevice(): DeviceEntry | null {
  return state.devices.find((d) => d.id === state.currentId) ?? null
}

/** 把错误规整为 UI 展示用（分级标签 + 消息） */
export function errorOf(e: unknown): { kind: string; message: string } {
  if (e instanceof real.BackendError) return { kind: e.kindLabel, message: e.message }
  if (e instanceof ApiError) return { kind: FAILURE_KIND_LABELS[e.kind], message: e.message }
  return { kind: '连接失败', message: e instanceof Error ? e.message : String(e) }
}

/** 连接测试 + 配置识别；失败写入 lastError（不抛出） */
export async function probe(entry: DeviceEntry): Promise<boolean> {
  try {
    const config = await deviceApi.configQuery(entry)
    entry.config = config
    entry.state = 'online'
    entry.lastSeen = Date.now()
    entry.lastError = undefined
    void refreshBattery(entry)
    return true
  } catch (e) {
    const err = errorOf(e)
    entry.state = err.kind === '鉴权失败' ? 'pending-auth' : 'offline'
    entry.lastError = err
    return false
  }
}

export async function refreshBattery(entry: DeviceEntry): Promise<void> {
  try {
    entry.battery = await deviceApi.batteryQuery(entry)
  } catch {
    entry.battery = undefined
  }
}

/* ── 启动 ── */

/** real 模式：从后端装载连接库并逐一探测 */
async function bootReal(): Promise<void> {
  state.mode = 'real'
  await reloadDevices()
  if (state.devices.length) {
    state.currentId = state.devices[0]!.id
    state.devices.forEach((d) => void probe(d))
  }
  startAutoReconnect()
}

export async function reloadDevices(): Promise<void> {
  if (state.mode !== 'real') return
  try {
    const { items } = await real.listConnections()
    const prev = new Map(state.devices.map((d) => [d.id, d]))
    state.devices = items.map((e) => {
      const old = prev.get(e.id)
      return {
        id: e.id,
        host: e.host,
        port: e.port,
        safetyMode: e.safetyMode,
        secret: e.secret,
        remark: e.remark,
        state: old?.state ?? 'scanning',
        config: old?.config,
        battery: old?.battery,
        lastSeen: old?.lastSeen,
        lastError: old?.lastError,
      }
    })
    // 新出现的连接立即探测（补 config，保证名称统一与在线状态）
    state.devices.forEach((d) => {
      if (!d.config) void probe(d)
    })
  } catch {
    /* 后端暂时不可达时保留现有列表 */
  }
}

/* ── 自动重连 ──
   手机息屏时 AndServer 可能不响应（上游已知行为）；恢复可达后自动回到在线。
   页面不可见时暂停，避免后台空耗。 */
const RECONNECT_INTERVAL = 12_000
let reconnectTimer: ReturnType<typeof setInterval> | null = null

function startAutoReconnect(): void {
  if (reconnectTimer) return
  reconnectTimer = setInterval(() => {
    if (typeof document !== 'undefined' && document.visibilityState !== 'visible') return
    if (state.mode !== 'real') return
    for (const d of state.devices) {
      if (d.state !== 'online') void probe(d)
    }
  }, RECONNECT_INTERVAL)
}

/** mock 模式：预填合成机队（note12 待凭据、荣耀 X30 离线，覆盖各错误分支） */
function seedMockDevices(): void {
  for (const d of MOCK_DEVICES) {
    state.devices.push({
      id: d.id,
      host: d.conn.host,
      port: d.conn.port,
      safetyMode: d.conn.safetyMode,
      secret: d.id === 'dev-14c' ? d.conn.secret : undefined,
      state: 'scanning',
    })
  }
  const note12 = state.devices.find((d) => d.id === 'dev-note12')
  if (note12) {
    note12.battery = { level: '64', scale: '100', voltage: '3982', temperature: '31.2', status: '2', health: '2', plugged: '2' }
  }
  state.currentId = MOCK_DEVICES[0]!.id
  state.devices.forEach((d) => void probe(d))
}

/** 应用启动：探测后端，选择 real/mock */
export async function initStore(): Promise<void> {
  if (state.mode !== 'booting') return
  if (await real.health()) {
    await bootReal()
  } else {
    state.mode = 'mock'
    seedMockDevices()
  }
}

/* 演示命名（mock 模式与无备注设备） */
const DEMO_NAMES: Record<string, string> = {
  'dev-14c': '演示手机 A',
  'dev-note12': '测试机 note12',
  'dev-x30': '仓库充电桩',
  'dev-honor': '荣耀 X30',
}

export function deviceDisplayName(d: DeviceEntry | null): string {
  if (!d) return ''
  if (state.mode === 'mock') return DEMO_NAMES[d.id] ?? d.host
  return d.remark || d.config?.extra_device_mark || `${d.host}:${d.port}`
}
