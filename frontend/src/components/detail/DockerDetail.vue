<script setup lang="ts">
import { ref } from 'vue'
import type { DockerMetrics, ContainerMetrics } from '@/types/metrics'
import type { MenuItem } from '@/components/common/KebabMenu.vue'
import MetricCard from '@/components/common/MetricCard.vue'
import ProgressBar from '@/components/charts/ProgressBar.vue'
import KebabMenu from '@/components/common/KebabMenu.vue'
import ConfirmModal from '@/components/common/ConfirmModal.vue'
import { useActions, type DockerAction } from '@/composables/useActions'

const props = defineProps<{
  metrics: DockerMetrics
  systemId: string
}>()

const { isLoading, error, dockerAction } = useActions(props.systemId)

const confirmModal = ref({
  show: false,
  title: '',
  message: '',
  confirmLabel: '',
  variant: 'warning' as 'warning' | 'danger',
  containerName: '',
  action: '' as DockerAction,
})

const getMenuItems = (container: ContainerMetrics): MenuItem[] => {
  if (container.state === 'running' || container.state === 'paused') {
    return [
      { label: 'Restart', action: 'restart' },
      { label: 'Stop', action: 'stop', variant: 'danger' },
    ]
  }
  return [
    { label: 'Start', action: 'start' },
  ]
}

const onAction = (containerName: string, action: string) => {
  if (action === 'start') {
    dockerAction(containerName, 'start')
    return
  }

  const isStop = action === 'stop'
  confirmModal.value = {
    show: true,
    title: isStop ? 'Stop Container' : 'Restart Container',
    message: isStop
      ? `Stop "${containerName}"? Running processes will be terminated.`
      : `Restart "${containerName}"? It will be briefly unavailable.`,
    confirmLabel: isStop ? 'Stop' : 'Restart',
    variant: isStop ? 'danger' : 'warning',
    containerName,
    action: action as DockerAction,
  }
}

const onConfirm = () => {
  const { containerName, action } = confirmModal.value
  confirmModal.value.show = false
  dockerAction(containerName, action)
}

const onCancel = () => {
  confirmModal.value.show = false
}

const formatBytes = (bytes: number): string => {
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(1)} ${units[i]}`
}
</script>

<template>
  <div class="space-y-6">
    <!-- Overview Cards -->
    <div class="grid grid-cols-3 gap-4">
      <MetricCard label="Total Containers" :value="metrics.total_count" />
      <MetricCard label="Running" :value="metrics.running_count" />
      <MetricCard label="Stopped" :value="metrics.stopped_count" />
    </div>

    <!-- Containers List -->
    <div class="card overflow-hidden">
      <div class="p-5 border-b border-slate-200 dark:border-slate-700">
        <h3 class="text-lg font-medium text-slate-900 dark:text-white">Containers</h3>
      </div>

      <div v-if="metrics.containers.length === 0" class="p-8 text-center text-slate-500 dark:text-slate-400">
        No containers found
      </div>

      <div v-else class="p-4 grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div
          v-for="container in metrics.containers"
          :key="container.id"
          class="p-4 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-800/50"
        >
          <div class="flex items-center justify-between mb-3">
            <div class="min-w-0 flex-1 pr-2">
              <div class="font-medium text-slate-900 dark:text-white truncate">{{ container.name }}</div>
              <div class="text-sm text-slate-500 dark:text-slate-400 truncate">{{ container.image }}</div>
            </div>
            <div class="flex items-center gap-2 flex-shrink-0">
              <span
                class="px-2 py-1 text-xs font-medium rounded-full"
                :class="{
                  'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400': container.state === 'running',
                  'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400': container.state === 'paused',
                  'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400': container.state === 'exited',
                  'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400': container.state === 'dead',
                }"
              >
                {{ container.state }}
              </span>
              <KebabMenu
                :items="getMenuItems(container)"
                :loading="isLoading(container.name)"
                @select="onAction(container.name, $event)"
              />
            </div>
          </div>

          <!-- Error indicator -->
          <div
            v-if="error?.targetId === container.name"
            class="mb-2 px-2.5 py-1.5 rounded-lg bg-red-50 dark:bg-red-500/10 text-xs text-red-600 dark:text-red-400"
          >
            {{ error.message }}
          </div>

          <div v-if="container.state === 'running'" class="space-y-2">
            <div>
              <div class="flex justify-between text-sm mb-1">
                <span class="text-slate-500 dark:text-slate-400">CPU</span>
                <span class="text-slate-900 dark:text-white">{{ container.cpu_percent.toFixed(1) }}%</span>
              </div>
              <ProgressBar :value="container.cpu_percent" :max="100" />
            </div>
            <div>
              <div class="flex justify-between text-sm mb-1">
                <span class="text-slate-500 dark:text-slate-400">Memory</span>
                <span class="text-slate-900 dark:text-white">
                  {{ formatBytes(container.memory_usage) }} / {{ formatBytes(container.memory_limit) }}
                </span>
              </div>
              <ProgressBar :value="container.memory_percent" :max="100" />
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
      @confirm="onConfirm"
      @cancel="onCancel"
    />
  </div>
</template>
