<script setup lang="ts">
import { computed } from 'vue'
import { errorOf } from '../stores/devices'

const props = defineProps<{ error: unknown }>()

const parsed = computed(() => errorOf(props.error))
</script>

<template>
  <div class="err-banner" role="alert">
    <b>{{ parsed.kind }}</b>
    <span>{{ parsed.message }}</span>
  </div>
</template>

<style scoped>
.err-banner {
  display: flex;
  gap: 10px;
  align-items: baseline;
  padding: 10px 14px;
  border-radius: 8px;
  background: #fef2f2;
  border: 1px solid #fecaca;
  font-size: 13px;
}
.err-banner b {
  color: var(--red);
  font-weight: 600;
  flex: none;
}
.err-banner span { color: #7f1d1d; }
</style>
