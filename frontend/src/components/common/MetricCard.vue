<script setup lang="ts">
defineProps<{
  label: string
  value: string | number
  unit?: string
  icon?: string
}>()
</script>

<template>
  <div class="bg-slate-50 dark:bg-slate-700/50 rounded-lg p-4">
    <div class="metric-label mb-1">{{ label }}</div>
    <div class="flex items-baseline gap-1">
      <span class="metric-value">{{ value }}</span>
      <span v-if="unit" class="text-sm text-slate-500 dark:text-slate-400">{{ unit }}</span>
    </div>
  </div>
</template>
