/**
 * Shared formatting utilities for metrics display.
 * Extracted to avoid duplication across components.
 */

/**
 * Format bytes to human-readable string (B, KB, MB, GB, TB, PB)
 */
export function formatBytes(bytes: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(1)} ${units[i]}`
}

/**
 * Format bytes per second to human-readable speed string
 */
export function formatSpeed(bytesPerSec: number): string {
  if (bytesPerSec < 1024) return `${bytesPerSec.toFixed(0)} B/s`
  if (bytesPerSec < 1024 * 1024) return `${(bytesPerSec / 1024).toFixed(1)} KB/s`
  return `${(bytesPerSec / 1024 / 1024).toFixed(2)} MB/s`
}

/**
 * Format seconds to human-readable uptime string (Xd Xh Xm)
 */
export function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)

  if (days > 0) return `${days}d ${hours}h ${minutes}m`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}

/**
 * Format ETA seconds to human-readable string
 */
export function formatEta(seconds: number | null | undefined): string {
  if (!seconds) return '-'
  if (seconds > 86400) return `${Math.floor(seconds / 86400)}d`
  if (seconds > 3600) return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
  if (seconds > 60) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`
  return `${seconds}s`
}

/**
 * Format percentage with fixed decimals
 */
export function formatPercent(value: number, decimals = 1): string {
  return `${value.toFixed(decimals)}%`
}

/**
 * Format a date as a 24h wall-clock time (HH:MM:SS)
 */
export function formatClock(date: Date | string | null | undefined): string {
  if (!date) return '--:--:--'
  const d = typeof date === 'string' ? new Date(date) : date
  return d.toLocaleTimeString([], { hour12: false })
}

/**
 * Format a duration in milliseconds (e.g. 120ms, 1.2s)
 */
export function formatMs(ms: number): string {
  return ms < 1000 ? `${Math.round(ms)}ms` : `${(ms / 1000).toFixed(1)}s`
}

export type Level = 'ok' | 'warn' | 'crit'

/**
 * Map a usage percentage to a semantic level class
 */
export function usageLevel(percent: number, warn = 75, crit = 90): Level {
  if (percent >= crit) return 'crit'
  if (percent >= warn) return 'warn'
  return 'ok'
}

const PARTIAL_BLOCKS = ['', '▏', '▎', '▍', '▌', '▋', '▊', '▉']

/**
 * Render a percentage as a fixed-width text bar (eighth-block resolution)
 */
export function formatBar(percent: number, width = 20): string {
  const eighths = Math.round((Math.min(Math.max(percent, 0), 100) / 100) * width * 8)
  const full = Math.floor(eighths / 8)
  const part = PARTIAL_BLOCKS[eighths % 8]
  const used = full + (part ? 1 : 0)
  return '█'.repeat(full) + part + '░'.repeat(width - used)
}

/**
 * Composable that returns all formatters for use in components
 */
export function useFormatters() {
  return {
    formatBytes,
    formatSpeed,
    formatUptime,
    formatEta,
    formatPercent,
    formatClock,
    formatMs,
    formatBar,
    usageLevel,
  }
}
