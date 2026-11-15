<script setup lang="ts">
import { computed } from 'vue'
import type { SystemMetrics, UNASMetrics } from '@/types/metrics'
import { formatBytes } from '@/composables/useFormatters'
import { useFilter } from '@/composables/useFilter'
import { rowsMeta } from '@/lib/systems'
import TermBlock from '@/components/term/TermBlock.vue'
import TextBar from '@/components/term/TextBar.vue'
import PowerBlock from './PowerBlock.vue'

const props = defineProps<{ system: SystemMetrics; metrics: UNASMetrics }>()

const { matches } = useFilter()
const k = (facet: string) => `${props.system.id}:${facet}`

const pools = computed(() => props.metrics.pools.filter(p => matches(p.name, p.status)))
const disks = computed(() => props.metrics.disks.filter(d => matches(d.name, d.model, d.serial, d.status)))
const healthClass = (status: string) =>
  ['healthy', 'online', 'normal', 'ok'].includes(status.toLowerCase()) ? 'ok' : 'warn'
const diskTempClass = (t: number) => (t >= 55 ? 'crit' : t >= 45 ? 'warn' : '')
</script>

<template>
  <TermBlock :id="k('pools')" title="zpool list" :host="system.id" :meta="`${formatBytes(metrics.used_capacity)} / ${formatBytes(metrics.total_capacity)}`">
    <table class="tbl">
      <thead><tr><th>POOL</th><th>STATUS</th><th>USE</th><th class="r">USED</th><th class="r">SIZE</th><th class="r">%</th></tr></thead>
      <tbody>
        <tr v-for="p in pools" :key="p.name">
          <td>{{ p.name }}</td>
          <td :class="healthClass(p.status)">{{ p.status }}</td>
          <td><TextBar :percent="p.percent" :width="14" /></td>
          <td class="r">{{ formatBytes(p.used) }}</td>
          <td class="r">{{ formatBytes(p.total) }}</td>
          <td class="r">{{ p.percent.toFixed(1) }}</td>
        </tr>
      </tbody>
    </table>
  </TermBlock>

  <TermBlock :id="k('disks')" title="smartctl --scan" :meta="rowsMeta(disks.length, metrics.disks.length)">
    <table class="tbl">
      <thead><tr><th>DISK</th><th>MODEL</th><th>SERIAL</th><th>STATUS</th><th class="r">TEMP</th></tr></thead>
      <tbody>
        <tr v-for="d in disks" :key="d.name">
          <td>{{ d.name }}</td>
          <td class="dim">{{ d.model }}</td>
          <td class="dim">{{ d.serial }}</td>
          <td :class="healthClass(d.status)">{{ d.status }}</td>
          <td class="r" :class="d.temperature != null ? diskTempClass(d.temperature) : 'dim'">{{ d.temperature != null ? `${d.temperature.toFixed(0)}°C` : '–' }}</td>
        </tr>
      </tbody>
    </table>
  </TermBlock>

  <PowerBlock :system="system" />
</template>
