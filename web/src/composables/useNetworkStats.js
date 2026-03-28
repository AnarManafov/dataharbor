import { ref, computed, readonly } from 'vue'

// Singleton state shared across all components
const latencyMs = ref(null)
const connectMs = ref(null)
const latencyStatus = ref('unknown') // 'good', 'fair', 'poor', 'unknown'
const queryTimeMs = ref(null)
const downloadSpeeds = ref([]) // recent download speeds in bytes/sec
const lastPingTime = ref(null)
const isPinging = ref(false)

const MAX_SPEED_HISTORY = 20

/**
 * Composable for tracking and displaying network performance stats
 * for the XRD storage connection. Shared singleton state.
 */
export function useNetworkStats() {
  // Computed: average download speed from recent history
  const avgSpeedBytesPerSec = computed(() => {
    if (downloadSpeeds.value.length === 0) return null
    const sum = downloadSpeeds.value.reduce((a, b) => a + b, 0)
    return sum / downloadSpeeds.value.length
  })

  // Computed: formatted average speed
  const avgSpeedFormatted = computed(() => {
    const speed = avgSpeedBytesPerSec.value
    if (speed === null) return null
    return formatSpeed(speed)
  })

  // Computed: latency quality indicator
  const latencyQuality = computed(() => {
    if (latencyMs.value === null) return { label: 'N/A', color: 'var(--el-text-color-secondary)' }
    if (latencyMs.value < 50) return { label: 'Excellent', color: 'var(--el-color-success)' }
    if (latencyMs.value < 150) return { label: 'Good', color: 'var(--el-color-success-light-3)' }
    if (latencyMs.value < 500) return { label: 'Fair', color: 'var(--el-color-warning)' }
    return { label: 'Poor', color: 'var(--el-color-danger)' }
  })

  /**
   * Record a completed download's speed for averaging
   */
  function recordDownloadSpeed(bytesPerSec) {
    if (bytesPerSec > 0) {
      downloadSpeeds.value.push(bytesPerSec)
      // Keep only recent entries
      if (downloadSpeeds.value.length > MAX_SPEED_HISTORY) {
        downloadSpeeds.value.shift()
      }
    }
  }

  /**
   * Update latency from a ping response
   */
  function updateLatency(ms, connMs) {
    latencyMs.value = parseFloat(ms.toFixed(1))
    if (connMs != null) connectMs.value = parseFloat(connMs.toFixed(1))
    lastPingTime.value = Date.now()
    if (ms < 50) latencyStatus.value = 'good'
    else if (ms < 150) latencyStatus.value = 'fair'
    else latencyStatus.value = 'poor'
  }

  /**
   * Update the directory query time
   */
  function updateQueryTime(ms) {
    queryTimeMs.value = ms
  }

  /**
   * Estimate download time for a file of given size in bytes
   * Returns object with formatted time and raw seconds, or null if no data
   */
  function estimateDownloadTime(fileSizeBytes) {
    const speed = avgSpeedBytesPerSec.value
    if (speed === null || speed === 0 || !fileSizeBytes) return null

    const seconds = fileSizeBytes / speed
    return {
      seconds,
      formatted: formatDuration(seconds),
      speedFormatted: formatSpeed(speed)
    }
  }

  return {
    // State (readonly)
    latencyMs: readonly(latencyMs),
    connectMs: readonly(connectMs),
    latencyStatus: readonly(latencyStatus),
    queryTimeMs: readonly(queryTimeMs),
    avgSpeedBytesPerSec,
    avgSpeedFormatted,
    latencyQuality,
    downloadSpeeds: readonly(downloadSpeeds),
    lastPingTime: readonly(lastPingTime),
    isPinging: readonly(isPinging),
    // For internal use by ping caller
    _isPinging: isPinging,
    // Methods
    recordDownloadSpeed,
    updateLatency,
    updateQueryTime,
    estimateDownloadTime
  }
}

/**
 * Format bytes/sec into human-readable speed string
 */
export function formatSpeed(bytesPerSec) {
  if (bytesPerSec >= 1024 * 1024 * 1024) {
    return `${(bytesPerSec / (1024 * 1024 * 1024)).toFixed(1)} GB/s`
  }
  if (bytesPerSec >= 1024 * 1024) {
    return `${(bytesPerSec / (1024 * 1024)).toFixed(1)} MB/s`
  }
  if (bytesPerSec >= 1024) {
    return `${(bytesPerSec / 1024).toFixed(1)} KB/s`
  }
  return `${Math.round(bytesPerSec)} B/s`
}

/**
 * Format seconds into human-readable duration
 */
export function formatDuration(seconds) {
  if (seconds < 1) return '< 1s'
  if (seconds < 60) return `~${Math.round(seconds)}s`
  if (seconds < 3600) {
    const mins = Math.floor(seconds / 60)
    const secs = Math.round(seconds % 60)
    return secs > 0 ? `~${mins}m ${secs}s` : `~${mins}m`
  }
  const hours = Math.floor(seconds / 3600)
  const mins = Math.round((seconds % 3600) / 60)
  return mins > 0 ? `~${hours}h ${mins}m` : `~${hours}h`
}
