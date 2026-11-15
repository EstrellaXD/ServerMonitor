<script setup lang="ts">
import { computed } from 'vue'
import type { LinuxMetrics, SystemMetrics } from '@/types/metrics'
import { formatBytes, formatPercent, formatSpeed, formatUptime } from '@/composables/useFormatters'
import { useActions } from '@/composables/useActions'
import { useFilter } from '@/composables/useFilter'
import { tempLevel, rowsMeta } from '@/lib/systems'
import TermBlock from '@/components/term/TermBlock.vue'
import TextBar from '@/components/term/TextBar.vue'
import ActBtn from '@/components/term/ActBtn.vue'
import ActionPrompt from '@/components/term/ActionPrompt.vue'
import FanBlock from './FanBlock.vue'
import PowerBlock from './PowerBlock.vue'

const props = defineProps<{ system: SystemMetrics; metrics: LinuxMetrics }>()

const { matches } = useFilter()
const procRunner = useActions(() => props.system)
const svcRunner = useActions(() => props.system)

const k = (facet: string) => `${props.system.id}:${facet}`
const loadAvg = computed(() => props.metrics.cpu.load_avg.map(v => v.toFixed(2)).join(' '))
const disks = computed(() => props.metrics.disks.filter(d => matches(d.mount)))
const temps = computed(() => props.metrics.temperatures.filter(t => matches(t.label)))
const procs = computed(() => props.metrics.processes.filter(p => matches(p.pid, p.name)))
const services = computed(() => props.metrics.services.filter(s => matches(s.name, s.active_state, s.sub_state)))


const serviceClass = (state: string) =>
  state === 'active' ? 'ok' : state === 'failed' ? 'crit' : state === 'activating' || state === 'deactivating' ? 'warn' : 'off'
</script>

