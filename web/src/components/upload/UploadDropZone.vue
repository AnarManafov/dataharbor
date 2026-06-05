<template>
  <!--
    UploadDropZone
    Transparent full-surface overlay that activates while the user is dragging
    files over the browser view. It emits `files` with a DataTransferItemList
    when the user drops, so the parent can build its own enumeration and open
    the confirmation dialog.
  -->
  <div
    class="upload-dropzone"
    :class="{ active: isDragging }"
    @dragenter.prevent="onEnter"
    @dragover.prevent="onOver"
    @dragleave.prevent="onLeave"
    @drop.prevent="onDrop"
  >
    <div v-if="isDragging" class="upload-dropzone__hint">
      <el-icon :size="36"><UploadFilled /></el-icon>
      <div class="hint-title">Drop files to upload</div>
      <div class="hint-sub">
        Files will be uploaded to
        <code>{{ destDir || 'current directory' }}</code>
      </div>
    </div>
    <slot />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { UploadFilled } from '@element-plus/icons-vue'

const props = defineProps({
  destDir: { type: String, default: '' },
  // When false the overlay is inert (e.g. user is not authenticated or
  // upload feature is globally disabled).
  enabled: { type: Boolean, default: true },
})

const emit = defineEmits(['files'])

// The browser fires dragenter/leave for every child element traversed. We
// rely on a counter to debounce flicker — overlay stays active as long as
// _any_ pending enter event hasn't been matched by a leave.
const dragCounter = ref(0)
const isDragging = ref(false)

function onEnter(evt) {
  if (!props.enabled) return
  if (!hasFiles(evt)) return
  dragCounter.value++
  isDragging.value = true
}

function onOver(evt) {
  if (!props.enabled) return
  if (!hasFiles(evt)) return
  // dataTransfer.dropEffect must be set on every dragover to keep the cursor.
  evt.dataTransfer.dropEffect = 'copy'
}

function onLeave() {
  if (!props.enabled) return
  dragCounter.value = Math.max(0, dragCounter.value - 1)
  if (dragCounter.value === 0) isDragging.value = false
}

function onDrop(evt) {
  dragCounter.value = 0
  isDragging.value = false
  if (!props.enabled) return
  const items = evt.dataTransfer?.items
  const files = evt.dataTransfer?.files
  if (items && items.length) {
    emit('files', { kind: 'items', items })
  } else if (files && files.length) {
    emit('files', { kind: 'files', files })
  }
}

function hasFiles(evt) {
  // Chrome reports types as ['Files']; Firefox sometimes includes extra types.
  const types = evt.dataTransfer?.types
  if (!types) return false
  return Array.from(types).includes('Files')
}
</script>

<style scoped>
.upload-dropzone {
  position: relative;
  width: 100%;
  height: 100%;
}

.upload-dropzone.active::before {
  content: '';
  position: absolute;
  inset: 0;
  background: rgba(64, 158, 255, 0.08);
  border: 2px dashed var(--el-color-primary);
  border-radius: 6px;
  pointer-events: none;
  z-index: 10;
}

.upload-dropzone__hint {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  color: var(--el-color-primary);
  pointer-events: none;
  z-index: 11;
  background: var(--el-bg-color);
  padding: 16px 24px;
  border-radius: 8px;
  box-shadow: 0 4px 18px rgba(0, 0, 0, 0.15);
}

.hint-title {
  font-size: 16px;
  font-weight: 600;
}

.hint-sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.hint-sub code {
  padding: 0 4px;
  background: var(--el-fill-color);
  border-radius: 3px;
  font-family: var(--el-font-family-monospace, monospace);
}
</style>
