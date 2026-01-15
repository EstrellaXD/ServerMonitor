<script setup lang="ts">
import { computed } from 'vue'
import type { SystemMetrics, LinuxMetrics, DockerMetrics, QBittorrentMetrics, UnifiMetrics, UNASMetrics } from '@/types/metrics'
import StatusIndicator from './StatusIndicator.vue'
import ProgressBar from '@/components/charts/ProgressBar.vue'

const props = defineProps<{
  system: SystemMetrics
}>()

const typeConfig: Record<string, { icon: string; label: string; gradient: string }> = {
  linux: {
    icon: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 3c1.66 0 3 .67 3 1.5S13.66 8 12 8s-3-.67-3-1.5S10.34 5 12 5zm-4 9c-.55 0-1-.45-1-1s.45-1 1-1 1 .45 1 1-.45 1-1 1zm8 0c-.55 0-1-.45-1-1s.45-1 1-1 1 .45 1 1-.45 1-1 1zm-4 3c-2.33 0-4.32-1.45-5.12-3.5h1.67c.69 1.19 1.97 2 3.45 2s2.76-.81 3.45-2h1.67c-.8 2.05-2.79 3.5-5.12 3.5z',
    label: 'Linux Server',
    gradient: 'from-orange-500 to-amber-500',
  },
  docker: {
    icon: 'M21.81 10.25c-.06-.04-.56-.43-1.64-.43-.28 0-.56.03-.84.08-.21-1.4-1.38-2.11-1.43-2.14l-.29-.17-.18.27c-.24.36-.43.77-.51 1.19-.2.8-.08 1.56.33 2.21-.49.28-1.29.35-1.46.35H2.62c-.34 0-.62.28-.62.63 0 1.04.16 2.08.49 3.07.38 1.14.97 1.99 1.75 2.52.93.64 2.45.99 4.17.99.73 0 1.5-.07 2.27-.22.96-.18 1.88-.49 2.74-.93.72-.36 1.38-.82 1.96-1.37.98-.93 1.56-2.03 1.98-2.98h.17c1.07 0 1.74-.43 2.1-.79.25-.23.45-.53.57-.87l.07-.27-.18-.13zM3.48 11h1.95v1.95H3.48V11zm2.59 0h1.95v1.95H6.07V11zm0-2.59h1.95v1.95H6.07V8.41zm2.59 2.59h1.95v1.95H8.66V11zm0-2.59h1.95v1.95H8.66V8.41zm2.59 2.59h1.95v1.95h-1.95V11zm0-2.59h1.95v1.95h-1.95V8.41zm2.6 2.59h1.95v1.95h-1.95V11zm0-2.59h1.95v1.95h-1.95V8.41z',
    label: 'Docker',
    gradient: 'from-sky-500 to-blue-500',
  },
  qbittorrent: {
    icon: 'M12 2L2 7v10l10 5 10-5V7L12 2zm0 2.33l6.67 3.34L12 11 5.33 7.67 12 4.33zM4 8.67l7 3.5v6.16l-7-3.5V8.67zm16 6.16l-7 3.5v-6.16l7-3.5v6.16z',
    label: 'qBittorrent',
    gradient: 'from-emerald-500 to-teal-500',
  },
  unifi: {
    icon: 'M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.42 0-8-3.58-8-8s3.58-8 8-8 8 3.58 8 8-3.58 8-8 8zm-1-13h2v6h-2zm0 8h2v2h-2z',
    label: 'UniFi',
    gradient: 'from-violet-500 to-purple-500',
  },
  unas: {
    icon: 'M2 4h20v4H2V4zm0 6h20v4H2v-4zm0 6h20v4H2v-4zm2-10v2h2V6H4zm0 6v2h2v-2H4zm0 6v2h2v-2H4z',
    label: 'Storage',
    gradient: 'from-pink-500 to-rose-500',
  },
}

const config = computed(() => typeConfig[props.system.type] || typeConfig.linux)

const primaryMetric = computed(() => {
  const metrics = props.system.metrics

  switch (props.system.type) {
    case 'linux': {
      const m = metrics as LinuxMetrics
      return { label: 'CPU', value: `${m.cpu?.percent?.toFixed(1) ?? 0}%` }
    }
    case 'docker': {
      const m = metrics as DockerMetrics
      return { label: 'Running', value: `${m.running_count ?? 0}/${m.total_count ?? 0}` }
    }
    case 'qbittorrent': {
      const m = metrics as QBittorrentMetrics
      return { label: 'Active', value: `${m.active_downloads ?? 0}` }
    }
    case 'unifi': {
      const m = metrics as UnifiMetrics
      return { label: 'Clients', value: `${m.clients_count ?? 0}` }
    }
    case 'unas': {
      const m = metrics as UNASMetrics
      const percent = m.total_capacity > 0 ? ((m.used_capacity / m.total_capacity) * 100).toFixed(1) : 0
      return { label: 'Used', value: `${percent}%` }
    }
    default:
      return { label: 'Status', value: props.system.status }
  }
})

