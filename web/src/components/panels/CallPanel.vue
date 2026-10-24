<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { deviceApi, currentDevice, store } from '../../stores/devices'
import type { CallInfo } from '../../api/types'
import ErrorBanner from '../ErrorBanner.vue'

const callType = ref<0 | 1 | 2 | 3>(0) // 0=全部（上游仅支持单类型查询，全部=前端合并三类）
const items = ref<CallInfo[]>([])
const loading = ref(false)
const error = ref<unknown>(null)

const typeLabel: Record<number, string> = { 1: '呼入', 2: '呼出', 3: '未接' }

const PAGE_SIZE = 20
const pageNum = ref(1)
const hasMore = ref(false)

/** 请求序号：丢弃过期响应 */
let querySeq = 0

async function query(page = 1): Promise<void> {
  const seq = ++querySeq
  const d = currentDevice()
  if (!d || d.state !== 'online') { items.value = []; hasMore.value = false; return }
  loading.value = true
  error.value = null
  try {
    if (callType.value === 0) {
      // 「全部」：并发拉三类各一页，按时间归并（上游无跨类型查询）
      const merged: CallInfo[] = []
      await Promise.all([1, 2, 3].map(async (t) => {
        try {
          merged.push(...await deviceApi.callQuery(d, { type: t as 1 | 2 | 3, page_num: page, page_size: PAGE_SIZE }))
        } catch { /* 单类失败不阻塞其余类型 */ }
      }))
      merged.sort((a, b) => b.dateLong - a.dateLong)
      if (seq !== querySeq) return
      items.value = merged
      hasMore.value = merged.length >= PAGE_SIZE
    } else {
      const list = await deviceApi.callQuery(d, {
        type: callType.value as 1 | 2 | 3,
        page_num: page,
        page_size: PAGE_SIZE,
      })
      if (seq !== querySeq) return
      items.value = list
      hasMore.value = list.length >= PAGE_SIZE
    }
    pageNum.value = page
  } catch (e) {
    if (seq !== querySeq) return
    items.value = []
    hasMore.value = false
    error.value = e
  } finally {
    loading.value = false
  }
}

/** 切换类型：立即查询并回到第一页 */
function switchType(t: 0 | 1 | 2 | 3): void {
  if (callType.value === t) return
  callType.value = t
  void query(1)
}

function fmt(ms: number): string {
  const dt = new Date(ms)
  const p = (n: number) => String(n).padStart(2, '0')
  return `${p(dt.getMonth() + 1)}-${p(dt.getDate())} ${p(dt.getHours())}:${p(dt.getMinutes())}`
}

function dur(s: number): string {
  if (!s) return '未接通'
  const m = Math.floor(s / 60)
  return m ? `${m}分${s % 60}秒` : `${s}秒`
}

const rows = computed(() => items.value)

watch(() => store.currentId, () => { void query() })
watch(() => currentDevice()?.state, (s) => { if (s === 'online') void query() })
onMounted(() => { void query() })
</script>

<template>
  <div>
    <div class="tools">
      <div class="seg">
        <span :class="{ on: callType === 0 }" @click="switchType(0)">全部</span>
        <span :class="{ on: callType === 1 }" @click="switchType(1)">呼入</span>
        <span :class="{ on: callType === 2 }" @click="switchType(2)">呼出</span>
        <span :class="{ on: callType === 3 }" @click="switchType(3)">未接</span>
      </div>
      <span class="note">空列表可能与手机端权限有关，不一定是没有记录</span>
    </div>
    <ErrorBanner v-if="error" :error="error" class="err" />
    <div class="table-card">
      <table>
        <thead>
          <tr><th>时间</th><th>对方</th><th>类型</th><th>时长</th><th>SIM</th></tr>
        </thead>
        <tbody>
          <tr v-for="c in rows" :key="c.dateLong + c.number">
            <td class="t mono">{{ fmt(c.dateLong) }}</td>
            <td>
              <div class="num mono">{{ c.number }}</div>
              <div v-if="c.name" class="t">{{ c.name }}</div>
            </td>
            <td><span class="tag" :class="'k' + c.type">{{ typeLabel[c.type] }}</span></td>
            <td class="c">{{ dur(c.duration) }}</td>
            <td><span class="simtag">SIM{{ (c.sim_id ?? 0) + 1 }}</span></td>
          </tr>
          <tr v-if="!loading && !rows.length">
            <td colspan="5" class="empty">未查询到通话记录——空列表可能与手机端权限有关</td>
          </tr>
        </tbody>
      </table>
      <div class="pager">
        <span>第 {{ pageNum }} 页 · 本页 {{ rows.length }} 条<template v-if="hasMore"> · 可能还有更多</template></span>
        <div class="pg">
          <i :class="{ dim: pageNum <= 1 }" @click="pageNum > 1 && query(pageNum - 1)">‹</i>
          <i class="on">{{ pageNum }}</i>
          <i :class="{ dim: !hasMore }" @click="hasMore && query(pageNum + 1)">›</i>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tools {
  display: flex;
  gap: 12px;
  padding: 16px 26px 0;
  align-items: center;
}
.seg {
  display: flex;
  border: 1px solid var(--line);
  border-radius: 8px;
  overflow: hidden;
  background: var(--card);
}
.seg span { padding: 7px 16px; font-size: 13px; color: var(--dim); cursor: pointer; }
.seg span.on { color: var(--blue); background: var(--blue-soft); font-weight: 600; }
.note { font-size: 11.5px; color: var(--faint); }
.err { margin: 12px 26px 0; }
.table-card {
  margin: 16px 26px;
  background: var(--card);
  border: 1px solid var(--line);
  border-radius: 10px;
  overflow: hidden;
}
table { width: 100%; border-collapse: collapse; }
th { text-align: left; font-size: 12px; color: var(--dim); font-weight: 600; padding: 12px 18px; background: #fafbfc; border-bottom: 1px solid var(--line); }
td { padding: 12px 18px; font-size: 13.5px; border-bottom: 1px solid var(--line-soft); vertical-align: top; }
tr:last-child td { border-bottom: none; }
td .num { color: var(--ink); font-weight: 600; }
td .t { color: var(--dim); font-size: 12.5px; }
td .c { color: #374151; }
.empty { text-align: center; color: var(--faint); padding: 26px; }
.pager {
  display: flex;
  align-items: center;
  padding: 10px 18px;
  font-size: 11px;
  color: var(--dim);
  border-top: 1px solid var(--line-soft);
}
.pager .pg { margin-left: auto; display: flex; gap: 6px; }
.pager .pg i {
  width: 21px;
  height: 21px;
  border: 1px solid var(--line);
  border-radius: 5px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-style: normal;
  font-size: 10.5px;
  background: var(--card);
}
.pager .pg i.on { background: var(--blue); color: #fff; border-color: var(--blue); min-width: 28px; }
.pager .pg i.dim { opacity: 0.4; cursor: default; }
.pager .pg i:not(.dim):not(.on) { cursor: pointer; }
.pager .pg i:not(.dim):not(.on):hover { border-color: var(--blue); color: var(--blue); }
.tag { font-size: 11px; padding: 2px 9px; border-radius: 4px; }
.tag.k1 { background: var(--green-soft); color: var(--green); }
.tag.k2 { background: var(--blue-soft); color: var(--blue); }
.tag.k3 { background: #fef3f2; color: var(--red); }
.simtag { font-size: 10.5px; padding: 2px 8px; border-radius: 4px; background: var(--blue-soft); color: var(--blue); }
</style>
