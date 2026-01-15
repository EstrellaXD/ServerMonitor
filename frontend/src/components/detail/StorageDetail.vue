<script setup lang="ts">
import { computed } from 'vue'
import type { UNASMetrics } from '@/types/metrics'
import MetricCard from '@/components/common/MetricCard.vue'

const props = defineProps<{
  metrics: UNASMetrics
}>()

const formatBytes = (bytes: number): string => {
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(1)} ${units[i]}`
}

const totalUsedPercent = computed(() => {
  if (props.metrics.total_capacity === 0) return 0
  return (props.metrics.used_capacity / props.metrics.total_capacity) * 100
})
</script>

<template>
  <div class="space-y-6">
    <!-- Overview Cards -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <MetricCard label="Total Capacity" :value="formatBytes(metrics.total_capacity)" />
      <MetricCard label="Used" :value="formatBytes(metrics.used_capacity)" />
      <MetricCard label="Used %" :value="totalUsedPercent.toFixed(1)" unit="%" />
      <MetricCard label="Pools" :value="metrics.pools.length" />
    </div>

    <!-- Storage Pools -->
    <div class="card p-4 sm:p-5">
      <h3 class="text-lg font-medium text-slate-900 dark:text-white mb-4">Storage Pools</h3>

      <div v-if="metrics.pools.length === 0" class="text-center text-slate-500 dark:text-slate-400 py-4">
        No storage pools found
      </div>

      <div v-else class="space-y-4">
        <div v-for="pool in metrics.pools" :key="pool.name" class="p-3 sm:p-4 bg-slate-50 dark:bg-slate-700/50 rounded-lg">
          <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2 sm:gap-4">
            <div class="flex items-center gap-2 sm:gap-3">
              <span class="font-medium text-slate-900 dark:text-white">{{ pool.name }}</span>
              <span
                class="px-2 py-0.5 text-xs font-medium rounded-full"
                :class="{
                  'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400': pool.status === 'online',
                  'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400': pool.status === 'degraded',
                  'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400': pool.status === 'faulted',
                  'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400': pool.status === 'offline',
                }"
              >
                {{ pool.status }}
              </span>
            </div>
            <div class="flex items-center justify-between sm:justify-end gap-4">
              <span class="text-sm text-slate-500 dark:text-slate-400">
                {{ formatBytes(pool.used) }} / {{ formatBytes(pool.total) }}
              </span>
              <span class="text-xl sm:text-2xl font-semibold text-slate-900 dark:text-white tabular-nums">
                {{ pool.percent.toFixed(1) }}%
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Disk Health -->
    <div v-if="metrics.disks.length > 0" class="card overflow-hidden">
      <div class="p-4 sm:p-5 border-b border-slate-200 dark:border-slate-700">
        <h3 class="text-lg font-medium text-slate-900 dark:text-white">Disk Health</h3>
      </div>

      <div class="p-4 grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div
          v-for="disk in metrics.disks"
          :key="disk.name"
          class="p-3 sm:p-4 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50"
        >
          <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
            <div class="min-w-0">
              <div class="font-medium text-slate-900 dark:text-white truncate">{{ disk.name }}</div>
              <div class="text-sm text-slate-500 dark:text-slate-400 truncate">
                {{ disk.model }} ({{ disk.serial }})
              </div>
            </div>
            <div class="flex items-center gap-3 flex-shrink-0">
              <span
                v-if="disk.temperature"
                class="text-sm font-medium"
                :class="{
                  'text-green-500': disk.temperature < 40,
                  'text-amber-500': disk.temperature >= 40 && disk.temperature < 50,
                  'text-red-500': disk.temperature >= 50,
                }"
              >
                {{ disk.temperature }}°C
              </span>
              <span
                class="px-2 py-1 text-xs font-medium rounded-full"
                :class="disk.status === 'healthy'
                  ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                  : 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'"
              >
                {{ disk.status }}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
