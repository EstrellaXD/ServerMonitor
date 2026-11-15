<script setup lang="ts">
import { actionLog } from '@/composables/useActions'
import { formatClock, formatMs } from '@/composables/useFormatters'
import TermBlock from './TermBlock.vue'
</script>

<template>
  <TermBlock id="action-log" title="history" :meta="`${actionLog.length} action${actionLog.length === 1 ? '' : 's'} this session`">
    <div v-if="!actionLog.length" class="dim">(no actions yet)</div>
    <table v-else class="tbl">
      <tbody>
        <tr v-for="e in actionLog" :key="e.id">
          <td class="dim">{{ formatClock(e.at) }}</td>
          <td class="dim">@{{ e.systemId }}</td>
          <td>{{ e.label }}</td>
          <td v-if="e.ok" class="ok">done {{ formatMs(e.ms) }}</td>
          <td v-else class="crit">{{ e.message }}</td>
        </tr>
      </tbody>
    </table>
  </TermBlock>
</template>
