<script setup lang="ts">
import type { SystemStatus } from '@/types/metrics'
import { STATUS_CLASS, STATUS_GLYPH } from '@/lib/systems'

defineProps<{ status: SystemStatus; label?: boolean }>()
</script>

<template>
  <span :class="STATUS_CLASS[status]" :title="status">{{ STATUS_GLYPH[status] }}{{ label ? ` ${status}` : '' }}</span>
</template>
