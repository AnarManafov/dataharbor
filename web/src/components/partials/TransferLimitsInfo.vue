<template>
  <!--
    TransferLimitsInfo
    Small info affordance (e.g. placed next to the Upload button) that on
    click/hover shows the server-configured upload and download limits. Helps
    users understand up-front why a drop or selection might be rejected.
  -->
  <el-popover
    placement="bottom-end"
    :width="320"
    trigger="click"
    :teleported="true"
  >
    <template #reference>
      <el-button size="small" link type="primary" :icon="InfoFilled">
        Transfer limits
      </el-button>
    </template>
    <div class="limits-popover">
      <div class="section-title">Upload</div>
      <ul v-if="limits?.upload">
        <li>Enabled: <b>{{ limits.upload.enabled ? 'yes' : 'no' }}</b></li>
        <li>Max file size: <b>{{ formatSize(limits.upload.maxFileSize) }}</b></li>
        <li>Max batch size: <b>{{ formatSize(limits.upload.maxBatchSize) }}</b></li>
        <li>Max files per batch: <b>{{ limits.upload.maxFilesPerBatch }}</b></li>
        <li>Concurrent uploads / user: <b>{{ limits.upload.maxConcurrentPerUser }}</b></li>
        <li>Chunk size: <b>{{ formatSize(limits.upload.chunkSize) }}</b></li>
        <li>Overwrite allowed: <b>{{ limits.upload.allowOverwrite ? 'yes' : 'no' }}</b></li>
        <li>Checksum: <b>{{ limits.upload.checksumAlgo }}</b></li>
      </ul>
      <div v-else class="muted">Upload limits are not available.</div>
      <div class="section-title" style="margin-top: 10px;">Download</div>
      <ul v-if="limits?.download">
        <li>Max batch files: <b>{{ limits.download.maxBatchFiles }}</b></li>
        <li>Max batch size: <b>{{ formatSize(limits.download.maxBatchSizeMB * 1024 * 1024) }}</b></li>
        <li>Compression: <b>{{ limits.download.batchCompression ? 'yes' : 'no' }}</b></li>
      </ul>
      <div v-else class="muted">Download limits are not available.</div>
    </div>
  </el-popover>
</template>

<script setup>
import { InfoFilled } from '@element-plus/icons-vue'

defineProps({
  // The shape returned by GET /api/v1/xrd/upload/limits, i.e.
  // { upload: {...}, download: {...} }
  limits: { type: Object, default: null },
})

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
.limits-popover ul {
  list-style: none;
  padding: 0;
  margin: 0;
  font-size: var(--dh-font-size-sm);
  line-height: 1.7;
}

.section-title {
  font-weight: 600;
  font-size: var(--dh-font-size-sm);
  color: var(--el-color-primary);
  margin-bottom: 4px;
}

.muted {
  color: var(--el-text-color-secondary);
  font-size: var(--dh-font-size-sm);
}
</style>
