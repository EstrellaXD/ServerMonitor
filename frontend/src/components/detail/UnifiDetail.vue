<script setup lang="ts">
import type { UnifiMetrics } from '@/types/metrics'
import MetricCard from '@/components/common/MetricCard.vue'

defineProps<{
  metrics: UnifiMetrics
}>()

const formatUptime = (seconds: number): string => {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)

  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h`
  return '< 1h'
}
</script>

<template>
  <div class="space-y-6">
    <!-- Overview Cards -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <MetricCard label="Clients" :value="metrics.clients_count" />
      <MetricCard label="Guests" :value="metrics.guests_count" />
      <MetricCard label="WAN Download" :value="metrics.wan_download.toFixed(1)" unit="Mbps" />
      <MetricCard label="WAN Upload" :value="metrics.wan_upload.toFixed(1)" unit="Mbps" />
    </div>

    <!-- Devices List -->
    <div class="card overflow-hidden">
      <div class="p-5 border-b border-slate-200 dark:border-slate-700">
        <h3 class="text-lg font-medium text-slate-900 dark:text-white">Network Devices</h3>
      </div>

      <div v-if="metrics.devices.length === 0" class="p-8 text-center text-slate-500 dark:text-slate-400">
        No devices found
      </div>

      <div v-else class="divide-y divide-slate-200 dark:divide-slate-700">
        <div
          v-for="device in metrics.devices"
          :key="device.mac"
          class="p-4 hover:bg-slate-50 dark:hover:bg-slate-800/50"
        >
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-3">
              <div
                class="w-10 h-10 rounded-lg flex items-center justify-center"
                :class="device.status === 'online' ? 'bg-green-100 dark:bg-green-900/30' : 'bg-slate-100 dark:bg-slate-700'"
              >
                <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5" :class="device.status === 'online' ? 'text-green-500' : 'text-slate-400'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M5 12.55a11 11 0 0 1 14.08 0"></path>
                  <path d="M1.42 9a16 16 0 0 1 21.16 0"></path>
                  <path d="M8.53 16.11a6 6 0 0 1 6.95 0"></path>
                  <line x1="12" y1="20" x2="12.01" y2="20"></line>
                </svg>
              </div>
              <div>
                <div class="font-medium text-slate-900 dark:text-white">{{ device.name }}</div>
                <div class="text-sm text-slate-500 dark:text-slate-400">{{ device.model }}</div>
              </div>
            </div>
            <div class="text-right">
              <span
                class="px-2 py-1 text-xs font-medium rounded-full"
                :class="device.status === 'online'
                  ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400'
                  : 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400'"
              >
                {{ device.status }}
              </span>
              <div v-if="device.uptime > 0" class="text-sm text-slate-500 dark:text-slate-400 mt-1">
                Uptime: {{ formatUptime(device.uptime) }}
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
