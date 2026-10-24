/**
 * 与上游 SmsForwarder 3.3.x 线上协议一致的字段结构。
 * 来源：SmsForwarder/app/src/main/kotlin/cn/ppps/forwarder/server/{model,controller} 与 entity/（已逐一核对 Gson @SerializedName）。
 * 语义约束见 docs/development-plan.md §6：HTTP 200 不代表成功，必须检查业务 code。
 */

/** 0=无安全措施 1=时间戳签名 2=RSA 3=SM4（上游 HttpServerUtils.safetyMeasures） */
export type SafetyMode = 0 | 1 | 2 | 3

export const SAFETY_MODE_LABELS: Record<SafetyMode, string> = {
  0: '无安全措施',
  1: '时间戳签名',
  2: 'RSA 加密',
  3: 'SM4 加密',
}

/** 上游统一响应包装（HttpServerUtils.response）：String 且非 "success" 时 code=500、msg=错误文案 */
export interface BaseResponse<T> {
  code: number // 200 成功；500 业务失败
  msg: string
  data?: T
  timestamp: number // 毫秒
  sign?: string // 仅签名模式返回
}

export interface BaseRequest<T> {
  data: T
  /** 签名模式必填（毫秒）；服务端校验时间容差 */
  timestamp?: number
  /** 签名模式：HMAC-SHA256(`${timestamp}\n${secret}`, secret) → Base64 → URL 编码；时间戳签名不是完整防篡改/防重放机制 */
  sign?: string
}

export interface SimInfo {
  mCarrierName?: string | null
  mIccId?: string | null
  mSimSlotIndex: number
  mNumber?: string | null
  mCountryIso?: string | null
  mSubscriptionId: number
}

/** /config/query 返回。九个 enable_api_* 开关决定操作入口是否启用。 */
export interface ConfigData {
  enable_api_clone: boolean
  enable_api_sms_send: boolean
  enable_api_sms_query: boolean
  enable_api_call_query: boolean
  enable_api_contact_query: boolean
  enable_api_contact_add: boolean
  enable_api_battery_query: boolean
  enable_api_wol: boolean
  enable_api_location: boolean
  extra_device_mark: string
  extra_sim1: string
  extra_sim2: string
  /** gson Map<Int, SimInfo>，JSON 键为字符串 */
  sim_info_list: Record<string, SimInfo>
  version_code: number
  version_name: string
}

export interface SmsInfo {
  name: string
  number: string
  content: string
  date: number // epoch 毫秒
  type: 1 | 2 // 1=接收 2=发送
  sim_id: number // 0=SIM1 1=SIM2 -1=获取失败
  sub_id: number
}

export interface SmsQueryData {
  type: 1 | 2
  page_num: number
  page_size: number
  keyword: string
}

export interface SmsSendData {
  sim_slot: number // 1=SIM1 2=SIM2（上游以 sim_slot-1 取 subscription；无法确认 SIM 时前端阻止提交）
  phone_numbers: string
  msg_content: string
}

export interface CallInfo {
  name: string
  number: string
  dateLong: number // epoch 毫秒
  duration: number // 秒
  type: 1 | 2 | 3 // 1=呼入 2=呼出 3=未接（适配层核对）
  via_number: string
  sim_id: number
  sub_id: number
  is_forwarded: boolean
}

export interface CallQueryData {
  type: 1 | 2 | 3
  page_num: number
  page_size: number
  phone_number?: string
}

export interface ContactInfo {
  name: string
  phone_number: string
}

export interface ContactQueryData {
  page_num: number
  page_size: number
  phone_number?: string
  name?: string
}

/** /contact/add 请求结构（适配层核对上游 ContactController） */
export interface ContactAddData {
  name: string
  phone_number: string
}

/** 电池字段上游全部为字符串（Android BatteryManager 常量的字符串形式），由适配层映射语义 */
export interface BatteryInfo {
  level: string
  scale: string
  voltage: string
  temperature: string
  status: string
  health: string
  plugged: string
  msg?: string
}

/** 定位为缓存读取：time 是定位发生时间，不代表刚触发了新定位 */
export interface LocationInfo {
  longitude: number
  latitude: number
  address: string
  time: string
  provider: string
  msg?: string
}

export interface WolData {
  mac: string
  ip?: string
  port?: number // 默认 9
}

/** /clone/pull|push（配置克隆、导出与恢复；恢复会替换设置并清理部分数据） */
export interface ClonePayload {
  [key: string]: unknown
}
