<script setup lang="ts">
import { useMetricsStore } from '@/stores/metrics'
import OverviewGrid from '@/components/dashboard/OverviewGrid.vue'

const metricsStore = useMetricsStore()
</script>

<template>
  <div>
    <OverviewGrid />
  </div>
</template>
