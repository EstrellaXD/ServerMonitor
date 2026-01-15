<script setup lang="ts">
import { useMetricsStore } from '@/stores/metrics'
import OverviewGrid from '@/components/dashboard/OverviewGrid.vue'

const metricsStore = useMetricsStore()
</script>

<template>
  <div>
    <!-- Header -->
    <div class="mb-8">
      <h1 class="text-3xl font-heading font-bold text-slate-900 dark:text-white mb-2">
        Dashboard
      </h1>
      <p class="text-slate-500 dark:text-slate-400">
        Monitoring <span class="font-medium text-slate-700 dark:text-slate-300">{{ metricsStore.systemsCount }}</span> systems in real-time
      </p>
    </div>

    <OverviewGrid />
  </div>
</template>
