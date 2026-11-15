import type {
  DockerMetrics,
  LinuxMetrics,
  QBittorrentMetrics,
  SystemMetrics,
  SystemStatus,
  SystemType,
  TemperatureMetrics,
  UNASMetrics,
  UnifiMetrics,
} from '@/types/metrics'
import { formatBytes, formatSpeed, formatUptime } from '@/composables/useFormatters'

// An offline system carries `metrics: {}`; these guards return the typed payload or null.
export const linuxOf = (s: SystemMetrics) => (s.type === 'linux' && 'cpu' in s.metrics ? (s.metrics as LinuxMetrics) : null)
export const dockerOf = (s: SystemMetrics) => (s.type === 'docker' && 'containers' in s.metrics ? (s.metrics as DockerMetrics) : null)
export const qbitOf = (s: SystemMetrics) => (s.type === 'qbittorrent' && 'torrents' in s.metrics ? (s.metrics as QBittorrentMetrics) : null)
export const unifiOf = (s: SystemMetrics) => (s.type === 'unifi' && 'devices' in s.metrics ? (s.metrics as UnifiMetrics) : null)
export const unasOf = (s: SystemMetrics) => (s.type === 'unas' && 'pools' in s.metrics ? (s.metrics as UNASMetrics) : null)

export const STATUS_GLYPH: Record<SystemStatus, string> = {
  healthy: '●',
  warning: '▲',
  critical: '✖',
  offline: '○',
}

export const STATUS_CLASS: Record<SystemStatus, string> = {
  healthy: 'ok',
  warning: 'warn',
  critical: 'crit',
  offline: 'off',
}

export const TYPE_LABEL: Record<SystemType, string> = {
  linux: 'linux',
  docker: 'docker',
  qbittorrent: 'torrent',
  unifi: 'router',
  unas: 'storage',
}

// Command-like title of a system's compact overview block.
export const TYPE_COMMAND: Record<SystemType, string> = {
  linux: 'top -bn1',
  docker: 'docker ps',
  qbittorrent: 'qbt stats',
  unifi: 'unifi stat',
  unas: 'zpool list',
}

/** Temperature level: crit at the sensor's critical point, warn at its high point (defaults 85/70 °C). */
export function tempLevel(t: TemperatureMetrics): 'ok' | 'warn' | 'crit' {
  if (t.current >= (t.critical ?? 85)) return 'crit'
  if (t.current >= (t.high ?? 70)) return 'warn'
  return 'ok'
}

export function hottest(temps: TemperatureMetrics[] | null | undefined): TemperatureMetrics | null {
  if (!temps?.length) return null
  return temps.reduce((a, b) => (b.current > a.current ? b : a))
}

/** One-line summary used in the `status --all` table. */
export function summarize(s: SystemMetrics): string {
  if (s.status === 'offline') return s.error ?? 'offline'
  const linux = linuxOf(s)
  if (linux) {
    return `cpu ${linux.cpu.percent.toFixed(0)}% mem ${linux.memory.percent.toFixed(0)}% up ${formatUptime(linux.uptime).split(' ')[0]}`
  }
  const docker = dockerOf(s)
  if (docker) return `${docker.running_count}/${docker.total_count} running`
  const qbit = qbitOf(s)
  if (qbit) {
    return `↓${formatSpeed(qbit.download_speed)} ↑${formatSpeed(qbit.upload_speed)} ${qbit.active_downloads + qbit.active_uploads} active`
  }
  const unifi = unifiOf(s)
  if (unifi) {
    const online = unifi.devices.filter(d => d.status === 'online').length
    const hot = hottest(unifi.devices.flatMap(d => d.temperatures ?? []))
    return `${unifi.clients_count} clients, ${online}/${unifi.devices.length} devices up${hot ? `, max ${hot.current.toFixed(0)}°C` : ''}`
  }
  const unas = unasOf(s)
  if (unas) {
    const pct = unas.total_capacity ? (unas.used_capacity / unas.total_capacity) * 100 : 0
    return `${unas.pools.length} pool${unas.pools.length === 1 ? '' : 's'} ${pct.toFixed(0)}% of ${formatBytes(unas.total_capacity)}`
  }
  return s.error ?? ''
}

/** Block meta for a filtered table: "3 rows" or "2/3 rows". */
export function rowsMeta(shown: number, total: number): string {
  const unit = total === 1 ? 'row' : 'rows'
  return shown === total ? `${total} ${unit}` : `${shown}/${total} ${unit}`
}
