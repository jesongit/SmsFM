<script setup lang="ts">
import { computed, ref, type Component } from 'vue'
import { currentDevice, refreshBattery, store } from '../stores/devices'
import SmsPanel from './panels/SmsPanel.vue'
import type { ConfigData } from '../api/types'
import CallPanel from './panels/CallPanel.vue'
import ContactPanel from './panels/ContactPanel.vue'
import OtherPanel from './panels/OtherPanel.vue'
import AccountPanel from './panels/AccountPanel.vue'
import { logout } from '../api/auth'
import ComposeDrawer from './ComposeDrawer.vue'
import SettingsDrawer from './SettingsDrawer.vue'
import ScanModal from './ScanModal.vue'

const device = computed(() => currentDevice())

/* 标签页：入口按 config 的功能开关启用（开发计划 §1） */
const tabs = computed(() => {
  const c = device.value?.config
  // 电量/定位/WOL/配置合并为「其他」页（内容量少）；设备信息也在其中
  const enabledAny = (c: ConfigData | undefined, keys: string[]): boolean =>
    !c || keys.some((k) => Boolean(c[k as keyof ConfigData]))
  interface TabDef { key: string; label: string; flag: string; enabled: boolean; label2: string }
  const def: { key: string; label: string; flag: string }[] = [
    { key: 'sms', label: '短信', flag: 'enable_api_sms_query' },
    { key: 'call', label: '通话', flag: 'enable_api_call_query' },
    { key: 'contact', label: '联系人', flag: 'enable_api_contact_query' },
  ]
  const tabsList: TabDef[] = def.map((t) => ({
    ...t,
    enabled: !c || Boolean(c[t.flag as keyof ConfigData]),
    label2: t.label,
  }))
  tabsList.push({
    key: 'other', label: '其他', flag: 'enable_api_battery_query',
    enabled: enabledAny(c, ['enable_api_battery_query', 'enable_api_location', 'enable_api_wol', 'enable_api_clone']),
    label2: '其他',
  })
  tabsList.push({ key: 'account', label: '账号', flag: '', enabled: true, label2: '账号' })
  return tabsList
})
const activeTab = ref<string>('sms')

/* 当前面板组件；:key 绑定设备 id——切设备时重建（新设备需要重新查询），切面板时缓存 */
const PANELS: Record<string, Component> = {
  sms: SmsPanel,
  call: CallPanel,
  contact: ContactPanel,
  other: OtherPanel,
  account: AccountPanel,
}

async function doLogout(): Promise<void> {
  await logout()
  location.reload()
}
const currentPanel = computed(() => PANELS[activeTab.value] ?? SmsPanel)

const showCompose = ref(false)
const showSettings = ref(false)
const showScan = ref(false)

function openSettings(): void {
  showSettings.value = true
}

function refresh(): void {
  const d = device.value
  if (d) void refreshBattery(d)
}
</script>

<template>
  <main class="main">
    <div class="tabs">
      <div
        v-for="t in tabs"
        :key="t.key"
        class="tab"
        :class="{ active: t.key === activeTab && t.enabled, off: !t.enabled }"
        @click="t.enabled && (activeTab = t.key)"
      >{{ t.label2 }}</div>
      <div class="tab-actions">
        <span v-if="store.mode === 'mock'" class="demo-tag" title="当前界面全部为合成演示数据，非真实设备内容">演示数据为合成</span>
        <button class="btn" @click="refresh">刷新状态</button>
        <button class="btn" @click="openSettings">连接设置</button>
        <button class="btn primary" @click="showCompose = true">写短信</button>
        <button class="btn" title="退出登录" @click="doLogout">退出</button>
      </div>
    </div>

    <!-- KeepAlive：切换标签页保留面板与已查数据（key 含页签+设备：切页签换缓存面板，切设备重建） -->
    <KeepAlive>
      <component :is="currentPanel" :key="activeTab + '-' + (store.currentId ?? 'none')" />
    </KeepAlive>

    <ComposeDrawer :open="showCompose" @close="showCompose = false" />
    <SettingsDrawer :open="showSettings" @close="showSettings = false" />
    <ScanModal :open="showScan" @close="showScan = false" />
  </main>
</template>

<style scoped>
.main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  overflow-y: auto;
}
.badge.pending-auth { background: #fffbeb; color: var(--amber); }
.badge.offline, .badge.scanning { background: #f2f4f7; color: var(--faint); }
.btn {
  border: 1px solid var(--line);
  background: var(--card);
  border-radius: 8px;
  padding: 8px 16px;
  font-size: 13px;
  color: var(--ink);
}
.btn.primary {
  background: var(--blue);
  border-color: var(--blue);
  color: #fff;
  font-weight: 600;
}
.tabs {
  display: flex;
  gap: 22px;
  padding: 0 26px;
  background: var(--card);
  border-bottom: 1px solid var(--line);
  position: sticky;
  top: 0;
  z-index: 4;
  align-items: center;
}
.tab-actions {
  margin-left: auto;
  display: flex;
  gap: 8px;
  align-items: center;
  padding: 8px 0;
}
.tab {
  padding: 13px 2px;
  font-size: 14px;
  color: var(--dim);
  border-bottom: 2px solid transparent;
  cursor: pointer;
  transition: color 0.14s ease, border-color 0.14s ease;
}
.tab.active {
  color: var(--blue);
  border-bottom-color: var(--blue);
  font-weight: 600;
}
.tab.off { color: #c3c8d2; cursor: not-allowed; }

.demo-tag {
  font-size: 10.5px;
  color: var(--faint);
  border: 1px dashed var(--line);
  border-radius: 10px;
  padding: 2px 8px;
  flex: none;
}

@media (max-width: 768px) {
  .tabs { gap: 14px; padding: 0 14px; overflow-x: auto; position: static; }
  .tab { flex: none; }
  .tab-actions { padding: 6px 0; }
  .tab-actions .btn { padding: 6px 10px; font-size: 12px; }
}
</style>