const secondaryMetric = computed(() => {
  const metrics = props.system.metrics

  switch (props.system.type) {
    case 'linux': {
      // Hide RAM percentage since we show it as a progress bar
      return null
    }
    case 'docker': {
      const m = metrics as DockerMetrics
      return { label: 'Stopped', value: `${m.stopped_count ?? 0}` }
    }
    case 'qbittorrent': {
      const m = metrics as QBittorrentMetrics
      const speed = ((m.download_speed ?? 0) / 1024 / 1024).toFixed(1)
      return { label: 'Speed', value: `${speed} MB/s` }
    }
    case 'unifi': {
      const m = metrics as UnifiMetrics
      return { label: 'Devices', value: `${m.devices?.length ?? 0}` }
    }
    case 'unas': {
      const m = metrics as UNASMetrics
      return { label: 'Pools', value: `${m.pools?.length ?? 0}` }
    }
    default:
      return null
  }
})

const temperatureMetric = computed(() => {
  if (props.system.type !== 'linux') return null
  const m = props.system.metrics as LinuxMetrics
  if (!m.temperatures || m.temperatures.length === 0) return null
  const temp = m.temperatures[0]
  return {
    label: 'Temp',
    value: temp.current,
    isHot: temp.current >= 70
  }
})

const storagePercent = computed(() => {
  if (props.system.type !== 'unas') return null
  const m = props.system.metrics as UNASMetrics
  if (!m.total_capacity || m.total_capacity === 0) return null
  return (m.used_capacity / m.total_capacity) * 100
})

const memoryPercent = computed(() => {
  if (props.system.type !== 'linux') return null
  const m = props.system.metrics as LinuxMetrics
  if (!m.memory) return null
  return m.memory.percent
})
</script>

<template>
  <router-link
    :to="`/system/${system.id}`"
    class="card card-interactive group block p-5"
  >
    <!-- Header -->
    <div class="flex items-start justify-between mb-4">
      <div class="flex items-center gap-3">
        <!-- Icon with gradient -->
        <div
          class="w-11 h-11 rounded-xl bg-gradient-to-br flex items-center justify-center shadow-lg transition-transform duration-300 group-hover:scale-105"
          :class="config.gradient"
        >
          <svg class="w-5 h-5 text-white" viewBox="0 0 24 24" fill="currentColor">
            <path :d="config.icon" />
          </svg>
        </div>
        <div>
          <h3 class="font-heading font-semibold text-slate-900 dark:text-white group-hover:text-primary-600 dark:group-hover:text-primary-400 transition-colors">
            {{ system.name }}
          </h3>
          <p class="text-sm text-slate-500 dark:text-slate-400">{{ config.label }}</p>
        </div>
      </div>
      <StatusIndicator :status="system.status" />
    </div>

    <!-- Error State -->
    <div v-if="system.error" class="text-sm text-red-500 dark:text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-3 py-2">
      {{ system.error }}
    </div>

    <!-- Metrics -->
    <div v-else class="space-y-4">
      <div class="grid gap-4" :class="temperatureMetric ? 'grid-cols-3' : 'grid-cols-2'">
        <div class="space-y-1">
          <div class="metric-label">{{ primaryMetric.label }}</div>
          <div class="metric-value">{{ primaryMetric.value }}</div>
        </div>
        <div v-if="secondaryMetric" class="space-y-1">
          <div class="metric-label">{{ secondaryMetric.label }}</div>
          <div class="metric-value">{{ secondaryMetric.value }}</div>
        </div>
        <div v-if="temperatureMetric" class="space-y-1">
          <div class="metric-label">{{ temperatureMetric.label }}</div>
          <div class="metric-value" :class="temperatureMetric.isHot ? 'text-amber-500' : ''">
            {{ temperatureMetric.value.toFixed(0) }}°
          </div>
        </div>
      </div>

      <!-- Storage Progress Bar (UNAS only) -->
      <div v-if="storagePercent !== null">
        <ProgressBar :value="storagePercent" :max="100" size="lg" />
      </div>

      <!-- Memory Progress Bar (Linux only) -->
      <div v-if="memoryPercent !== null" class="space-y-1">
        <div class="metric-label">Memory</div>
        <ProgressBar :value="memoryPercent" :max="100" size="lg" />
      </div>
    </div>

    <!-- Hover indicator -->
    <div class="mt-2 flex items-center gap-1 text-sm text-slate-400 dark:text-slate-500 opacity-0 group-hover:opacity-100 transition-opacity">
      <span>View details</span>
      <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 transform group-hover:translate-x-1 transition-transform" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <line x1="5" y1="12" x2="19" y2="12"></line>
        <polyline points="12 5 19 12 12 19"></polyline>
      </svg>
    </div>
  </router-link>
</template>
