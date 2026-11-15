import { computed, reactive, ref } from 'vue'
import type { SystemMetrics, SystemType } from '@/types/metrics'
import type { ActionRequestBody, ActionResponse, ApiError, DockerLogs } from '@/types/api'

export type Risk = 'low' | 'mid' | 'high'
export type ActionParams = Record<string, unknown>

interface ActionSpec {
  control: string
  risk: Risk
}

const low = (control: string): ActionSpec => ({ control, risk: 'low' })
const mid = (control: string): ActionSpec => ({ control, risk: 'mid' })
const high = (control: string): ActionSpec => ({ control, risk: 'high' })

// Mirror of the contract action table: action -> required control and base risk.
const SPECS: Record<SystemType, Record<string, ActionSpec>> = {
  docker: {
    start: low('container'),
    stop: low('container'),
    restart: low('container'),
    pause: low('container'),
    unpause: low('container'),
    kill: mid('container'),
    recreate: mid('recreate'),
  },
  qbittorrent: {
    resume: low('torrent'),
    pause: low('torrent'),
    recheck: low('torrent'),
    top_prio: low('torrent'),
    bottom_prio: low('torrent'),
    increase_prio: low('torrent'),
    decrease_prio: low('torrent'),
    delete: mid('torrent'),
  },
  linux: {
    reboot: high('power'),
    poweroff: high('power'),
    kill: mid('process'),
    service_start: mid('service'),
    service_stop: mid('service'),
    service_restart: mid('service'),
    fan_set: mid('fan'),
    fan_auto: low('fan'),
  },
  unas: {
    reboot: high('power'),
    poweroff: high('power'),
  },
  unifi: {
    restart_device: mid('device'),
    kick: mid('client'),
    block: mid('client'),
    unblock: mid('client'),
  },
}

const isDeleteWithFiles = (type: SystemType, action: string, params?: ActionParams) =>
  type === 'qbittorrent' && action === 'delete' && params?.delete_files === true

export function canRun(system: SystemMetrics, action: string, params?: ActionParams): boolean {
  const spec = SPECS[system.type][action]
  if (!spec || !system.controls.includes(spec.control)) return false
  if (isDeleteWithFiles(system.type, action, params)) return system.controls.includes('delete_files')
  return true
}

export function riskOf(type: SystemType, action: string, params?: ActionParams): Risk {
  if (isDeleteWithFiles(type, action, params)) return 'high'
  if (type === 'linux' && action === 'kill' && params?.signal === 'KILL') return 'high'
  // 0% stops the fan while the kernel governor is disabled.
  if (type === 'linux' && action === 'fan_set' && params?.percent === 0) return 'high'
  return SPECS[type][action]?.risk ?? 'high'
}

// ---- session action log (client-side, all systems) ----

export interface ActionLogEntry {
  id: number
  at: Date
  systemId: string
  label: string
  ok: boolean
  message: string
  ms: number
}

const LOG_SIZE = 10
let nextLogId = 1
export const actionLog = reactive<ActionLogEntry[]>([])

function appendLog(entry: Omit<ActionLogEntry, 'id'>) {
  actionLog.unshift({ id: nextLogId++, ...entry })
  actionLog.splice(LOG_SIZE)
}

// ---- HTTP ----

async function errorText(res: Response): Promise<string> {
  const body = (await res.json().catch(() => null)) as ApiError | null
  return body?.error ?? `HTTP ${res.status}`
}

export async function postAction(systemId: string, body: ActionRequestBody): Promise<ActionResponse> {
  const res = await fetch(`/api/systems/${encodeURIComponent(systemId)}/actions`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
  if (!res.ok) throw new Error(await errorText(res))
  return res.json()
}

export async function fetchDockerLogs(systemId: string, container: string, tail = 200): Promise<DockerLogs> {
  const q = new URLSearchParams({ container, tail: String(tail) })
  const res = await fetch(`/api/systems/${encodeURIComponent(systemId)}/docker/logs?${q}`)
  if (!res.ok) throw new Error(await errorText(res))
  return res.json()
}

// ---- per-block action runner ----

export interface PendingAction {
  action: string
  target: string
  params?: ActionParams
  label: string
  risk: Risk
}

export interface ActionResult {
  ok: boolean
  label: string
  text: string
  ms: number
}

const RESULT_TTL_MS = 10_000

/**
 * One runner per block: it owns that block's confirm prompt and result line.
 * low risk runs at once, mid asks [y/N], high asks for the system id.
 */
export function useActions(system: () => SystemMetrics) {
  const pending = ref<PendingAction | null>(null)
  const running = reactive(new Set<string>())
  const result = ref<ActionResult | null>(null)
  let resultTimer: ReturnType<typeof setTimeout> | undefined

  const systemId = computed(() => system().id)
  const key = (action: string, target: string) => `${action}\u0000${target}`

  async function run(p: PendingAction) {
    const id = systemId.value
    const k = key(p.action, p.target)
    running.add(k)
    const t0 = performance.now()
    let ok = true
    let text: string
    try {
      const res = await postAction(id, { action: p.action, target: p.target, params: p.params })
      text = res.output?.trim() ?? ''
    } catch (e) {
      ok = false
      text = e instanceof Error ? e.message : String(e)
    }
    const ms = performance.now() - t0
    running.delete(k)
    clearTimeout(resultTimer)
    result.value = { ok, label: p.label, text, ms }
    resultTimer = setTimeout(() => { result.value = null }, RESULT_TTL_MS)
    appendLog({ at: new Date(), systemId: id, label: p.label, ok, message: text, ms })
  }

  function request(action: string, target: string, opts: { params?: ActionParams; label?: string } = {}) {
    const risk = riskOf(system().type, action, opts.params)
    const label = opts.label ?? `${action} ${target}`.trim()
    const p: PendingAction = { action, target, params: opts.params, risk, label }
    if (risk === 'low') {
      pending.value = null
      void run(p)
    } else {
      pending.value = p
    }
  }

  function confirm() {
    const p = pending.value
    pending.value = null
    if (p) void run(p)
  }

  function cancel() {
    pending.value = null
  }

  return reactive({
    systemId,
    pending,
    result,
    can: (action: string, params?: ActionParams) => canRun(system(), action, params),
    risk: (action: string, params?: ActionParams) => riskOf(system().type, action, params),
    isBusy: (action: string, target: string) => running.has(key(action, target)),
    request,
    confirm,
    cancel,
  })
}

export type ActionRunner = ReturnType<typeof useActions>
