<script setup lang="ts">
import { computed, ref } from 'vue'
import type { QBittorrentMetrics, SystemMetrics } from '@/types/metrics'
import { formatBytes, formatEta, formatSpeed } from '@/composables/useFormatters'
import { useActions } from '@/composables/useActions'
import { useFilter } from '@/composables/useFilter'
import { rowsMeta } from '@/lib/systems'
import TermBlock from '@/components/term/TermBlock.vue'
import TextBar from '@/components/term/TextBar.vue'
import ActBtn from '@/components/term/ActBtn.vue'
import ActionPrompt from '@/components/term/ActionPrompt.vue'

const props = defineProps<{ system: SystemMetrics; metrics: QBittorrentMetrics }>()

const { matches } = useFilter()
const runner = useActions(() => props.system)
const selectedHash = ref<string | null>(null)

const torrents = computed(() => props.metrics.torrents.filter(t => matches(t.name, t.state, t.hash)))
const selected = computed(() => props.metrics.torrents.find(t => t.hash === selectedHash.value) ?? null)
const canSelect = computed(() => runner.can('resume'))
const meta = computed(() => rowsMeta(torrents.value.length, props.metrics.torrents.length))

const select = (hash: string) => {
  if (canSelect.value) selectedHash.value = selectedHash.value === hash ? null : hash
}

const stateClass = (state: string) => {
  if (state === 'error' || state === 'missingFiles') return 'crit'
  if (state.startsWith('paused') || state.startsWith('stopped')) return 'off'
  if (state.startsWith('stalled') || state.startsWith('queued') || state.startsWith('checking')) return 'warn'
  return 'ok'
}

const short = (name: string) => (name.length > 40 ? `${name.slice(0, 39)}…` : name)
</script>

<template>
  <TermBlock :id="`${system.id}:stats`" title="qbt stats" :host="system.id" :meta="`${metrics.total_torrents} torrents`">
    <div><span class="cy">↓ {{ formatSpeed(metrics.download_speed) }}</span>&nbsp; <span class="mg">↑ {{ formatSpeed(metrics.upload_speed) }}</span></div>
    <div class="dim">{{ metrics.active_downloads }} downloading · {{ metrics.active_uploads }} seeding</div>
  </TermBlock>

  <TermBlock :id="`${system.id}:torrents`" title="qbt list" :meta="meta">
    <table class="tbl">
      <thead>
        <tr><th>NAME</th><th>PROGRESS</th><th class="r">%</th><th>STATE</th><th class="r">↓</th><th class="r">↑</th><th class="r">ETA</th><th class="r">SIZE</th></tr>
      </thead>
      <tbody>
        <tr
          v-for="t in torrents"
          :key="t.hash"
          :class="{ pick: canSelect, sel: t.hash === selectedHash }"
          @click="select(t.hash)"
        >
          <td :title="t.name">{{ short(t.name) }}</td>
          <td><TextBar :percent="t.progress" :width="10" :warn="101" :crit="101" /></td>
          <td class="r">{{ t.progress.toFixed(1) }}</td>
          <td :class="stateClass(t.state)">{{ t.state }}</td>
          <td class="r cy">{{ t.download_speed ? formatSpeed(t.download_speed) : '–' }}</td>
          <td class="r mg">{{ t.upload_speed ? formatSpeed(t.upload_speed) : '–' }}</td>
          <td class="r dim">{{ formatEta(t.eta) }}</td>
          <td class="r dim">{{ formatBytes(t.size) }}</td>
        </tr>
      </tbody>
    </table>
    <div v-if="canSelect && !selected" class="dim mt-1">select a row for actions</div>
    <div v-if="selected" class="mt-1.5 whitespace-normal">
      <div class="mb-1"><span class="dim">sel</span> {{ selected.name }}</div>
      <div class="flex flex-wrap gap-1">
        <ActBtn :runner="runner" action="resume" :target="selected.hash" :label="`resume ${short(selected.name)}`" />
        <ActBtn :runner="runner" action="pause" :target="selected.hash" :label="`pause ${short(selected.name)}`" />
        <ActBtn :runner="runner" action="recheck" :target="selected.hash" :label="`recheck ${short(selected.name)}`" />
        <ActBtn :runner="runner" action="top_prio" :target="selected.hash" text="top" :label="`top prio ${short(selected.name)}`" />
        <ActBtn :runner="runner" action="increase_prio" :target="selected.hash" text="up" :label="`prio up ${short(selected.name)}`" />
        <ActBtn :runner="runner" action="decrease_prio" :target="selected.hash" text="down" :label="`prio down ${short(selected.name)}`" />
        <ActBtn :runner="runner" action="bottom_prio" :target="selected.hash" text="bottom" :label="`bottom prio ${short(selected.name)}`" />
        <ActBtn :runner="runner" action="delete" :target="selected.hash" :params="{ delete_files: false }" :label="`delete ${short(selected.name)}`" />
        <ActBtn :runner="runner" action="delete" :target="selected.hash" :params="{ delete_files: true }" text="delete+files" :label="`delete ${short(selected.name)} and its files`" />
      </div>
    </div>
    <ActionPrompt :runner="runner" />
  </TermBlock>
</template>
