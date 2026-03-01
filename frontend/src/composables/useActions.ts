import { reactive, ref } from 'vue'

export type DockerAction = 'start' | 'stop' | 'restart'
export type QbitAction = 'resume' | 'pause' | 'delete'

interface ActionError {
  targetId: string
  message: string
}

export function useActions(systemId: string) {
  const loadingTargets = reactive(new Map<string, string>())
  const error = ref<ActionError | null>(null)
  let errorTimeout: ReturnType<typeof setTimeout> | null = null

  const clearError = () => {
    error.value = null
    if (errorTimeout) {
      clearTimeout(errorTimeout)
      errorTimeout = null
    }
  }

  const setError = (targetId: string, message: string) => {
    error.value = { targetId, message }
    errorTimeout = setTimeout(clearError, 3000)
  }

  const isLoading = (targetId: string) => loadingTargets.has(targetId)

  const dockerAction = async (containerName: string, action: DockerAction) => {
    loadingTargets.set(containerName, action)
    clearError()

    try {
      const response = await fetch(`/api/systems/${systemId}/actions/docker`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ container_name: containerName, action }),
      })

      if (!response.ok) {
        const data = await response.json().catch(() => ({}))
        throw new Error(data.error || `Failed to ${action} container`)
      }
    } catch (e) {
      setError(containerName, e instanceof Error ? e.message : `Failed to ${action}`)
    } finally {
      loadingTargets.delete(containerName)
    }
  }

  const qbitAction = async (hash: string, action: QbitAction, deleteFiles = false) => {
    loadingTargets.set(hash, action)
    clearError()

    try {
      const body: Record<string, unknown> = { hash, action }
      if (action === 'delete') {
        body.delete_files = deleteFiles
      }

      const response = await fetch(`/api/systems/${systemId}/actions/qbittorrent`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      })

      if (!response.ok) {
        const data = await response.json().catch(() => ({}))
        throw new Error(data.error || `Failed to ${action} torrent`)
      }
    } catch (e) {
      setError(hash, e instanceof Error ? e.message : `Failed to ${action}`)
    } finally {
      loadingTargets.delete(hash)
    }
  }

  return {
    loadingTargets,
    error,
    isLoading,
    dockerAction,
    qbitAction,
    clearError,
  }
}
