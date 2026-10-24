<script setup lang="ts">
import { computed, ref } from 'vue'
import { deviceDisplayName, selectDevice, store } from '../stores/devices'
import ScanModal from './ScanModal.vue'

const devices = computed(() => store.devices)
const showScan = ref(false)

function deviceName(d: (typeof devices.value)[number]): string {
  return deviceDisplayName(d)
}

/** 电量展示：上游可能返回 "99" 或 "99%"（本地化字符串），统一为带一个 % */
function battText(level: string): string {
  const n = String(level ?? '').trim()
  return n.endsWith('%') ? n : `${n}%`
}

function stateDot(state: string): string {
  if (state === 'online') return 'dot'
  if (state === 'pending-auth') return 'dot warn'
  return 'dot off'
}
</script>

<template>
  <aside class="side">
    <div class="brand"><span class="logo">S</span>SmsFM 控制台</div>
    <button class="scanbtn" @click="showScan = true">扫描局域网设备</button>
    <div class="side-label">设备（{{ devices.length }}）</div>
    <div class="m-devices">
      <select
        class="m-select mono"
        :value="store.currentId ?? ''"
        @change="selectDevice(($event.target as HTMLSelectElement).value)"
      >
        <option v-for="d in devices" :key="d.id" :value="d.id">
          {{ deviceName(d) }}{{ d.battery ? ` · ${d.battery.level}%` : '' }}
        </option>
      </select>
    </div>
    <div
      v-for="d in devices"
      :key="d.id"
      class="dev"
      :class="{ active: d.id === store.currentId }"
      @click="selectDevice(d.id)"
    >
      <span :class="stateDot(d.state)"></span>
      <div class="devinfo">
        <div class="nm">{{ deviceName(d) }}</div>
        <div class="sub mono">{{ d.host }}</div>
      </div>
      <span class="batt">{{ d.battery ? battText(d.battery.level) : '—' }}</span>
    </div>
    <div class="foot">连接设置 · 安全模式 · 已保存 {{ devices.length }} 个连接</div>
    <ScanModal :open="showScan" @close="showScan = false" />
  </aside>
</template>

<style scoped>
.side {
  width: var(--sidebar-w);
  flex: none;
  background: var(--card);
  border-right: 1px solid var(--line);
  display: flex;
  flex-direction: column;
  padding: 18px 14px;
  gap: 4px;
}
.brand {
  font-size: 17px;
  font-weight: 800;
  padding: 6px 10px 16px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.brand .logo {
  width: 26px;
  height: 26px;
  border-radius: 7px;
  background: var(--blue);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
}
.side-label {
  font-size: 11px;
  color: var(--dim);
  letter-spacing: 2px;
  padding: 10px 10px 6px;
}
.scanbtn {
  margin: 0 4px 12px;
  background: var(--blue);
  color: #fff;
  border: none;
  border-radius: 8px;
  padding: 10px;
  font-size: 13.5px;
  font-weight: 600;
}
.dev {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.14s ease;
}
.dev:hover { background: #f3f4f6; }
.dev.active { background: var(--blue-soft); }
.dot { width: 8px; height: 8px; border-radius: 50%; background: var(--green); flex: none; }
.dot.warn { background: var(--amber); }
.dot.off { background: #d1d5db; }
.devinfo { min-width: 0; }
.nm { font-size: 13.5px; font-weight: 600; }
.sub { font-size: 11px; color: var(--dim); margin-top: 1px; }
.batt { margin-left: auto; font-size: 11.5px; color: var(--dim); }
.foot {
  margin-top: auto;
  font-size: 11px;
  color: var(--dim);
  padding: 10px;
  letter-spacing: -0.1px;
}
.m-devices { display: none; }

/* 手机：侧栏折叠为顶部设备选择条 */
@media (max-width: 768px) {
  .side {
    width: 100%;
    flex-direction: row;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    border-right: none;
    border-bottom: 1px solid var(--line);
  }
  .brand { padding: 0; font-size: 15px; }
  .side-label, .dev, .foot { display: none; }
  .scanbtn { margin: 0; padding: 8px 12px; font-size: 12px; flex: none; }
  .m-devices { display: block; flex: 1; min-width: 0; }
  .m-select {
    width: 100%;
    border: 1px solid var(--line);
    border-radius: 8px;
    padding: 8px 10px;
    font-size: 13px;
    background: var(--card);
  }
}
</style>
