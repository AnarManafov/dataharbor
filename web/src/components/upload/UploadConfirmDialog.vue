<template>
  <!--
    UploadConfirmDialog
    Shows a summary of the files the user intends to upload, enforces
    server-side limits (max files, max size, max batch), surfaces any conflict
    (file already exists in destDir) and lets the user pick a per-file
    resolution: fail / skip / overwrite / rename.

    Emits `confirm` with a map { relPath -> onConflict } when the user clicks
    Upload. Emits `cancel` otherwise.
  -->
  <el-dialog
    v-model="visible"
    :title="title"
    width="720px"
    :close-on-click-modal="false"
    @close="$emit('cancel')"
  >
    <div class="dest-line">
      Uploading to:
      <code>{{ destDir }}</code>
    </div>

    <!-- Summary / limit warnings -->
    <el-alert
      v-if="limitMessages.length"
      type="error"
      :closable="false"
      class="limit-alert"
    >
      <ul>
        <li v-for="m in limitMessages" :key="m">{{ m }}</li>
      </ul>
    </el-alert>

    <div class="summary">
      <span>{{ files.length }} file{{ files.length === 1 ? '' : 's' }}</span>
      <span class="sep">&middot;</span>
      <span>{{ formatSize(totalSize) }} total</span>
      <span v-if="limits" class="sep">&middot;</span>
      <span v-if="limits" class="limit-hint">
        max {{ limits.maxFilesPerBatch }} files,
        {{ formatSize(limits.maxFileSize) }} per file,
        {{ formatSize(limits.maxBatchSize) }} per batch
      </span>
    </div>

    <el-table :data="filesWithMeta" size="small" height="340" class="upload-file-table">
      <el-table-column prop="relPath" label="File" show-overflow-tooltip />
      <el-table-column prop="size" label="Size" width="100">
        <template #default="{ row }">{{ formatSize(row.size) }}</template>
      </el-table-column>
      <el-table-column label="Status" width="120">
        <template #default="{ row }">
          <el-tag v-if="row.tooBig" type="danger" size="small">Too large</el-tag>
          <el-tag v-else-if="row.conflict === 'exists'" type="warning" size="small">Exists</el-tag>
          <el-tag v-else type="success" size="small">New</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="On conflict" width="160">
        <template #default="{ row }">
          <el-select
            v-model="decisions[row.relPath]"
            size="small"
            :disabled="row.conflict !== 'exists'"
            style="width: 140px"
          >
            <el-option label="Fail (reject)" value="fail" />
            <el-option label="Skip" value="skip" />
            <el-option
              label="Overwrite"
              value="overwrite"
              :disabled="!limits?.allowOverwrite"
            />
            <el-option label="Keep both (rename)" value="rename" />
          </el-select>
        </template>
      </el-table-column>
    </el-table>

    <template #footer>
      <el-button @click="$emit('cancel')">Cancel</el-button>
      <el-button
        type="primary"
        :disabled="limitMessages.length > 0 || files.length === 0"
        @click="onConfirm"
      >
        Upload
      </el-button>
    </template>
  </el-dialog>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  destDir: { type: String, required: true },
  /** @type {Array<{relPath:string,size:number,conflict?:string}>} */
  files: { type: Array, default: () => [] },
  /** UploadLimits from GET /api/v1/xrd/upload/limits */
  limits: { type: Object, default: null },
})

const emit = defineEmits(['update:modelValue', 'confirm', 'cancel'])

const visible = ref(props.modelValue)
watch(() => props.modelValue, (v) => { visible.value = v })
watch(visible, (v) => emit('update:modelValue', v))

// Map relPath -> default conflict decision. Default is 'fail' which mirrors
// the server-side default (per GH-56 design). If a file does not yet exist
// on the server the selector is disabled but the value still flows through.
const decisions = reactive({})
watch(
  () => props.files,
  (list) => {
    for (const f of list) {
      if (!(f.relPath in decisions)) decisions[f.relPath] = 'fail'
    }
  },
  { immediate: true, deep: false }
)

const title = computed(() =>
  props.files.length > 1
    ? `Upload ${props.files.length} files`
    : 'Upload file'
)

const totalSize = computed(() =>
  props.files.reduce((a, f) => a + (f.size || 0), 0)
)

const limitMessages = computed(() => {
  const out = []
  if (!props.limits) return out
  if (!props.limits.enabled) {
    out.push('Uploads are currently disabled on this server.')
    return out
  }
  if (props.files.length > props.limits.maxFilesPerBatch) {
    out.push(
      `Batch has ${props.files.length} files; server limit is ${props.limits.maxFilesPerBatch}.`
    )
  }
  if (totalSize.value > props.limits.maxBatchSize) {
    out.push(
      `Total size ${formatSize(totalSize.value)} exceeds server batch limit of ${formatSize(props.limits.maxBatchSize)}.`
    )
  }
  // Per-file size is flagged on each row via row.tooBig; include a summary
  // line so the Upload button disabling is obvious.
  const oversized = props.files.filter((f) => f.size > props.limits.maxFileSize)
  if (oversized.length) {
    out.push(
      `${oversized.length} file(s) exceed the per-file limit of ${formatSize(props.limits.maxFileSize)}.`
    )
  }
  return out
})

// Decorate file rows with tooBig flag without mutating the props array.
const filesWithMeta = computed(() =>
  props.files.map((f) => ({
    ...f,
    tooBig: props.limits ? f.size > props.limits.maxFileSize : false,
  }))
)

function onConfirm() {
  // Strip defaulted 'fail' for files that don't actually conflict so the
  // backend doesn't have to consider them, but keep the chosen action for
  // conflicting files intact.
  const out = {}
  for (const f of props.files) {
    out[f.relPath] = decisions[f.relPath] || 'fail'
  }
  emit('confirm', out)
}

function formatSize(bytes) {
  if (bytes === null || bytes === undefined) return '-'
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
.dest-line {
  font-size: 13px;
  margin-bottom: 8px;
  color: var(--el-text-color-regular);
}

.dest-line code {
  padding: 1px 6px;
  background: var(--el-fill-color);
  border-radius: 3px;
  font-family: var(--el-font-family-monospace, monospace);
}

.limit-alert {
  margin-bottom: 10px;
}

.limit-alert ul {
  margin: 0;
  padding-left: 18px;
}

.summary {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-bottom: 8px;
}

.summary .sep {
  margin: 0 6px;
}

.limit-hint {
  color: var(--el-text-color-placeholder);
}

.upload-file-table :deep(.cell) {
  font-size: 12px;
}
</style>
