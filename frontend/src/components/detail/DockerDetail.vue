<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ContainerMetrics, DockerMetrics, SystemMetrics } from '@/types/metrics'
import { formatBytes } from '@/composables/useFormatters'
import { useActions } from '@/composables/useActions'
import { useFilter } from '@/composables/useFilter'
import TermBlock from '@/components/term/TermBlock.vue'
import ActBtn from '@/components/term/ActBtn.vue'
import ActionPrompt from '@/components/term/ActionPrompt.vue'
import LogsBlock from './LogsBlock.vue'

const props = defineProps<{ system: SystemMetrics; metrics: DockerMetrics }>()

const { matches } = useFilter()
const runner = useActions(() => props.system)
const openLogs = ref<string[]>([])

const containers = computed(() =>
  props.metrics.containers.filter(c => matches(c.name, c.image, c.state, c.status)),
)
const meta = computed(() => {
  const shown = containers.value.length
  const total = props.metrics.total_count
  return `${props.metrics.running_count}/${total} running${shown === props.metrics.containers.length ? '' : ` · ${shown} shown`}`
})

const stateClass = (state: string) =>
  state === 'running' ? 'ok' : state === 'paused' || state === 'restarting' ? 'warn' : state === 'dead' ? 'crit' : 'off'

const isOn = (c: ContainerMetrics) => c.state === 'running'

function toggleLogs(name: string) {
  const i = openLogs.value.indexOf(name)
  if (i === -1) openLogs.value.push(name)
  else openLogs.value.splice(i, 1)
}
</script>

<template>
  <TermBlock :id="`${system.id}:containers`" title="docker ps -a" :host="system.id" :meta="meta">
    <table class="tbl">
      <thead><tr><th>NAME</th><th>STATE</th><th class="r">CPU</th><th class="r">MEM</th><th /><th>IMAGE</th></tr></thead>
      <tbody>
        <tr v-for="c in containers" :key="c.id">
          <td class="max-w-[26ch] truncate" :title="c.status">{{ c.name }}</td>
          <td :class="stateClass(c.state)">{{ c.state }}</td>
          <td class="r">{{ isOn(c) ? `${c.cpu_percent.toFixed(1)}%` : '–' }}</td>
          <td class="r">{{ isOn(c) ? formatBytes(c.memory_usage) : '–' }}</td>
          <td>
            <template v-if="c.state === 'running'">
              <ActBtn :runner="runner" action="restart" :target="c.name" />
              <ActBtn :runner="runner" action="stop" :target="c.name" />
              <ActBtn :runner="runner" action="pause" :target="c.name" />
              <ActBtn :runner="runner" action="kill" :target="c.name" />
            </template>
            <template v-else-if="c.state === 'paused'">
              <ActBtn :runner="runner" action="unpause" :target="c.name" />
              <ActBtn :runner="runner" action="stop" :target="c.name" />
            </template>
            <ActBtn v-else :runner="runner" action="start" :target="c.name" />
            <ActBtn :runner="runner" action="recreate" :target="c.name" />
            <button type="button" class="btn ml-1" :class="{ on: openLogs.includes(c.name) }" @click="toggleLogs(c.name)">logs</button>
          </td>
          <td class="dim max-w-[36ch] truncate">{{ c.image }}</td>
        </tr>
      </tbody>
    </table>
    <ActionPrompt :runner="runner" />
  </TermBlock>

  <LogsBlock
    v-for="name in openLogs"
    :key="name"
    :system-id="system.id"
    :container="name"
    @close="toggleLogs(name)"
  />
</template>
