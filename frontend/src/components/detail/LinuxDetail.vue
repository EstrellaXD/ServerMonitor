<script setup lang="ts">
import { computed } from 'vue'
import type { LinuxMetrics } from '@/types/metrics'
import MetricCard from '@/components/common/MetricCard.vue'
import ProgressBar from '@/components/charts/ProgressBar.vue'

const props = defineProps<{
  metrics: LinuxMetrics
}>()

const formatUptime = (seconds: number): string => {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)

  if (days > 0) return `${days}d ${hours}h ${minutes}m`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

const formatBytes = (bytes: number): string => {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(1)} ${units[i]}`
}

const formatSpeed = (bytesPerSec: number): string => {
  if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(0)} B/s`
  if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`
  return `${(bytesPerSec / 1024 / 1024).toFixed(1)} MB/s`
}

const uptime = computed(() => formatUptime(props.metrics.uptime))
</script>

<template>
  <div class="space-y-6">
    <!-- Overview Cards -->
    <div class="grid grid-cols-2 lg:grid-cols-4 gap-4">
      <div class="card p-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="icon-box icon-box-primary">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 text-primary-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <rect x="4" y="4" width="16" height="16" rx="2" ry="2"></rect>
              <rect x="9" y="9" width="6" height="6"></rect>
              <line x1="9" y1="1" x2="9" y2="4"></line>
              <line x1="15" y1="1" x2="15" y2="4"></line>
              <line x1="9" y1="20" x2="9" y2="23"></line>
              <line x1="15" y1="20" x2="15" y2="23"></line>
              <line x1="20" y1="9" x2="23" y2="9"></line>
              <line x1="20" y1="14" x2="23" y2="14"></line>
              <line x1="1" y1="9" x2="4" y2="9"></line>
              <line x1="1" y1="14" x2="4" y2="14"></line>
            </svg>
          </div>
          <div class="metric-label">CPU Usage</div>
        </div>
        <div class="metric-value-lg">{{ metrics.cpu.percent.toFixed(1) }}%</div>
        <ProgressBar :value="metrics.cpu.percent" :max="100" size="sm" class="mt-3" />
      </div>

      <div class="card p-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="icon-box icon-box-success">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M6 19v-3M10 19v-3M14 19v-3M18 19v-3M6 15v-3M10 15v-3M14 15v-3M18 15v-3M6 9V6M10 9V6M14 9V6M18 9V6"></path>
            </svg>
          </div>
          <div class="metric-label">Memory</div>
        </div>
        <div class="metric-value-lg">{{ metrics.memory.percent.toFixed(1) }}%</div>
        <ProgressBar :value="metrics.memory.percent" :max="100" size="sm" class="mt-3" />
      </div>

      <div class="card p-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="icon-box bg-violet-500/10 dark:bg-violet-500/20">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 text-violet-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10"></circle>
              <polyline points="12 6 12 12 16 14"></polyline>
            </svg>
          </div>
          <div class="metric-label">Uptime</div>
        </div>
        <div class="metric-value-lg font-mono">{{ uptime }}</div>
      </div>

      <div class="card p-5">
        <div class="flex items-center gap-3 mb-3">
          <div class="icon-box bg-amber-500/10 dark:bg-amber-500/20">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 text-amber-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M22 12h-4l-3 9L9 3l-3 9H2"></path>
            </svg>
          </div>
          <div class="metric-label">Load Average</div>
        </div>
        <div class="metric-value-lg font-mono">
          {{ metrics.cpu.load_avg.length > 0 ? metrics.cpu.load_avg[0].toFixed(2) : '0.00' }}
        </div>
        <div class="text-xs text-slate-500 dark:text-slate-400 mt-1">
          {{ metrics.cpu.load_avg.slice(0, 3).map(l => l.toFixed(2)).join(' / ') }}
        </div>
      </div>
    </div>

    <!-- CPU Cores -->
    <div v-if="metrics.cpu.cores.length > 0" class="card p-6">
      <h3 class="section-title mb-4">CPU Cores</h3>
      <div class="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-8 gap-4">
        <div v-for="(percent, index) in metrics.cpu.cores" :key="index" class="text-center">
          <div class="relative w-12 h-12 mx-auto mb-2">
            <svg class="w-12 h-12 transform -rotate-90">
              <circle
                cx="24" cy="24" r="20"
                stroke="currentColor"
                stroke-width="4"
                fill="none"
                class="text-slate-200 dark:text-surface-800"
              />
              <circle
                cx="24" cy="24" r="20"
                stroke="currentColor"
                stroke-width="4"
                fill="none"
                :stroke-dasharray="`${percent * 1.256} 999`"
                class="text-primary-500 transition-all duration-500"
              />
            </svg>
            <span class="absolute inset-0 flex items-center justify-center text-xs font-medium text-slate-700 dark:text-slate-300">
              {{ percent.toFixed(0) }}%
            </span>
          </div>
          <div class="text-xs text-slate-500 dark:text-slate-400">Core {{ index }}</div>
        </div>
      </div>
    </div>

    <!-- Memory -->
    <div class="card p-6">
      <h3 class="section-title mb-4">Memory</h3>
      <ProgressBar :value="metrics.memory.percent" :max="100" size="lg" variant="gradient" class="mb-4" />
      <div class="grid grid-cols-3 gap-4">
        <div class="text-center p-3 rounded-xl bg-slate-50 dark:bg-white/5">
          <div class="text-lg font-heading font-semibold text-slate-900 dark:text-white">{{ formatBytes(metrics.memory.used) }}</div>
          <div class="text-sm text-slate-500 dark:text-slate-400">Used</div>
        </div>
        <div class="text-center p-3 rounded-xl bg-slate-50 dark:bg-white/5">
          <div class="text-lg font-heading font-semibold text-slate-900 dark:text-white">{{ formatBytes(metrics.memory.available) }}</div>
          <div class="text-sm text-slate-500 dark:text-slate-400">Available</div>
        </div>
        <div class="text-center p-3 rounded-xl bg-slate-50 dark:bg-white/5">
          <div class="text-lg font-heading font-semibold text-slate-900 dark:text-white">{{ formatBytes(metrics.memory.total) }}</div>
          <div class="text-sm text-slate-500 dark:text-slate-400">Total</div>
        </div>
      </div>
    </div>

    <!-- Disks -->
    <div v-if="metrics.disks.length > 0" class="card p-6">
      <h3 class="section-title mb-4">Disk Usage</h3>
      <div class="space-y-4">
        <div v-for="disk in metrics.disks" :key="disk.mount" class="p-4 rounded-xl bg-slate-50 dark:bg-white/5">
          <div class="flex justify-between items-center mb-2">
            <div class="flex items-center gap-2">
              <svg xmlns="http://www.w3.org/2000/svg" class="w-4 h-4 text-slate-400" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
              </svg>
              <span class="font-medium text-slate-900 dark:text-white font-mono text-sm">{{ disk.mount }}</span>
            </div>
            <span class="text-sm text-slate-500 dark:text-slate-400">
              {{ formatBytes(disk.used) }} / {{ formatBytes(disk.total) }}
            </span>
          </div>
          <ProgressBar :value="disk.percent" :max="100" />
        </div>
      </div>
    </div>

    <!-- Network -->
    <div class="card p-6">
      <h3 class="section-title mb-4">Network</h3>
      <div class="grid grid-cols-2 gap-6">
        <div class="p-4 rounded-xl bg-emerald-50 dark:bg-emerald-500/10 border border-emerald-500/20">
          <div class="flex items-center gap-2 mb-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 text-emerald-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="12" y1="19" x2="12" y2="5"></line>
              <polyline points="5 12 12 5 19 12"></polyline>
            </svg>
            <span class="text-sm font-medium text-emerald-700 dark:text-emerald-400">Download</span>
          </div>
          <div class="text-2xl font-heading font-bold text-emerald-600 dark:text-emerald-400">
            {{ formatSpeed(metrics.network.download_rate) }}
          </div>
          <div class="text-sm text-emerald-600/70 dark:text-emerald-400/70 mt-1">
            Total: {{ formatBytes(metrics.network.bytes_recv) }}
          </div>
        </div>

        <div class="p-4 rounded-xl bg-blue-50 dark:bg-blue-500/10 border border-blue-500/20">
          <div class="flex items-center gap-2 mb-2">
            <svg xmlns="http://www.w3.org/2000/svg" class="w-5 h-5 text-blue-500" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <line x1="12" y1="5" x2="12" y2="19"></line>
              <polyline points="19 12 12 19 5 12"></polyline>
            </svg>
            <span class="text-sm font-medium text-blue-700 dark:text-blue-400">Upload</span>
          </div>
          <div class="text-2xl font-heading font-bold text-blue-600 dark:text-blue-400">
            {{ formatSpeed(metrics.network.upload_rate) }}
          </div>
          <div class="text-sm text-blue-600/70 dark:text-blue-400/70 mt-1">
            Total: {{ formatBytes(metrics.network.bytes_sent) }}
          </div>
        </div>
      </div>
    </div>

    <!-- Temperatures -->
    <div v-if="metrics.temperatures.length > 0" class="card p-6">
      <h3 class="section-title mb-4">Temperatures</h3>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
        <div
          v-for="temp in metrics.temperatures"
          :key="temp.label"
          class="p-4 rounded-xl text-center"
          :class="{
            'bg-emerald-50 dark:bg-emerald-500/10': temp.current < 60,
            'bg-amber-50 dark:bg-amber-500/10': temp.current >= 60 && temp.current < 80,
            'bg-red-50 dark:bg-red-500/10': temp.current >= 80
          }"
        >
          <div
            class="text-3xl font-heading font-bold"
            :class="{
              'text-emerald-600 dark:text-emerald-400': temp.current < 60,
              'text-amber-600 dark:text-amber-400': temp.current >= 60 && temp.current < 80,
              'text-red-600 dark:text-red-400': temp.current >= 80
            }"
          >
            {{ temp.current.toFixed(0) }}°
          </div>
          <div class="text-sm text-slate-600 dark:text-slate-400 mt-1">{{ temp.label }}</div>
        </div>
      </div>
    </div>
  </div>
</template>
