<script setup lang="ts">
import { useMetricsStore } from '@/stores/metrics'
import SystemCard from './SystemCard.vue'
import StatusSummary from './StatusSummary.vue'

const metricsStore = useMetricsStore()
</script>

<template>
  <div>
    <StatusSummary class="mb-6" />

    <div v-if="metricsStore.systemsList.length === 0" class="card p-12 text-center">
      <svg xmlns="http://www.w3.org/2000/svg" class="w-16 h-16 mx-auto text-slate-300 dark:text-slate-600 mb-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <rect x="2" y="3" width="20" height="14" rx="2" ry="2"></rect>
        <line x1="8" y1="21" x2="16" y2="21"></line>
        <line x1="12" y1="17" x2="12" y2="21"></line>
      </svg>
      <h3 class="text-lg font-medium text-slate-900 dark:text-white mb-2">No systems configured</h3>
      <p class="text-slate-500 dark:text-slate-400 max-w-md mx-auto">
        Add systems to monitor in your <code class="bg-slate-100 dark:bg-slate-700 px-1 rounded">config.yaml</code> file and restart the backend.
      </p>
    </div>

    <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
      <SystemCard
        v-for="system in metricsStore.systemsList"
        :key="system.id"
        :system="system"
      />
    </div>
  </div>
</template>
