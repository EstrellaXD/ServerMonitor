<script setup lang="ts">
import { nextTick, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useMetricsStore } from '@/stores/metrics'
import StatusGlyph from '@/components/term/StatusGlyph.vue'

const store = useMetricsStore()
const route = useRoute()
const nav = ref<HTMLElement | null>(null)
const isOn = (id?: string) => (id ? route.params.id === id : route.name === 'dashboard')

// In the narrow tab row the active tab can sit off-screen; keep it visible.
watch(() => [route.fullPath, store.systemsCount], async () => {
  await nextTick()
  nav.value?.querySelector('.item.on')?.scrollIntoView({ block: 'nearest', inline: 'nearest' })
})
</script>

<template>
  <nav
    ref="nav"
    class="nav sticky top-0 z-10 flex overflow-x-auto border-b border-line bg-bg sm:h-screen sm:flex-col sm:overflow-y-auto sm:border-b-0 sm:border-r sm:py-2.5"
    aria-label="systems"
  >
    <router-link to="/" class="item" :class="{ on: isOn() }">
      <span class="ok">❯</span><span>overview</span><span class="key">esc</span>
    </router-link>
    <router-link
      v-for="(s, i) in store.systemsList"
      :key="s.id"
      :to="`/system/${s.id}`"
      class="item"
      :class="{ on: isOn(s.id) }"
      :title="s.name"
    >
      <StatusGlyph :status="s.status" /><span :class="{ dim: s.status === 'offline' }" class="truncate">{{ s.id }}</span><span v-if="i < 9" class="key">{{ i + 1 }}</span>
    </router-link>
  </nav>
</template>

<style scoped>
.item {
  display: flex;
  align-items: baseline;
  gap: 6px;
  padding: 4px 12px;
  white-space: nowrap;
  flex-shrink: 0;
}
.item:hover { background: var(--t-hover); }
.item.on { background: var(--t-sel); box-shadow: inset 0 -2px 0 var(--ok); }
.key { margin-left: auto; padding-left: 8px; color: var(--t-dim); font-size: 11px; }

@media (min-width: 640px) {
  .item { padding: 3px 12px; }
  .item.on { box-shadow: inset 2px 0 0 var(--ok); }
}
@media (max-width: 639px) {
  .nav { scrollbar-width: none; }
  .key { display: none; }
}
</style>
