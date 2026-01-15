<script setup lang="ts">
import { useMetricsStore } from '@/stores/metrics'

const metricsStore = useMetricsStore()

const summaryItems = [
  {
    key: 'healthy',
    label: 'Healthy',
    icon: 'M22 11.08V12a10 10 0 1 1-5.93-9.14M22 4L12 14.01l-3-3',
    iconClass: 'text-emerald-500',
    bgClass: 'bg-emerald-500/10 dark:bg-emerald-500/20',
    getValue: () => metricsStore.healthyCount,
  },
  {
    key: 'warning',
    label: 'Warning',
    icon: 'M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0zM12 9v4M12 17h.01',
    iconClass: 'text-amber-500',
    bgClass: 'bg-amber-500/10 dark:bg-amber-500/20',
    getValue: () => metricsStore.warningCount,
  },
  {
    key: 'critical',
    label: 'Critical',
    icon: 'M12 2a10 10 0 1 0 0 20 10 10 0 0 0 0-20zM15 9l-6 6M9 9l6 6',
    iconClass: 'text-red-500',
    bgClass: 'bg-red-500/10 dark:bg-red-500/20',
    getValue: () => metricsStore.criticalCount,
  },
  {
    key: 'offline',
    label: 'Offline',
    icon: 'M1 1l22 22M16.72 11.06A10.94 10.94 0 0 1 19 12.55M5 12.55a10.94 10.94 0 0 1 5.17-2.39M10.71 5.05A16 16 0 0 1 22.58 9M1.42 9a15.91 15.91 0 0 1 4.7-2.88M8.53 16.11a6 6 0 0 1 6.95 0M12 20h.01',
    iconClass: 'text-slate-500',
    bgClass: 'bg-slate-500/10 dark:bg-slate-500/20',
    getValue: () => metricsStore.offlineCount,
  },
]
</script>

<template>
  <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
    <div
      v-for="item in summaryItems"
      :key="item.key"
      class="card p-4 flex items-center gap-4"
    >
      <div class="icon-box" :class="item.bgClass">
        <svg
          xmlns="http://www.w3.org/2000/svg"
          class="w-5 h-5"
          :class="item.iconClass"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <path :d="item.icon" />
        </svg>
      </div>
      <div>
        <div class="metric-value">{{ item.getValue() }}</div>
        <div class="metric-label">{{ item.label }}</div>
      </div>
    </div>
  </div>
</template>
