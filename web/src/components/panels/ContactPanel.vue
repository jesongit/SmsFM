<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { deviceApi, currentDevice, store } from '../../stores/devices'
import type { ContactInfo } from '../../api/types'
import ErrorBanner from '../ErrorBanner.vue'

const kwName = ref('')
const items = ref<ContactInfo[]>([])
const loading = ref(false)
const error = ref<unknown>(null)

/* 新增联系人 */
const addOpen = ref(false)
const addName = ref('')
const addPhone = ref('')
const addBusy = ref(false)
const addResult = ref<string | null>(null)

/** 请求序号：丢弃过期响应 */
let querySeq = 0

async function query(): Promise<void> {
  const seq = ++querySeq
  const d = currentDevice()
  if (!d || d.state !== 'online') { items.value = []; return }
  loading.value = true
  error.value = null
  try {
    const list = await deviceApi.contactQuery(d, {
      page_num: 1,
      page_size: 20,
      name: kwName.value || undefined,
    })
    if (seq !== querySeq) return
    items.value = list
  } catch (e) {
    if (seq !== querySeq) return
    items.value = []
    error.value = e
  } finally {
    loading.value = false
  }
}

async function submitAdd(): Promise<void> {
  const d = currentDevice()
  if (!d) return
  addBusy.value = true
  addResult.value = null
  try {
    await deviceApi.contactAdd(d, { name: addName.value, phone_number: addPhone.value })
    addResult.value = '已提交添加请求，可在列表中刷新核对'
    addOpen.value = false
    addName.value = ''
    addPhone.value = ''
    await query()
  } catch (e) {
    addResult.value = e instanceof Error ? e.message : String(e)
  } finally {
    addBusy.value = false
  }
}

// firstLetter 为上游计算属性不随 JSON 序列化，本地按姓名首字分组
const letter = (c: ContactInfo): string => (c.name || '#').slice(0, 1).toUpperCase()
const grouped = computed(() => {
  const map = new Map<string, ContactInfo[]>()
  for (const c of [...items.value].sort((a, b) => a.name.localeCompare(b.name, 'zh'))) {
    const k = letter(c)
    if (!map.has(k)) map.set(k, [])
    map.get(k)!.push(c)
  }
  return [...map.entries()]
})

watch(() => store.currentId, () => { void query() })
watch(() => currentDevice()?.state, (s) => { if (s === 'online') void query() })
onMounted(() => { void query() })
</script>

<template>
  <div>
    <div class="tools">
      <input v-model="kwName" class="search" placeholder="按姓名过滤" @keydown.enter="query" />
      <button class="btn" @click="query">查询</button>
      <button class="btn primary add" @click="addOpen = !addOpen">新增联系人</button>
      <span class="note">分页行为以适配层核对结果为准</span>
    </div>
    <form v-if="addOpen" class="add-card" @submit.prevent="submitAdd">
      <label>姓名<input v-model="addName" required placeholder="联系人姓名" /></label>
      <label>电话<input v-model="addPhone" required placeholder="手机号码" /></label>
      <button class="btn primary" type="submit" :disabled="addBusy">{{ addBusy ? '提交中…' : '添加' }}</button>
    </form>
    <p v-if="addResult" class="add-result">{{ addResult }}</p>
    <ErrorBanner v-if="error" :error="error" class="err" />
    <div class="table-card">
      <div v-if="!error && !loading && !grouped.length" class="empty">未查询到联系人——空列表可能与手机端权限有关</div>
      <div v-for="[k, list] in grouped" :key="k" class="letter-group">
        <div class="letter">{{ k }}</div>
        <div v-for="c in list" :key="c.phone_number + c.name" class="row">
          <div class="who"><b>{{ c.name }}</b></div>
          <div class="mono phone">{{ c.phone_number }}</div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tools { display: flex; gap: 10px; padding: 16px 26px 0; align-items: center; }
.search { flex: 0 1 260px; border: 1px solid var(--line); border-radius: 8px; padding: 8px 12px; font-size: 13px; background: var(--card); outline: none; }
.btn { border: 1px solid var(--line); background: var(--card); border-radius: 8px; padding: 8px 16px; font-size: 13px; }
.btn.primary { background: var(--blue); border-color: var(--blue); color: #fff; font-weight: 600; }
.btn:disabled { opacity: .6; }
.add { margin-left: auto; }
.note { font-size: 11.5px; color: var(--faint); }
.add-card {
  margin: 14px 26px 0;
  background: var(--card);
  border: 1px solid var(--line);
  border-radius: 10px;
  padding: 14px 16px;
  display: flex;
  gap: 14px;
  align-items: end;
}
.add-card label { display: flex; flex-direction: column; gap: 4px; font-size: 12px; color: var(--dim); }
.add-card input { border: 1px solid var(--line); border-radius: 7px; padding: 8px 12px; font-size: 13px; width: 220px; }
.add-result { margin: 10px 26px 0; font-size: 12.5px; color: var(--dim); padding: 0 2px; }
.err { margin: 12px 26px 0; }
.table-card { margin: 16px 26px; background: var(--card); border: 1px solid var(--line); border-radius: 10px; overflow: hidden; padding: 6px 0 14px; }
.empty { text-align: center; color: var(--faint); padding: 30px; font-size: 13px; }
.letter { font-size: 11.5px; color: var(--faint); font-weight: 700; padding: 12px 22px 4px; }
.row { display: flex; align-items: baseline; padding: 8px 22px; border-bottom: 1px solid var(--line-soft); }
.row:last-child { border-bottom: none; }
.who b { font-size: 13.5px; font-weight: 600; }
.phone { margin-left: 16px; font-size: 12.5px; color: var(--dim); }
</style>
