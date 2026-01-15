import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { SystemMetrics } from '@/types/metrics'

export const useMetricsStore = defineStore('metrics', () => {
  const systems = ref<Map<string, SystemMetrics>>(new Map())
  const lastUpdate = ref<Date | null>(null)
  const connected = ref(false)

  const systemsList = computed(() => Array.from(systems.value.values()))

  const systemsCount = computed(() => systems.value.size)

  const healthyCount = computed(() =>
    systemsList.value.filter(s => s.status === 'healthy').length
  )

  const warningCount = computed(() =>
    systemsList.value.filter(s => s.status === 'warning').length
  )

  const criticalCount = computed(() =>
    systemsList.value.filter(s => s.status === 'critical').length
  )

  const offlineCount = computed(() =>
    systemsList.value.filter(s => s.status === 'offline').length
  )

  function updateSystems(newSystems: Record<string, SystemMetrics>) {
    for (const [id, metrics] of Object.entries(newSystems)) {
      systems.value.set(id, metrics)
    }
  }

  function getSystem(id: string): SystemMetrics | undefined {
    return systems.value.get(id)
  }

  function setLastUpdate(date: Date) {
    lastUpdate.value = date
  }

  function setConnected(value: boolean) {
    connected.value = value
  }

  function clearSystems() {
    systems.value.clear()
    lastUpdate.value = null
  }

  return {
    systems,
    lastUpdate,
    connected,
    systemsList,
    systemsCount,
    healthyCount,
    warningCount,
    criticalCount,
    offlineCount,
    updateSystems,
    getSystem,
    setLastUpdate,
    setConnected,
    clearSystems,
  }
})
