<template>
    <el-drawer v-model="visible" title="Downloads" direction="rtl" size="400px" :modal="false">
        <div class="download-manager">
            <!-- Downloads List -->
            <div v-if="downloads.length === 0" class="empty-downloads">
                <el-empty description="No active downloads" />
            </div>

            <div v-else class="downloads-list">
                <div v-for="download in downloads" :key="download.id" class="download-item"
                    :class="{ 'download-completed': download.status === 'completed' }">
                    <!-- File Info -->
                    <div class="download-header">
                        <div class="file-name" :title="download.fileName">
                            {{ download.fileName }}
                        </div>
                        <div class="file-size">
                            {{ formatFileSize(download.size) }}
                        </div>
                    </div>

                    <!-- Progress Bar -->
                    <el-progress :percentage="download.progress" :status="getProgressStatus(download.status)"
                        :stroke-width="6" />

                    <!-- Status Info -->
                    <div class="download-status">
                        <span class="status-text">{{ getStatusText(download) }}</span>
                        <span class="download-speed" v-if="download.speed">
                            {{ formatSpeed(download.speed) }}
                        </span>
                    </div>

                    <!-- Action Buttons -->
                    <div class="download-actions">
                        <el-button v-if="download.status === 'downloading'" size="small" type="danger"
                            @click="cancelDownload(download.id)">
                            Cancel
                        </el-button>
                        <el-button v-if="download.status === 'error'" size="small" type="primary"
                            @click="retryDownload(download.id)">
                            Retry
                        </el-button>
                        <el-button v-if="download.status === 'completed' || download.status === 'error'" size="small"
                            @click="removeDownload(download.id)">
                            Remove
                        </el-button>
                    </div>
                </div>
            </div>

            <!-- Clear All Button -->
            <div v-if="downloads.length > 0" class="download-footer">
                <el-button @click="clearCompleted" size="small" :disabled="!hasCompleted">
                    Clear Completed
                </el-button>
                <el-button @click="clearAll" size="small" type="danger" :disabled="hasActiveDownloads">
                    Clear All
                </el-button>
            </div>
        </div>
    </el-drawer>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'

// Props
const props = defineProps({
    modelValue: {
        type: Boolean,
        default: false
    }
})

// Emits
const emit = defineEmits(['update:modelValue', 'cancel-download', 'retry-download'])

// Reactive data
const visible = ref(props.modelValue)
const downloads = ref([])

// Computed properties
const hasCompleted = computed(() =>
    downloads.value.some(d => d.status === 'completed')
)

const hasActiveDownloads = computed(() =>
    downloads.value.some(d => d.status === 'downloading')
)

// Watch for visibility changes
watch(() => props.modelValue, (newVal) => {
    visible.value = newVal
})

watch(visible, (newVal) => {
    emit('update:modelValue', newVal)
})

// Methods
const getProgressStatus = (status: string) => {
    switch (status) {
        case 'completed': return 'success'
        case 'error': return 'exception'
        case 'cancelled': return 'warning'
        default: return undefined
    }
}

const getStatusText = (download: any) => {
    switch (download.status) {
        case 'downloading':
            return `${Math.round(download.progress)}% - ${formatBytes(download.downloaded)} / ${formatBytes(download.size)}`
        case 'completed':
            return 'Download completed'
        case 'error':
            return `Error: ${download.error}`
        case 'cancelled':
            return 'Download cancelled'
        default:
            return 'Preparing download...'
    }
}

const formatFileSize = (bytes: number) => {
    if (!bytes) return 'Unknown size'
    return formatBytes(bytes)
}

const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

const formatSpeed = (bytesPerSecond: number) => {
    return formatBytes(bytesPerSecond) + '/s'
}

const cancelDownload = (downloadId: string) => {
    emit('cancel-download', downloadId)
    const download = downloads.value.find(d => d.id === downloadId)
    if (download) {
        download.status = 'cancelled'
    }
}

const retryDownload = (downloadId: string) => {
    emit('retry-download', downloadId)
}

const removeDownload = (downloadId: string) => {
    const index = downloads.value.findIndex(d => d.id === downloadId)
    if (index !== -1) {
        downloads.value.splice(index, 1)
    }
}

const clearCompleted = () => {
    downloads.value = downloads.value.filter(d =>
        d.status !== 'completed' && d.status !== 'error' && d.status !== 'cancelled'
    )
}

const clearAll = () => {
    if (!hasActiveDownloads.value) {
        downloads.value = []
    }
}

// Public methods for parent component
const addDownload = (downloadInfo: any) => {
    downloads.value.push({
        id: downloadInfo.id || Date.now().toString(),
        fileName: downloadInfo.fileName,
        filePath: downloadInfo.filePath,
        size: downloadInfo.size || 0,
        downloaded: 0,
        progress: 0,
        status: 'preparing',
        error: null,
        speed: 0,
        startTime: Date.now()
    })
}

const updateDownload = (downloadId: string, updates: any) => {
    const download = downloads.value.find(d => d.id === downloadId)
    if (download) {
        Object.assign(download, updates)

        // Calculate speed if we have progress
        if (updates.downloaded !== undefined && download.startTime) {
            const elapsed = (Date.now() - download.startTime) / 1000 // seconds
            download.speed = elapsed > 0 ? download.downloaded / elapsed : 0
        }
    }
}

const getDownload = (downloadId: string) => {
    return downloads.value.find(d => d.id === downloadId)
}

// Expose methods to parent
defineExpose({
    addDownload,
    updateDownload,
    getDownload,
    removeDownload
})
</script>

<style scoped>
.download-manager {
    padding: 16px;
    height: 100%;
    display: flex;
    flex-direction: column;
}

.downloads-list {
    flex: 1;
    overflow-y: auto;
}

.download-item {
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    padding: 16px;
    margin-bottom: 12px;
    background: var(--el-bg-color);
    transition: all 0.3s ease;
}

.download-item:hover {
    border-color: var(--el-color-primary);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.download-completed {
    background: var(--el-color-success-light-9);
    border-color: var(--el-color-success-light-7);
}

.download-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 8px;
}

.file-name {
    font-weight: 500;
    color: var(--el-text-color-primary);
    max-width: 250px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
}

.file-size {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

.download-status {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin: 8px 0;
    font-size: 12px;
    color: var(--el-text-color-regular);
}

.download-speed {
    color: var(--el-color-primary);
    font-weight: 500;
}

.download-actions {
    display: flex;
    gap: 8px;
    margin-top: 12px;
}

.download-footer {
    border-top: 1px solid var(--el-border-color-light);
    padding-top: 16px;
    display: flex;
    gap: 8px;
    justify-content: flex-end;
}

.empty-downloads {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 200px;
}
</style>
