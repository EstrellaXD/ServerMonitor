<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useMetricsStore } from '@/stores/metrics'
import { formatClock } from '@/composables/useFormatters'
import { TYPE_LABEL, dockerOf, linuxOf, qbitOf, unasOf, unifiOf } from '@/lib/systems'
import TermBlock from '@/components/term/TermBlock.vue'
import StatusGlyph from '@/components/term/StatusGlyph.vue'
import ActionLogBlock from '@/components/term/ActionLogBlock.vue'
import LinuxDetail from '@/components/detail/LinuxDetail.vue'
import DockerDetail from '@/components/detail/DockerDetail.vue'
import QbitDetail from '@/components/detail/QbitDetail.vue'
import UnifiDetail from '@/components/detail/UnifiDetail.vue'
import UnasDetail from '@/components/detail/UnasDetail.vue'

const route = useRoute()
const store = useMetricsStore()

const id = computed(() => String(route.params.id))
const system = computed(() => store.getSystem(id.value))
</script>

<template>
  <TermBlock :id="`${id}:info`" :title="`inspect ${id}`" :collapsible="false" :tone="system?.status" :meta="system ? formatClock(system.last_updated) : ''">
    <template v-if="system">
      <div class="flex flex-wrap gap-x-4">
        <StatusGlyph :status="system.status" label />
        <span>{{ system.name }}</span>
        <span class="dim">{{ TYPE_LABEL[system.type] }}</span>
      </div>
      <div class="dim break-words whitespace-normal">controls: {{ system.controls.length ? system.controls.join(', ') : 'none (read-only)' }}</div>
      <div v-if="system.error" class="crit break-words whitespace-normal">✖ {{ system.error }}</div>
    </template>
    <div v-else-if="store.systemsCount" class="crit">✖ no such system: {{ id }}</div>
    <div v-else class="dim">waiting for first update…</div>
  </TermBlock>

  <template v-if="system">
    <LinuxDetail v-if="linuxOf(system)" :key="system.id" :system="system" :metrics="linuxOf(system)!" />
    <DockerDetail v-else-if="dockerOf(system)" :key="system.id" :system="system" :metrics="dockerOf(system)!" />
    <QbitDetail v-else-if="qbitOf(system)" :key="system.id" :system="system" :metrics="qbitOf(system)!" />
    <UnifiDetail v-else-if="unifiOf(system)" :key="system.id" :system="system" :metrics="unifiOf(system)!" />
    <UnasDetail v-else-if="unasOf(system)" :key="system.id" :system="system" :metrics="unasOf(system)!" />
  </template>

  <ActionLogBlock />
</template>
