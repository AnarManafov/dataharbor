<template>
  <!--
    UploadProgressPanel
    GitHub-style floating panel in the bottom-right corner reflecting the
    current upload batch. Collapsible. Offers pause/resume/cancel when the
    batch is active and a dismiss button once it is complete.

    Bound directly to the reactive state from uploadService (props.state).
  -->
  <transition name="slide-up">
    <div v-if="state && state.files.length" class="upload-panel" :class="{ collapsed }">
      <div class="upload-panel__header" @click="collapsed = !collapsed">
        <el-icon class="handle"><component :is="collapsed ? ArrowUp : ArrowDown" /></el-icon>
        <span class="title">{{ headerText }}</span>
        <span class="summary">{{ percentText }}</span>
        <div class="actions" @click.stop>
          <el-button
            v-if="isActive && state.status !== 'paused'"
            size="small"
            link
            type="primary"
            @click="$emit('pause')"
          >
            Pause
          </el-button>
          <el-button
            v-if="state.status === 'paused'"
            size="small"
            link
            type="primary"
            @click="$emit('resume')"
          >
            Resume
          </el-button>
          <el-button
            v-if="isActive || state.status === 'paused'"
            size="small"
            link
            type="danger"
            @click="onAbort"
          >
            Cancel
          </el-button>
          <el-button
            v-else
            size="small"
            link
            @click="$emit('dismiss')"
          >
            Dismiss
          </el-button>
        </div>
      </div>

      <div v-if="!collapsed" class="upload-panel__body">
        <el-alert
          v-if="state.errorMessage && state.status === 'failed'"
          class="upload-panel__error"
          :title="state.errorMessage"
          type="error"
          :closable="false"
          show-icon
        />
        <div class="overall">
          <el-progress
            :percentage="percentNumber"
            :status="progressStatus"
            :stroke-width="6"
          />
          <div class="overall-meta">
            <span>{{ bytesText }}</span>
            <span v-if="speedText">&middot; {{ speedText }}</span>
          </div>
        </div>

        <div class="files">
          <div
            v-for="f in state.files"
            :key="f.id"
            class="file-row"
            :class="`state-${f.state}`"
          >
            <el-icon class="file-icon"><component :is="iconFor(f.state)" /></el-icon>
            <span class="file-name" :title="f.relPath">{{ f.relPath }}</span>
            <span class="file-size">{{ formatSize(f.size) }}</span>
            <span class="file-state">{{ labelFor(f) }}</span>
          </div>
        </div>
      </div>
    </div>
  </transition>
</template>

<script setup>
import { computed, ref } from 'vue'
import {
  ArrowUp,
  ArrowDown,
  Loading,
  CircleCheck,
  CircleClose,
  Warning,
  Clock,
  VideoPause,
} from '@element-plus/icons-vue'

const props = defineProps({
  // Reactive state from uploadService.state.
  state: { type: Object, required: true },
})

const emit = defineEmits(['pause', 'resume', 'abort', 'dismiss'])

const collapsed = ref(false)

const isActive = computed(() =>
  ['preparing', 'uploading'].includes(props.state.status)
)

const percentNumber = computed(() => {
  const total = props.state.overall.totalBytes
  const done = props.state.overall.transferredBytes
  if (total <= 0) return 0
  return Math.min(100, Math.round((done / total) * 100))
})

const percentText = computed(() => `${percentNumber.value}%`)

const bytesText = computed(() =>
  `${formatSize(props.state.overall.transferredBytes)} / ${formatSize(
    props.state.overall.totalBytes
  )}`
)

const speedText = computed(() => {
  const bps = props.state.overall.speedBps
  if (!bps || !isActive.value) return ''
  return `${formatSize(bps)}/s`
})

const headerText = computed(() => {
  const n = props.state.files.length
  const done = props.state.files.filter((f) => f.state === 'done').length
  const errored = props.state.files.filter((f) => f.state === 'error').length
  const skipped = props.state.files.filter((f) => f.state === 'skipped').length
  switch (props.state.status) {
    case 'preparing':
      return `Preparing ${n} upload${n === 1 ? '' : 's'}…`
    case 'uploading':
      return `Uploading ${done + 1}/${n}`
    case 'paused':
      return `Paused · ${done}/${n} complete`
    case 'aborted':
      return 'Upload cancelled'
    case 'failed':
      return `Failed (${errored} error${errored === 1 ? '' : 's'})`
    case 'skipped':
      return `Nothing uploaded · ${skipped} skipped (already exist)`
    case 'completed':
      return errored
        ? `Completed with ${errored} error${errored === 1 ? '' : 's'}`
        : `Uploaded ${done} file${done === 1 ? '' : 's'}`
    default:
      return 'Uploads'
  }
})

const progressStatus = computed(() => {
  if (props.state.status === 'failed') return 'exception'
  if (props.state.status === 'completed' &&
    !props.state.files.some((f) => f.state === 'error')) return 'success'
  return ''
})

function iconFor(s) {
  switch (s) {
    case 'done':
      return CircleCheck
    case 'error':
      return CircleClose
    case 'skipped':
      return Warning
    case 'paused':
      return VideoPause
    case 'queued':
    case 'hashing':
      return Clock
    default:
      return Loading
  }
}

function labelFor(f) {
  if (f.state === 'error') return f.error || 'error'
  if (f.state === 'skipped') return f.error || 'skipped'
  if (f.state === 'uploading') {
    const pct = f.size > 0 ? Math.round((f.bytesSent / f.size) * 100) : 0
    return `${pct}%`
  }
  return f.state
}

function onAbort() {
  emit('abort')
}

function formatSize(bytes) {
  if (!bytes && bytes !== 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = bytes
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}
</script>

<style scoped>
.upload-panel {
  position: fixed;
  right: 24px;
  bottom: 24px;
  width: 420px;
  max-width: calc(100vw - 48px);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
  z-index: 2000;
  overflow: hidden;
}

.upload-panel__header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  cursor: pointer;
  user-select: none;
}

.upload-panel__header .title {
  font-weight: 600;
  font-size: 13px;
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.upload-panel__header .summary {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.upload-panel__header .actions {
  display: flex;
  gap: 4px;
}

.upload-panel__body {
  padding: 10px 12px;
  max-height: 320px;
  overflow-y: auto;
}

.upload-panel__error {
  margin-bottom: 10px;
}

.overall {
  margin-bottom: 10px;
}

.overall-meta {
  margin-top: 4px;
  font-size: 11px;
  color: var(--el-text-color-secondary);
}

.files {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.file-row {
  display: grid;
  grid-template-columns: 18px 1fr auto auto;
  gap: 8px;
  align-items: center;
  font-size: 12px;
  padding: 3px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.file-row:last-child {
  border-bottom: 0;
}

.file-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-size {
  color: var(--el-text-color-secondary);
  min-width: 60px;
  text-align: right;
}

.file-state {
  color: var(--el-text-color-secondary);
  min-width: 56px;
  text-align: right;
}

.state-done .file-icon {
  color: var(--el-color-success);
}

.state-error .file-icon,
.state-error .file-state {
  color: var(--el-color-danger);
}

.state-skipped .file-icon,
.state-skipped .file-state {
  color: var(--el-color-warning);
}

.slide-up-enter-active,
.slide-up-leave-active {
  transition: transform 0.2s ease, opacity 0.2s ease;
}

.slide-up-enter-from,
.slide-up-leave-to {
  transform: translateY(24px);
  opacity: 0;
}
</style>
