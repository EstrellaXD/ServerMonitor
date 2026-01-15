import { ref, onMounted, onUnmounted } from 'vue'
import { useMetricsStore } from '@/stores/metrics'
import type { MetricsUpdate, SystemMetrics } from '@/types/metrics'

export function useWebSocket() {
  const connected = ref(false)
  const reconnecting = ref(false)
  const metricsStore = useMetricsStore()

  let ws: WebSocket | null = null
  let reconnectTimeout: ReturnType<typeof setTimeout> | null = null
  let pingInterval: ReturnType<typeof setInterval> | null = null
  let pollInterval: ReturnType<typeof setInterval> | null = null

  const getWebSocketUrl = () => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = window.location.host
    return `${protocol}//${host}/ws`
  }

  // HTTP fallback for fetching data
  const fetchSystems = async () => {
    try {
      const response = await fetch('/api/systems')
      if (response.ok) {
        const systems: Record<string, SystemMetrics> = await response.json()
        metricsStore.updateSystems(systems)
        metricsStore.setLastUpdate(new Date())
        console.log('Fetched systems via HTTP:', Object.keys(systems).length)
      }
    } catch (e) {
      console.error('Failed to fetch systems:', e)
    }
  }

  // Start HTTP polling as fallback
  const startPolling = () => {
    if (pollInterval) return
    fetchSystems() // Fetch immediately
    pollInterval = setInterval(fetchSystems, 5000)
  }

  const stopPolling = () => {
    if (pollInterval) {
      clearInterval(pollInterval)
      pollInterval = null
    }
  }

  const connect = () => {
    if (ws?.readyState === WebSocket.OPEN) return

    try {
      ws = new WebSocket(getWebSocketUrl())

      ws.onopen = () => {
        connected.value = true
        reconnecting.value = false
        stopPolling() // Stop HTTP polling when WebSocket connects
        console.log('WebSocket connected')

        pingInterval = setInterval(() => {
          if (ws?.readyState === WebSocket.OPEN) {
            ws.send('ping')
          }
        }, 25000)
      }

      ws.onmessage = (event) => {
        if (event.data === 'ping' || event.data === 'pong') return

        try {
          const data: MetricsUpdate = JSON.parse(event.data)
          if (data.type === 'metrics_update') {
            metricsStore.updateSystems(data.systems)
            metricsStore.setLastUpdate(new Date(data.timestamp))
          }
        } catch (e) {
          console.error('Failed to parse WebSocket message:', e)
        }
      }

      ws.onclose = () => {
        connected.value = false
        cleanup()
        startPolling() // Fall back to HTTP polling
        scheduleReconnect()
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        startPolling() // Fall back to HTTP polling
        ws?.close()
      }
    } catch (error) {
      console.error('Failed to create WebSocket:', error)
      startPolling() // Fall back to HTTP polling
      scheduleReconnect()
    }
  }

  const cleanup = () => {
    if (pingInterval) {
      clearInterval(pingInterval)
      pingInterval = null
    }
  }

  const scheduleReconnect = () => {
    if (reconnectTimeout) return

    reconnecting.value = true
    reconnectTimeout = setTimeout(() => {
      reconnectTimeout = null
      connect()
    }, 3000)
  }

  const disconnect = () => {
    cleanup()
    stopPolling()
    if (reconnectTimeout) {
      clearTimeout(reconnectTimeout)
      reconnectTimeout = null
    }
    if (ws) {
      ws.close()
      ws = null
    }
  }

  onMounted(() => {
    // Fetch initial data immediately via HTTP
    fetchSystems()
    // Then try to connect WebSocket
    connect()
  })

  onUnmounted(() => {
    disconnect()
  })

  return {
    connected,
    reconnecting,
    connect,
    disconnect,
  }
}
