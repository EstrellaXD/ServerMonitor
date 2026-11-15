<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useMetricsStore } from '@/stores/metrics'
import { useFilter } from '@/composables/useFilter'
import { formatClock } from '@/composables/useFormatters'
import { STATUS_CLASS, TYPE_LABEL, summarize } from '@/lib/systems'
import TermBlock from '@/components/term/TermBlock.vue'
import OverviewBlock from '@/components/OverviewBlock.vue'

const store = useMetricsStore()
const router = useRouter()
const { matches } = useFilter()

const systems = computed(() =>
  store.systemsList.filter(s => matches(s.id, s.name, s.type, TYPE_LABEL[s.type], s.status)),
)
const meta = computed(() => {
  const n = systems.value.length === store.systemsCount ? `${store.systemsCount}` : `${systems.value.length}/${store.systemsCount}`
  return `${formatClock(store.lastUpdate)} · ${n} systems`
})
const open = (id: string) => router.push(`/system/${id}`)
</script>

<template>
  <TermBlock id="overview:status" title="status --all" :meta="meta">
    <div>
      <span class="ok">HEALTHY {{ store.healthyCount }}</span>&nbsp; <span class="warn">WARNING {{ store.warningCount }}</span>&nbsp; <span class="crit">CRITICAL {{ store.criticalCount }}</span>&nbsp; <span class="off">OFFLINE {{ store.offlineCount }}</span>
    </div>
    <div v-if="!store.systemsCount" class="dim">waiting for first update…</div>
    <table v-else class="tbl mt-1">
      <thead><tr><th>SYSTEM</th><th>TYPE</th><th>STATE</th><th>SUMMARY</th></tr></thead>
      <tbody>
        <tr v-for="s in systems" :key="s.id" class="pick" @click="open(s.id)">
          <td>{{ s.id }}</td>
          <td class="dim">{{ TYPE_LABEL[s.type] }}</td>
          <td :class="STATUS_CLASS[s.status]">{{ s.status }}</td>
          <td :class="{ crit: s.status === 'offline' }">{{ summarize(s) }}</td>
        </tr>
      </tbody>
    </table>
  </TermBlock>

  <OverviewBlock v-for="s in systems" :key="s.id" :system="s" />
</template>
