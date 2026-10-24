/**
 * 模拟传输层：结构与线上协议一致的数据，全部为合成演示数据（设备、消息、联系人均为虚构）。
 * 全部为合成演示数据（地址与机型均为占位，非真实设备）。
 * Go 后端就绪后由 real transport 替换，此文件删除。
 */

import type { ConfigData, SimInfo } from './types'
import { ApiError, type ConnectionParams, type SmsFmApi } from './client'

const DELAY = 180

function sim(carrier: string, slot: number, number_: string): SimInfo {
  return {
    mCarrierName: carrier,
    mIccId: `8986${slot}${Math.floor(Math.random() * 1e12).toString().padStart(12, '0')}`,
    mSimSlotIndex: slot,
    mNumber: number_,
    mCountryIso: 'cn',
    mSubscriptionId: slot + 1,
  }
}

interface MockDevice {
  id: string
  conn: ConnectionParams
  config: ConfigData
}

export const MOCK_DEVICES: MockDevice[] = [
  {
    id: 'dev-14c',
    conn: { host: '192.168.1.10', port: 5000, safetyMode: 0 },
    config: {
      enable_api_clone: true,
      enable_api_sms_send: true,
      enable_api_sms_query: true,
      enable_api_call_query: true,
      enable_api_contact_query: true,
      enable_api_contact_add: true,
      enable_api_battery_query: true,
      enable_api_wol: true,
      enable_api_location: false, // 开发计划 §2：真机定位关闭
      extra_device_mark: '演示设备·验证码',
      extra_sim1: '移动',
      extra_sim2: '',
      sim_info_list: {
        '0': sim('中国移动', 0, '138****4321'),
      },
      version_code: 100053,
      version_name: '3.3.2.240815',
    },
  },
  {
    id: 'dev-note12',
    conn: { host: '192.168.1.11', port: 5000, safetyMode: 1, secret: 'demo-sign-key' },
    config: {
      enable_api_clone: true,
      enable_api_sms_send: true,
      enable_api_sms_query: true,
      enable_api_call_query: true,
      enable_api_contact_query: true,
      enable_api_contact_add: false,
      enable_api_battery_query: true,
      enable_api_wol: true,
      enable_api_location: true,
      extra_device_mark: '测试机',
      extra_sim1: '联通',
      extra_sim2: '电信',
      sim_info_list: {
        '0': sim('中国联通', 0, '186****7788'),
        '1': sim('中国电信', 1, '189****0912'),
      },
      version_code: 100053,
      version_name: '3.3.2.240815',
    },
  },
  {
    id: 'dev-x30',
    conn: { host: '192.168.1.12', port: 5000, safetyMode: 0 },
    config: {
      enable_api_clone: false,
      enable_api_sms_send: true,
      enable_api_sms_query: false,
      enable_api_call_query: false,
      enable_api_contact_query: false,
      enable_api_contact_add: false,
      enable_api_battery_query: true,
      enable_api_wol: false,
      enable_api_location: false,
      extra_device_mark: '仓库充电桩',
      extra_sim1: '',
      extra_sim2: '',
      sim_info_list: {},
      version_code: 100041,
      version_name: '3.2.1.240311',
    },
  },
  {
    id: 'dev-honor',
    conn: { host: '192.168.1.13', port: 5000, safetyMode: 0 },
    config: null as unknown as ConfigData, // 离线设备：连接超时，无配置
  },
]

