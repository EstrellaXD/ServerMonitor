<script setup lang="ts">
import type { SystemMetrics } from '@/types/metrics'
import { useActions } from '@/composables/useActions'
import TermBlock from '@/components/term/TermBlock.vue'
import ActBtn from '@/components/term/ActBtn.vue'
import ActionPrompt from '@/components/term/ActionPrompt.vue'

const props = defineProps<{ system: SystemMetrics }>()
const runner = useActions(() => props.system)
</script>

<template>
  <TermBlock v-if="runner.can('reboot')" :id="`${system.id}:power`" title="power" :host="system.id">
    <div class="flex flex-wrap items-baseline gap-2">
      <ActBtn :runner="runner" action="reboot" target="" :label="`reboot ${system.id}`" />
      <ActBtn :runner="runner" action="poweroff" target="" :label="`poweroff ${system.id}`" />
      <span class="dim">requires typing the system id</span>
    </div>
    <ActionPrompt :runner="runner" />
  </TermBlock>
</template>
