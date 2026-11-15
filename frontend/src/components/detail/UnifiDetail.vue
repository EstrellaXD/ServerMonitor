<script setup lang="ts">
import { computed } from 'vue'
import type { SystemMetrics, UnifiClient, UnifiMetrics } from '@/types/metrics'
import { formatUptime } from '@/composables/useFormatters'
import { useActions } from '@/composables/useActions'
import { useFilter } from '@/composables/useFilter'
import { hottest, tempLevel, rowsMeta } from '@/lib/systems'
import TermBlock from '@/components/term/TermBlock.vue'
import ActBtn from '@/components/term/ActBtn.vue'
import ActionPrompt from '@/components/term/ActionPrompt.vue'

const props = defineProps<{ system: SystemMetrics; metrics: UnifiMetrics }>()

const { matches } = useFilter()
const deviceRunner = useActions(() => props.system)
const clientRunner = useActions(() => props.system)
const blockedRunner = useActions(() => props.system)

const k = (facet: string) => `${props.system.id}:${facet}`
const clientMatch = (c: UnifiClient) => matches(c.name, c.ip, c.mac)

const devices = computed(() => props.metrics.devices.filter(d => matches(d.name, d.model, d.mac, d.status)))
const clients = computed(() => props.metrics.clients.filter(clientMatch))
const blocked = computed(() => props.metrics.blocked_clients.filter(clientMatch))
const clientName = (c: UnifiClient) => c.name || c.mac
</script>

<template>
  <TermBlock :id="k('wan')" title="unifi wan" :host="system.id" :meta="`${metrics.devices.length} devices`">
    <div><span class="cy">↓ {{ metrics.wan_download.toFixed(1) }} Mbps</span>&nbsp; <span class="mg">↑ {{ metrics.wan_upload.toFixed(1) }} Mbps</span></div>
    <div class="dim">{{ metrics.clients_count }} clients · {{ metrics.guests_count }} guests</div>
  </TermBlock>

  <TermBlock :id="k('devices')" title="unifi devices" :meta="rowsMeta(devices.length, metrics.devices.length)">
    <table class="tbl">
      <thead><tr><th>NAME</th><th>MODEL</th><th>STATUS</th><th class="r">UPTIME</th><th class="r">TEMP</th><th v-if="deviceRunner.can('restart_device')" /></tr></thead>
      <tbody>
        <tr v-for="d in devices" :key="d.mac">
          <td>{{ d.name }}</td>
          <td class="dim">{{ d.model }}</td>
          <td :class="d.status === 'online' ? 'ok' : 'off'">{{ d.status }}</td>
          <td class="r">{{ d.status === 'online' ? formatUptime(d.uptime) : '–' }}</td>
          <td class="r">
            <template v-if="hottest(d.temperatures)">
              <span :class="tempLevel(hottest(d.temperatures)!)" :title="(d.temperatures ?? []).map(t => `${t.label} ${t.current.toFixed(0)}°C`).join(', ')">
                {{ hottest(d.temperatures)!.current.toFixed(0) }}°C
              </span>
            </template>
            <span v-else class="dim">–</span>
          </td>
          <td v-if="deviceRunner.can('restart_device')">
            <ActBtn :runner="deviceRunner" action="restart_device" :target="d.mac" text="restart" :label="`restart device ${d.name}`" />
          </td>
        </tr>
      </tbody>
    </table>
    <ActionPrompt :runner="deviceRunner" />
  </TermBlock>

  <TermBlock :id="k('clients')" title="unifi clients" :meta="rowsMeta(clients.length, metrics.clients.length)">
    <table class="tbl">
      <thead><tr><th>NAME</th><th>IP</th><th>MAC</th><th /><th v-if="clientRunner.can('kick')" /></tr></thead>
      <tbody>
        <tr v-for="c in clients" :key="c.mac">
          <td class="max-w-[24ch] truncate">{{ clientName(c) }}</td>
          <td>{{ c.ip || '–' }}</td>
          <td class="dim">{{ c.mac }}</td>
          <td class="warn">{{ c.is_guest ? 'guest' : '' }}</td>
          <td v-if="clientRunner.can('kick')">
            <ActBtn :runner="clientRunner" action="kick" :target="c.mac" :label="`kick ${clientName(c)}`" />
            <ActBtn :runner="clientRunner" action="block" :target="c.mac" :label="`block ${clientName(c)}`" />
          </td>
        </tr>
      </tbody>
    </table>
    <ActionPrompt :runner="clientRunner" />
  </TermBlock>

  <TermBlock :id="k('blocked')" title="unifi blocked" :meta="rowsMeta(blocked.length, metrics.blocked_clients.length)">
    <div v-if="!metrics.blocked_clients.length" class="dim">(none)</div>
    <table v-else class="tbl">
      <thead><tr><th>NAME</th><th>MAC</th><th v-if="blockedRunner.can('unblock')" /></tr></thead>
      <tbody>
        <tr v-for="c in blocked" :key="c.mac">
          <td class="max-w-[24ch] truncate">{{ clientName(c) }}</td>
          <td class="dim">{{ c.mac }}</td>
          <td v-if="blockedRunner.can('unblock')">
            <ActBtn :runner="blockedRunner" action="unblock" :target="c.mac" :label="`unblock ${clientName(c)}`" />
          </td>
        </tr>
      </tbody>
    </table>
    <ActionPrompt :runner="blockedRunner" />
  </TermBlock>
</template>