/** 合成短信库：最新在前；首屏时间戳固定以与批准效果图逐字一致（演示数据） */
function makeSms(type: 1 | 2): { name: string; number: string; content: string; date: number; type: 1 | 2; sim_id: number; sub_id: number }[] {
  const d = (h: number, m: number) => new Date(2026, 9, 24, h, m).getTime()
  if (type === 1) {
    return [
      { name: '', number: '10692365018372', content: '【美团】验证码 482913，5分钟内有效。请勿泄露给他人。', date: d(12, 6), type, sim_id: 0, sub_id: 1 },
      { name: '', number: '10086', content: '尊敬的客户，您本月流量剩余 3.2GB，话费余额 42.50 元。回复 QXCX 查询详情。', date: d(11, 22), type, sim_id: 0, sub_id: 1 },
      { name: '房东', number: '13755442321', content: '下个月房租还是 1800，月底前转我就行', date: d(10, 0), type, sim_id: 0, sub_id: 1 },
      { name: '', number: '10690111234567', content: '【京东】您的订单已由快递员揽收，单号 JD1234567890。', date: d(6, 44), type, sim_id: 0, sub_id: 1 },
      { name: '', number: '95588', content: '您尾号6217的账户于10月23日18:02支出 156.00 元（快捷支付），余额 3,204.11 元。', date: d(1, 52), type, sim_id: 0, sub_id: 1 },
      { name: '', number: '10694512120003', content: '【顺丰速运】您的快件已到达【广州白云集散中心】，预计今日派送。', date: d(0, 41), type, sim_id: 0, sub_id: 1 },
      { name: '妈', number: '13901234567', content: '周末回家吃饭吗？你爸买了排骨', date: new Date(2026, 9, 23, 21, 7).getTime(), type, sim_id: 0, sub_id: 1 },
      { name: '', number: '10010', content: '您的话费余额已不足 10 元，请及时充值，以免影响使用。', date: new Date(2026, 9, 23, 18, 30).getTime(), type, sim_id: 0, sub_id: 1 },
    ]
  }
  return [
    { name: '', number: '13755442321', content: '好的房东，周六前转给您', date: d(10, 2), type, sim_id: 0, sub_id: 1 },
    { name: '', number: '13901234567', content: '回，周六中午到', date: new Date(2026, 9, 23, 21, 10).getTime(), type, sim_id: 0, sub_id: 1 },
    { name: '', number: '13800138000', content: '验证码已转发，测试正常', date: new Date(2026, 9, 23, 9, 41).getTime(), type, sim_id: 0, sub_id: 1 },
  ]
}

const MOCK_CALLS: import('./types').CallInfo[] = [
  { name: '房东', number: '13755442321', dateLong: Date.now() - 90 * 60_000, duration: 63, type: 1 as const, via_number: '', sim_id: 0, sub_id: 1, is_forwarded: false },
  { name: '', number: '95338', dateLong: Date.now() - 400 * 60_000, duration: 0, type: 3 as const, via_number: '', sim_id: 0, sub_id: 1, is_forwarded: false },
  { name: '快递员', number: '15918887766', dateLong: Date.now() - 1500 * 60_000, duration: 24, type: 1 as const, via_number: '', sim_id: 0, sub_id: 1, is_forwarded: false },
  { name: '', number: '13755442321', dateLong: Date.now() - 1520 * 60_000, duration: 41, type: 2 as const, via_number: '', sim_id: 0, sub_id: 1, is_forwarded: false },
]

const MOCK_CONTACTS: import('./types').ContactInfo[] = [
  { name: '房东', phone_number: '13755442321' },
  { name: '快递柜', phone_number: '15918887766' },
  { name: '妈', phone_number: '13901234567' },
  { name: '物业前台', phone_number: '0755-86123456' },
  { name: '张工', phone_number: '13822224444' },
]

/** 模拟延迟后直接返回业务数据（错误用 ApiError 表达，无需经过 BaseResponse 解包） */
function r<T>(v: T, ms = DELAY): Promise<T> {
  return new Promise((res) => setTimeout(() => res(v), ms))
}

/** 按设备找到模拟数据源；未知连接或离线样例抛连接失败 */
function deviceOf(conn: ConnectionParams): MockDevice {
  if (conn.host === '192.168.1.13') throw new ApiError('connection', '连接超时（192.168.1.13:5000）')
  const d = MOCK_DEVICES.find((d) => d.conn.host === conn.host && d.conn.port === conn.port)
  if (!d) throw new ApiError('connection', `无法连接 ${conn.host}:${conn.port}`)
  if (conn.safetyMode !== d.conn.safetyMode) {
    throw new ApiError('auth', '签名校验失败：密钥或安全模式不匹配')
  }
  if (conn.safetyMode === 1 && !conn.secret) {
    throw new ApiError('auth', '鉴权失败：签名密钥未填写')
  }
  return d
}

