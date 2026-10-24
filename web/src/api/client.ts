/**
 * 传输层：请求信封、安全模式与错误分类。
 * v1 前端先行：transport 可切换 mock / real（Go 后端就绪后接入，见 docs/development-plan.md §5 阶段1）。
 * 错误分类对应开发计划 §3.2：连接失败、鉴权失败、功能关闭、业务错误必须区分展示。
 */

import type {
  BaseResponse,
  BatteryInfo,
  CallQueryData,
  CallInfo,
  ClonePayload,
  ConfigData,
  ContactAddData,
  ContactInfo,
  ContactQueryData,
  LocationInfo,
  SafetyMode,
  SmsInfo,
  SmsQueryData,
  SmsSendData,
  WolData,
} from './types'

/** 与上游一致的错误语义（HTTP 200 也要看 code） */
export type FailureKind = 'connection' | 'auth' | 'feature-off' | 'business' | 'protocol'

export class ApiError extends Error {
  readonly kind: FailureKind
  readonly bizCode?: number

  constructor(kind: FailureKind, message: string, bizCode?: number) {
    super(message)
    this.kind = kind
    this.bizCode = bizCode
  }
}

export const FAILURE_KIND_LABELS: Record<FailureKind, string> = {
  connection: '连接失败',
  auth: '鉴权失败',
  'feature-off': '功能未开启',
  business: '业务错误',
  protocol: '协议不符',
}

export interface ConnectionParams {
  host: string
  port: number
  safetyMode: SafetyMode
  /** 签名模式密钥 / RSA 私钥 / SM4 密钥，按模式取用 */
  secret?: string
  publicKey?: string
}

export interface Transport {
  /** 发送一次请求并解包 BaseResponse；网络层失败抛 ApiError('connection') */
  request<TReq, TRes>(conn: ConnectionParams, path: string, body: TReq): Promise<BaseResponse<TRes>>
}

/** 业务调用门面：每类调用对应一个上游接口 */
export interface SmsFmApi {
  configQuery(conn: ConnectionParams): Promise<ConfigData>
  smsQuery(conn: ConnectionParams, q: SmsQueryData): Promise<SmsInfo[]>
  smsSend(conn: ConnectionParams, d: SmsSendData): Promise<string>
  callQuery(conn: ConnectionParams, q: CallQueryData): Promise<CallInfo[]>
  contactQuery(conn: ConnectionParams, q: ContactQueryData): Promise<ContactInfo[]>
  contactAdd(conn: ConnectionParams, d: ContactAddData): Promise<string>
  batteryQuery(conn: ConnectionParams): Promise<BatteryInfo>
  locationQuery(conn: ConnectionParams): Promise<LocationInfo>
  wolSend(conn: ConnectionParams, d: WolData): Promise<string>
  clonePull(conn: ConnectionParams, body?: { version_code: number }): Promise<ClonePayload>
  clonePush(conn: ConnectionParams, payload: ClonePayload): Promise<string>
}

/** 统一包装：检查业务 code，转换失败种类 */
export async function unwrap<T>(p: Promise<BaseResponse<T>>): Promise<T> {
  let resp: BaseResponse<T>
  try {
    resp = await p
  } catch (e) {
    if (e instanceof ApiError) throw e
    throw new ApiError('connection', e instanceof Error ? e.message : String(e))
  }
  if (resp.code !== 200) {
    throw new ApiError('business', resp.msg || '业务错误', resp.code)
  }
  return resp.data as T
}