<template>
  <TermBlock :id="k('cpu')" title="cpu" :host="system.id" :meta="`${metrics.cpu.cores.length} cores · up ${formatUptime(metrics.uptime)}`">
    <div class="grid grid-cols-[5ch_auto_7ch] justify-start gap-x-2">
      <span>all</span><TextBar :percent="metrics.cpu.percent" /><span class="text-right">{{ formatPercent(metrics.cpu.percent) }}</span>
    </div>
    <div class="dim mb-1">load {{ loadAvg }}</div>
    <div class="grid grid-cols-[repeat(auto-fill,minmax(24ch,1fr))] gap-x-4">
      <div v-for="(c, i) in metrics.cpu.cores" :key="i" class="grid grid-cols-[4ch_auto_5ch] justify-start gap-x-1">
        <span class="dim">c{{ i }}</span><TextBar :percent="c" :width="12" /><span class="text-right">{{ c.toFixed(0) }}%</span>
      </div>
    </div>
  </TermBlock>

  <TermBlock :id="k('mem')" title="free -h" :meta="`${formatBytes(metrics.memory.total)} total`">
    <div class="grid grid-cols-[5ch_auto_7ch] justify-start gap-x-2">
      <span>mem</span><TextBar :percent="metrics.memory.percent" /><span class="text-right">{{ formatPercent(metrics.memory.percent) }}</span>
    </div>
    <div class="dim">used {{ formatBytes(metrics.memory.used) }} · avail {{ formatBytes(metrics.memory.available) }}</div>
  </TermBlock>

  <TermBlock :id="k('disks')" title="df -h" :meta="rowsMeta(disks.length, metrics.disks.length)">
    <table class="tbl">
      <thead><tr><th>MOUNT</th><th>USE</th><th class="r">USED</th><th class="r">SIZE</th><th class="r">%</th></tr></thead>
      <tbody>
        <tr v-for="d in disks" :key="d.mount">
          <td>{{ d.mount }}</td>
          <td><TextBar :percent="d.percent" :width="14" /></td>
          <td class="r">{{ formatBytes(d.used) }}</td>
          <td class="r">{{ formatBytes(d.total) }}</td>
          <td class="r">{{ d.percent.toFixed(1) }}</td>
        </tr>
      </tbody>
    </table>
  </TermBlock>

  <TermBlock :id="k('net')" title="net">
    <div><span class="cy">↓ {{ formatSpeed(metrics.network.download_rate) }}</span>&nbsp; <span class="mg">↑ {{ formatSpeed(metrics.network.upload_rate) }}</span></div>
    <div class="dim">rx {{ formatBytes(metrics.network.bytes_recv) }} · tx {{ formatBytes(metrics.network.bytes_sent) }}</div>
  </TermBlock>

  <TermBlock v-if="metrics.temperatures.length" :id="k('temps')" title="sensors" :meta="rowsMeta(temps.length, metrics.temperatures.length)">
    <table class="tbl">
      <thead><tr><th>SENSOR</th><th class="r">TEMP</th><th class="r">HIGH</th><th class="r">CRIT</th></tr></thead>
      <tbody>
        <tr v-for="t in temps" :key="t.label">
          <td>{{ t.label }}</td>
          <td class="r" :class="tempLevel(t)">{{ t.current.toFixed(1) }}°C</td>
          <td class="r dim">{{ t.high != null ? `${t.high.toFixed(0)}°C` : '–' }}</td>
          <td class="r dim">{{ t.critical != null ? `${t.critical.toFixed(0)}°C` : '–' }}</td>
        </tr>
      </tbody>
    </table>
  </TermBlock>

  <FanBlock v-if="metrics.fans.length" :system="system" :fans="metrics.fans" />

  <TermBlock :id="k('procs')" title="ps aux --sort=-%cpu" :meta="rowsMeta(procs.length, metrics.processes.length)">
    <table class="tbl">
      <thead>
        <tr>
          <th class="r">PID</th><th>NAME</th><th class="r">CPU%</th><th class="r">MEM%</th><th class="r">RSS</th>
          <th v-if="procRunner.can('kill')" />
        </tr>
      </thead>
      <tbody>
        <tr v-for="p in procs" :key="p.pid">
          <td class="r dim">{{ p.pid }}</td>
          <td class="max-w-[28ch] truncate">{{ p.name }}</td>
          <td class="r">{{ p.cpu_percent.toFixed(1) }}</td>
          <td class="r">{{ p.memory_percent.toFixed(1) }}</td>
          <td class="r dim">{{ formatBytes(p.memory_usage) }}</td>
          <td v-if="procRunner.can('kill')">
            <ActBtn :runner="procRunner" action="kill" :target="String(p.pid)" :params="{ signal: 'TERM' }" text="term" :label="`kill -TERM ${p.pid} (${p.name})`" />
            <ActBtn :runner="procRunner" action="kill" :target="String(p.pid)" :params="{ signal: 'KILL' }" text="kill" :label="`kill -KILL ${p.pid} (${p.name})`" />
          </td>
        </tr>
      </tbody>
    </table>
    <ActionPrompt :runner="procRunner" />
  </TermBlock>

  <TermBlock v-if="metrics.services.length" :id="k('services')" title="systemctl status" :meta="rowsMeta(services.length, metrics.services.length)">
    <table class="tbl">
      <thead><tr><th>UNIT</th><th>ACTIVE</th><th>SUB</th><th v-if="svcRunner.can('service_restart')" /></tr></thead>
      <tbody>
        <tr v-for="s in services" :key="s.name">
          <td>{{ s.name }}</td>
          <td :class="serviceClass(s.active_state)">{{ s.active_state }}</td>
          <td class="dim">{{ s.sub_state }}</td>
          <td v-if="svcRunner.can('service_restart')">
            <ActBtn :runner="svcRunner" action="service_start" :target="s.name" text="start" :label="`systemctl start ${s.name}`" />
            <ActBtn :runner="svcRunner" action="service_stop" :target="s.name" text="stop" :label="`systemctl stop ${s.name}`" />
            <ActBtn :runner="svcRunner" action="service_restart" :target="s.name" text="restart" :label="`systemctl restart ${s.name}`" />
          </td>
        </tr>
      </tbody>
    </table>
    <ActionPrompt :runner="svcRunner" />
  </TermBlock>

  <PowerBlock :system="system" />
</template>
