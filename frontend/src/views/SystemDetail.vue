<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useMetricsStore } from '@/stores/metrics'
import StatusIndicator from '@/components/dashboard/StatusIndicator.vue'
import LinuxDetail from '@/components/detail/LinuxDetail.vue'
import DockerDetail from '@/components/detail/DockerDetail.vue'
import QbitDetail from '@/components/detail/QbitDetail.vue'
import UnifiDetail from '@/components/detail/UnifiDetail.vue'
import StorageDetail from '@/components/detail/StorageDetail.vue'

const route = useRoute()
const router = useRouter()
const metricsStore = useMetricsStore()

const systemId = computed(() => route.params.id as string)
const system = computed(() => metricsStore.getSystem(systemId.value))

const detailComponent = computed(() => {
  switch (system.value?.type) {
    case 'linux': return LinuxDetail
    case 'docker': return DockerDetail
    case 'qbittorrent': return QbitDetail
    case 'unifi': return UnifiDetail
    case 'unas': return StorageDetail
    default: return null
  }
})

const typeLabels: Record<string, string> = {
  linux: 'Linux Server',
  docker: 'Docker Host',
  qbittorrent: 'qBittorrent',
  unifi: 'UniFi Network',
  unas: 'Storage System',
}

const goBack = () => {
  router.push('/')
}
</script>

<template>
  <div>
    <!-- Back button -->
    <button
      @click="goBack"
      class="inline-flex items-center gap-2 text-slate-500 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white mb-6 transition-colors group"
    >
      <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 transform group-hover:-translate-x-1 transition-transform" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="19" y1="12" x2="5" y2="12"></line>
        <polyline points="12 19 5 12 12 5"></polyline>
      </svg>
      <span class="font-medium">Back to Dashboard</span>
    </button>

    <!-- Not found state -->
    <div v-if="!system" class="card p-12 text-center">
      <div class="w-16 h-16 mx-auto mb-4 rounded-2xl bg-slate-100 dark:bg-white/5 flex items-center justify-center">
        <svg xmlns="http://www.w3.org/2000/svg" class="w-8 h-8 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"></circle>
          <line x1="12" y1="8" x2="12" y2="12"></line>
          <line x1="12" y1="16" x2="12.01" y2="16"></line>
        </svg>
      </div>
      <h3 class="text-xl font-heading font-semibold text-slate-900 dark:text-white mb-2">System not found</h3>
      <p class="text-slate-500 dark:text-slate-400 max-w-md mx-auto">
        The system "{{ systemId }}" was not found or is not being monitored.
      </p>
    </div>

    <!-- System detail -->
    <div v-else>
      <!-- Header card -->
      <div class="card p-4 sm:p-6 mb-6">
        <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <div class="flex flex-wrap items-center gap-2 sm:gap-3 mb-1">
              <h1 class="text-xl sm:text-2xl font-heading font-bold text-slate-900 dark:text-white">
                {{ system.name }}
              </h1>
              <StatusIndicator :status="system.status" show-label />
            </div>
            <p class="text-slate-500 dark:text-slate-400">
              {{ typeLabels[system.type] || system.type }}
            </p>
          </div>

          <!-- Last updated -->
          <div class="sm:text-right">
            <div class="text-sm text-slate-500 dark:text-slate-400">Last updated</div>
            <div class="text-sm font-mono text-slate-700 dark:text-slate-300">
              {{ new Date(system.last_updated).toLocaleTimeString() }}
            </div>
          </div>
        </div>
      </div>

      <!-- Error banner -->
      <div v-if="system.error" class="card p-4 mb-6 border-red-500/30 bg-red-50 dark:bg-red-500/10">
        <div class="flex items-center gap-3">
          <div class="icon-box icon-box-danger flex-shrink-0">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 text-red-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10"></circle>
              <line x1="15" y1="9" x2="9" y2="15"></line>
              <line x1="9" y1="9" x2="15" y2="15"></line>
            </svg>
          </div>
          <div>
            <div class="font-medium text-red-700 dark:text-red-300">Connection Error</div>
            <div class="text-sm text-red-600 dark:text-red-400">{{ system.error }}</div>
          </div>
        </div>
      </div>

      <!-- Detail component -->
      <component v-else :is="detailComponent" :metrics="system.metrics" :system-id="systemId" />
    </div>
  </div>
</template>
