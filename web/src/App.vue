<script setup lang="ts">
import { onMounted } from 'vue'
import ConsoleSidebar from './components/ConsoleSidebar.vue'
import ConsoleMain from './components/ConsoleMain.vue'
import { initStore } from './stores/devices'

onMounted(() => {
  void initStore()
})
</script>

<template>
  <div class="app-shell">
    <ConsoleSidebar />
    <ConsoleMain />
  </div>
</template>

<style scoped>
.app-shell {
  height: 100%;
  display: flex;
  background: var(--bg);
}

@media (max-width: 768px) {
  .app-shell {
    flex-direction: column;
  }
}
</style>
