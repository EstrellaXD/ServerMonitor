<script setup lang="ts">
import { ref } from 'vue'
import type { QBittorrentMetrics, TorrentInfo } from '@/types/metrics'
import type { MenuItem } from '@/components/common/KebabMenu.vue'
import MetricCard from '@/components/common/MetricCard.vue'
import ProgressBar from '@/components/charts/ProgressBar.vue'
import KebabMenu from '@/components/common/KebabMenu.vue'
import ConfirmModal from '@/components/common/ConfirmModal.vue'
import { useActions, type QbitAction } from '@/composables/useActions'

const props = defineProps<{
  metrics: QBittorrentMetrics
  systemId: string
}>()

const { isLoading, error, qbitAction } = useActions(props.systemId)

const confirmModal = ref({
  show: false,
  title: '',
  message: '',
  confirmLabel: '',
  variant: 'danger' as 'warning' | 'danger',
  hash: '',
  action: '' as QbitAction,
})

const pausedStates = new Set(['pausedDL', 'pausedUP'])
const activeStates = new Set([
  'downloading', 'forcedDL', 'metaDL', 'stalledDL',
  'uploading', 'forcedUP', 'stalledUP', 'queuedDL', 'queuedUP',
])

const getMenuItems = (torrent: TorrentInfo): MenuItem[] => {
  const items: MenuItem[] = []

  if (pausedStates.has(torrent.state)) {
    items.push({ label: 'Resume', action: 'resume' })
  } else if (activeStates.has(torrent.state)) {
    items.push({ label: 'Pause', action: 'pause' })
  }

  items.push({ label: 'Delete', action: 'delete', variant: 'danger' })
  return items
}

const onAction = (torrent: TorrentInfo, action: string) => {
  if (action === 'resume' || action === 'pause') {
    qbitAction(torrent.hash, action as QbitAction)
    return
  }

  confirmModal.value = {
    show: true,
    title: 'Delete Torrent',
    message: `Remove "${torrent.name}" from the list?`,
    confirmLabel: 'Delete',
    variant: 'danger',
    hash: torrent.hash,
    action: 'delete',
  }
}

const onConfirm = (deleteFiles?: boolean) => {
  const { hash, action } = confirmModal.value
  confirmModal.value.show = false
  qbitAction(hash, action, deleteFiles ?? false)
}

const onCancel = () => {
  confirmModal.value.show = false
}

const formatSpeed = (bytesPerSec: number): string => {
  if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(0)} B/s`
  if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`
  return `${(bytesPerSec / 1024 / 1024).toFixed(2)} MB/s`
}

const formatSize = (bytes: number): string => {
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(1)} ${units[i]}`
}

const formatEta = (seconds: number | null | undefined): string => {
  if (!seconds) return '-'
  if (seconds > 86400) return `${Math.floor(seconds / 86400)}d`
  if (seconds > 3600) return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
  if (seconds > 60) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  return `${seconds}s`
}

const stateLabels: Record<string, string> = {
  downloading: 'Downloading',
  forcedDL: 'Downloading (F)',
  metaDL: 'Metadata',
  queuedDL: 'Queued',
  stalledDL: 'Stalled',
  uploading: 'Seeding',
  forcedUP: 'Seeding (F)',
  stalledUP: 'Seeding',
  queuedUP: 'Queued',
  pausedDL: 'Paused',
  pausedUP: 'Paused',
  error: 'Error',
  missingFiles: 'Missing',
  checkingUP: 'Checking',
  checkingDL: 'Checking',
}

const stateColors: Record<string, string> = {
  downloading: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  forcedDL: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400',
  uploading: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  forcedUP: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  stalledUP: 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400',
  pausedDL: 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400',
  pausedUP: 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400',
  error: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
}
</script>

<template>
  <div class="space-y-6">
    <!-- Overview Cards -->
    <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
      <MetricCard label="Download Speed" :value="formatSpeed(metrics.download_speed)" />
      <MetricCard label="Upload Speed" :value="formatSpeed(metrics.upload_speed)" />
      <MetricCard label="Active Downloads" :value="metrics.active_downloads" />
      <MetricCard label="Total Torrents" :value="metrics.total_torrents" />
    </div>

    <!-- Torrents List -->
    <div class="card overflow-hidden">
      <div class="p-5 border-b border-slate-200 dark:border-slate-700">
        <h3 class="text-lg font-medium text-slate-900 dark:text-white">Active Torrents</h3>
      </div>

      <div v-if="metrics.torrents.length === 0" class="p-8 text-center text-slate-500 dark:text-slate-400">
        No active torrents
      </div>

      <div v-else class="p-4 grid grid-cols-1 lg:grid-cols-2 gap-4 max-h-[calc(100vh-22rem)] overflow-y-auto">
        <div
          v-for="torrent in metrics.torrents"
          :key="torrent.hash"
          class="p-4 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50"
        >
          <div class="flex items-center justify-between mb-2">
            <div class="font-medium text-slate-900 dark:text-white truncate pr-2 flex-1 min-w-0">
              {{ torrent.name }}
            </div>
            <div class="flex items-center gap-2 flex-shrink-0">
              <span
                class="px-2 py-1 text-xs font-medium rounded-full whitespace-nowrap"
                :class="stateColors[torrent.state] || 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400'"
              >
                {{ stateLabels[torrent.state] || torrent.state }}
              </span>
              <KebabMenu
                :items="getMenuItems(torrent)"
                :loading="isLoading(torrent.hash)"
                @select="onAction(torrent, $event)"
              />
            </div>
          </div>

          <!-- Error indicator -->
          <div
            v-if="error?.targetId === torrent.hash"
            class="mb-2 px-2.5 py-1.5 rounded-lg bg-red-50 dark:bg-red-500/10 text-xs text-red-600 dark:text-red-400"
          >
            {{ error.message }}
          </div>

          <div class="mb-2">
            <ProgressBar :value="torrent.progress" :max="100" colorMode="progress" />
          </div>

          <div class="flex items-center justify-between text-sm text-slate-500 dark:text-slate-400">
            <span>{{ torrent.progress.toFixed(1) }}% of {{ formatSize(torrent.size) }}</span>
            <div class="flex items-center gap-3 flex-shrink-0">
              <span v-if="torrent.download_speed > 0" class="text-blue-500">
                ↓ {{ formatSpeed(torrent.download_speed) }}
              </span>
              <span v-if="torrent.upload_speed > 0" class="text-green-500">
                ↑ {{ formatSpeed(torrent.upload_speed) }}
              </span>
              <span v-if="torrent.eta">{{ formatEta(torrent.eta) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Confirm Modal -->
    <ConfirmModal
      :show="confirmModal.show"
      :title="confirmModal.title"
      :message="confirmModal.message"
      :confirm-label="confirmModal.confirmLabel"
      :variant="confirmModal.variant"
      show-delete-files
      @confirm="onConfirm"
      @cancel="onCancel"
    />
  </div>
</template>