function guard(dev: MockDevice, flag: keyof ConfigData, label: string): void {
  if (!dev.config[flag]) throw new ApiError('feature-off', `${label}未在手机端开启（enable_api_* 为关）`)
}

export function createMockApi(): SmsFmApi {
  return {
    configQuery: (conn) => r(deviceOf(conn).config),
    smsQuery: (conn, q) => {
      const dev = deviceOf(conn)
      guard(dev, 'enable_api_sms_query', '短信查询')
      const all = makeSms(q.type)
        .filter((s) => !q.keyword || s.content.includes(q.keyword) || s.number.includes(q.keyword) || s.name.includes(q.keyword))
      const start = (q.page_num - 1) * q.page_size
      return r(all.slice(start, start + q.page_size) as import('./types').SmsInfo[])
    },
    smsSend: (conn, dd) => {
      const dev = deviceOf(conn)
      guard(dev, 'enable_api_sms_send', '短信发送')
      if (dd.sim_slot !== 1 && dd.sim_slot !== 2) throw new ApiError('business', '无法确认 SIM 卡槽，已阻止提交')
      return r('success')
    },
    callQuery: (conn, q) => {
      const dev = deviceOf(conn)
      guard(dev, 'enable_api_call_query', '通话记录查询')
      const all = MOCK_CALLS.filter((c) => (q.type ? c.type === q.type : true)).filter((c) => !q.phone_number || c.number.includes(q.phone_number))
      const start = (q.page_num - 1) * q.page_size
      return r(all.slice(start, start + q.page_size))
    },
    contactQuery: (conn, q) => {
      const dev = deviceOf(conn)
      guard(dev, 'enable_api_contact_query', '联系人查询')
      const all = MOCK_CONTACTS.filter((c) => (!q.name || c.name.includes(q.name)) && (!q.phone_number || c.phone_number.includes(q.phone_number)))
      const start = (q.page_num - 1) * q.page_size
      return r(all.slice(start, start + q.page_size))
    },
    contactAdd: (conn, data) => {
      const dev = deviceOf(conn)
      guard(dev, 'enable_api_contact_add', '添加联系人')
      void data
      return r('success')
    },
    batteryQuery: (conn) => {
      const dev = deviceOf(conn)
      guard(dev, 'enable_api_battery_query', '电量查询')
      return r(
        dev.id === 'dev-note12'
          ? { level: '64', scale: '100', voltage: '3982', temperature: '31.2', status: '2', health: '2', plugged: '2' } // 充电中（USB）
          : { level: '82', scale: '100', voltage: '4104', temperature: '28.6', status: '3', health: '2', plugged: '0' }, // 放电中
      )
    },
    locationQuery: (conn) => {
      const dev = deviceOf(conn)
      guard(dev, 'enable_api_location', '定位')
      // 缓存定位：时间早于当前，展示定位时间本身（开发计划 §6）
      return r({ longitude: 113.264385, latitude: 23.129112, address: '广东省广州市天河区体育西路', time: '2026-10-24 09:41:22', provider: 'gps', msg: '' })
    },
    wolSend: (conn, d) => {
      const dev = deviceOf(conn)
      guard(dev, 'enable_api_wol', 'WOL')
      void d
      // 上游语义：成功不代表目标电脑已开机
      return r('success', 600)
    },
    clonePull: (conn, _body) => {
      const dev = deviceOf(conn)
      guard(dev, 'enable_api_clone', '配置克隆')
      return r({ clone: { version: dev.config.version_name, mark: dev.config.extra_device_mark, rules: 12, senders: 3 } })
    },
    clonePush: (conn) => {
      const dev = deviceOf(conn)
      guard(dev, 'enable_api_clone', '配置恢复')
      // 上游语义：恢复会替换设置并清理部分数据，前端操作页需明确展示影响范围
      return r('success', 900)
    },
  }
}
