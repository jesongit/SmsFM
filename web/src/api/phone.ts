/**
 * SIM 卡号码展示美化。
 * 号码来源为手机 SIM 自报（config/query 的 sim_info_list.mNumber，MSISDN）；
 * 部分 SIM 不存本机号或带异常前缀（如 +00），上游无其他获取本机号码的接口。
 */
export function prettyMsisdn(raw: string | null | undefined): string {
  const s = String(raw ?? '').trim()
  if (!s) return ''
  if (s.startsWith('+00')) return '+' + s.slice(3) // +00852... → +852...
  if (s.startsWith('+86')) return s.slice(3)        // +86133... → 133...
  return s
}

/**
 * SIM 展示信息：号码可能来自 mNumber，也可能被 ROM 拼在 mCarrierName 里
 * （如 "中国电信_+8613800138000"）。统一拆出 {名称, 号码}，号码美化。
 */
export function simDisplay(carrier: string | null | undefined, msisdn: string | null | undefined): { name: string; number: string } {
  const c = String(carrier ?? '').trim()
  const m = prettyMsisdn(msisdn)
  if (m) return { name: c.replace(/[+_\-—]*\+?\d{5,}\s*$/, '').trim() || c, number: m }
  // 从运营商名/备注尾部拆号码：最后一个 +数字 或 _数字 段（+ 留在捕获组，供前缀清理）
  const mm = c.match(/^(.+?)[_\-—\s]*((?:\+)?\d{7,})\s*$/)
  if (mm) return { name: mm[1].trim(), number: prettyMsisdn(mm[2]) }
  return { name: c, number: '' }
}
