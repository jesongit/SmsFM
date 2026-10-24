<script setup lang="ts">
import { onMounted, ref } from 'vue'
import ConsoleSidebar from './components/ConsoleSidebar.vue'
import ConsoleMain from './components/ConsoleMain.vue'
import LoginView from './components/LoginView.vue'
import { initStore } from './stores/devices'
import { getToken, setUnauthorizedHandler } from './api/auth'

const logged = ref(false)

setUnauthorizedHandler(() => { logged.value = false })

onMounted(() => {
  logged.value = Boolean(getToken())
  if (logged.value) void initStore()
})
function onLogged(): void {
  logged.value = true
  void initStore()
}
</script>

<template>
  <LoginView v-if="!logged" @logged="onLogged" />
  <div v-else class="app-shell">
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
