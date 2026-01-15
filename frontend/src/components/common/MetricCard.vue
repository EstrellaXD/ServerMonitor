<script setup lang="ts">
defineProps<{
  label: string
  value: string | number
  unit?: string
  icon?: string
}>()
</script>

<template>
  <div class="bg-slate-50 dark:bg-white/5 rounded-lg p-4">
    <div class="metric-label mb-1">{{ label }}</div>
    <div class="flex items-baseline gap-1">
      <span class="text-2xl font-heading font-semibold text-slate-900 dark:text-white tabular-nums tracking-tight">{{ value }}</span>
      <span v-if="unit" class="text-sm text-slate-500 dark:text-slate-400">{{ unit }}</span>
    </div>
  </div>
</template>
