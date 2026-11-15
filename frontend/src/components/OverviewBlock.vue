<script setup lang="ts">
import { computed } from 'vue'
import type { SystemMetrics } from '@/types/metrics'
import { formatBytes, formatClock, formatPercent, formatSpeed, formatUptime } from '@/composables/useFormatters'
import { TYPE_COMMAND, dockerOf, hottest, linuxOf, qbitOf, tempLevel, unasOf, unifiOf } from '@/lib/systems'
import TermBlock from '@/components/term/TermBlock.vue'
import TextBar from '@/components/term/TextBar.vue'
import StatusGlyph from '@/components/term/StatusGlyph.vue'

const props = defineProps<{ system: SystemMetrics }>()

const linux = computed(() => linuxOf(props.system))
const docker = computed(() => dockerOf(props.system))
const qbit = computed(() => qbitOf(props.system))
const unifi = computed(() => unifiOf(props.system))
const unas = computed(() => unasOf(props.system))

const topContainers = computed(() =>
  [...(docker.value?.containers ?? [])]
    .filter(c => c.state === 'running')
    .sort((a, b) => b.cpu_percent - a.cpu_percent)
    .slice(0, 3)
    .map(c => `${c.name} ${c.cpu_percent.toFixed(1)}%`)
    .join(' · '),
)
const unifiOnline = computed(() => unifi.value?.devices.filter(d => d.status === 'online').length ?? 0)
const unifiHot = computed(() => hottest(unifi.value?.devices.flatMap(d => d.temperatures ?? [])))
</script>

<template>
  <TermBlock :id="`overview:${system.id}`" :title="TYPE_COMMAND[system.type]" :host="system.id" :tone="system.status">
    <template #meta><StatusGlyph :status="system.status" /> {{ formatClock(system.last_updated) }}</template>
    <router-link :to="`/system/${system.id}`" class="block">
      <div v-if="system.status === 'offline'" class="crit">✖ {{ system.error ?? 'offline' }}</div>

      <div v-else-if="linux" class="grid grid-cols-[7ch_auto_7ch] justify-start gap-x-2">
        <span>cpu</span><TextBar :percent="linux.cpu.percent" /><span class="text-right">{{ formatPercent(linux.cpu.percent) }}</span>
        <span>mem</span><TextBar :percent="linux.memory.percent" /><span class="text-right">{{ formatPercent(linux.memory.percent) }}</span>
        <template v-for="d in linux.disks.slice(0, 1)" :key="d.mount">
          <span class="truncate">{{ d.mount }}</span><TextBar :percent="d.percent" /><span class="text-right">{{ formatPercent(d.percent) }}</span>
        </template>
        <span class="dim col-span-3">up {{ formatUptime(linux.uptime) }} · load {{ linux.cpu.load_avg.map(v => v.toFixed(2)).join(' ') }}</span>
      </div>

      <div v-else-if="docker">
        <div><span class="ok">{{ docker.running_count }} running</span>&nbsp; <span class="off">{{ docker.stopped_count }} stopped</span>&nbsp; <span class="dim">{{ docker.total_count }} total</span></div>
        <div class="dim truncate">top cpu: {{ topContainers || '–' }}</div>
      </div>

      <div v-else-if="qbit">
        <div><span class="cy">↓ {{ formatSpeed(qbit.download_speed) }}</span>&nbsp; <span class="mg">↑ {{ formatSpeed(qbit.upload_speed) }}</span></div>
        <div class="dim">{{ qbit.active_downloads }} downloading · {{ qbit.active_uploads }} seeding · {{ qbit.total_torrents }} torrents</div>
      </div>

      <div v-else-if="unifi">
        <div><span class="cy">↓ {{ unifi.wan_download.toFixed(1) }} Mbps</span>&nbsp; <span class="mg">↑ {{ unifi.wan_upload.toFixed(1) }} Mbps</span></div>
        <div>
          {{ unifi.clients_count }} clients · {{ unifiOnline }}/{{ unifi.devices.length }} devices up
          <template v-if="unifiHot"> · <span :class="tempLevel(unifiHot)">max {{ unifiHot.current.toFixed(0) }}°C</span></template>
        </div>
      </div>

      <div v-else-if="unas" class="grid grid-cols-[7ch_auto_7ch] justify-start gap-x-2">
        <template v-for="p in unas.pools" :key="p.name">
          <span class="truncate">{{ p.name }}</span><TextBar :percent="p.percent" /><span class="text-right">{{ formatPercent(p.percent) }}</span>
        </template>
        <span class="dim col-span-3">{{ formatBytes(unas.used_capacity) }} of {{ formatBytes(unas.total_capacity) }} · {{ unas.disks.length }} disks</span>
      </div>
    </router-link>
  </TermBlock>
</template>
