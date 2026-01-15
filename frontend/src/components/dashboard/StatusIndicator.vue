<script setup lang="ts">
import type { SystemStatus } from '@/types/metrics'

defineProps<{
  status: SystemStatus
  showLabel?: boolean
  size?: 'sm' | 'md' | 'lg'
}>()

const statusConfig: Record<SystemStatus, { label: string; dotClass: string; badgeClass: string }> = {
  healthy: {
    label: 'Healthy',
    dotClass: 'status-healthy',
    badgeClass: 'status-badge-healthy',
  },
  warning: {
    label: 'Warning',
    dotClass: 'status-warning',
    badgeClass: 'status-badge-warning',
  },
  critical: {
    label: 'Critical',
    dotClass: 'status-critical',
    badgeClass: 'status-badge-critical',
  },
  offline: {
    label: 'Offline',
    dotClass: 'status-offline',
    badgeClass: 'status-badge-offline',
  },
}
</script>

<template>
  <!-- Badge style when showing label -->
  <span v-if="showLabel" class="status-badge" :class="statusConfig[status].badgeClass">
    <span class="status-dot !w-1.5 !h-1.5 !ring-0" :class="statusConfig[status].dotClass"></span>
    {{ statusConfig[status].label }}
  </span>

  <!-- Dot only -->
  <span v-else class="status-dot" :class="statusConfig[status].dotClass"></span>
</template>
