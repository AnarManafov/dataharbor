<template>
    <div class="download-test-page">
        <div class="container">
            <h1>StreamSaver Download Test</h1>

            <!-- Compatibility Check -->
            <el-card class="compatibility-card">
                <h3>Browser Compatibility</h3>
                <div class="compatibility-info">
                    <div class="compat-item"
                        :class="{ success: compatibility.isSecure, warning: !compatibility.isSecure }">
                        <el-icon>
                            <Check v-if="compatibility.isSecure" />
                            <Warning v-else />
                        </el-icon>
                        Secure Context (HTTPS): {{ compatibility.isSecure ? 'Yes' : 'No' }}
                    </div>
                    <div class="compat-item"
                        :class="{ success: compatibility.hasServiceWorker, warning: !compatibility.hasServiceWorker }">
                        <el-icon>
                            <Check v-if="compatibility.hasServiceWorker" />
                            <Warning v-else />
                        </el-icon>
                        Service Worker: {{ compatibility.hasServiceWorker ? 'Available' : 'Not Available' }}
                    </div>
                    <div class="compat-item"
                        :class="{ success: compatibility.hasStreams, warning: !compatibility.hasStreams }">
                        <el-icon>
                            <Check v-if="compatibility.hasStreams" />
                            <Warning v-else />
                        </el-icon>
                        Streams API: {{ compatibility.hasStreams ? 'Native' : 'Polyfill Required' }}
                    </div>
                </div>
                <div v-if="compatibility.warnings.length > 0" class="warnings">
                    <h4>Warnings:</h4>
                    <ul>
                        <li v-for="warning in compatibility.warnings" :key="warning">{{ warning }}</li>
                    </ul>
                </div>
            </el-card>

            <!-- Test Download Form -->
            <el-card class="test-card">
                <h3>Test Download</h3>
                <el-form :model="testForm" label-width="120px">
                    <el-form-item label="File Path">
                        <el-input v-model="testForm.filePath" placeholder="/path/to/test/file.txt" />
                    </el-form-item>
                    <el-form-item label="File Name">
                        <el-input v-model="testForm.fileName" placeholder="downloaded-file.txt" />
                    </el-form-item>
                    <el-form-item label="File Size">
                        <el-input v-model="testForm.fileSize" placeholder="1024000" type="number" />
                        <small>Size in bytes (optional, for progress tracking)</small>
                    </el-form-item>
                    <el-form-item>
                        <el-button type="primary" @click="testBasicDownload" :loading="downloading"
                            :disabled="downloading">
                            Test Basic Download
                        </el-button>
                        <el-button type="success" @click="testEnhancedDownload" :loading="downloading"
                            :disabled="downloading">
                            Test Enhanced Download
                        </el-button>
                    </el-form-item>
                </el-form>
            </el-card>

            <!-- Progress Display -->
            <el-card v-if="downloadProgress.active" class="progress-card">
                <h3>Download Progress</h3>
                <el-progress :percentage="downloadProgress.percentage" :status="downloadProgress.status"
                    :stroke-width="8" />
                <div class="progress-info">
                    <p>Downloaded: {{ formatBytes(downloadProgress.downloaded) }} / {{
                        formatBytes(downloadProgress.total) }}</p>
                    <p v-if="downloadProgress.speed > 0">Speed: {{ formatBytes(downloadProgress.speed) }}/s</p>
                    <p>Status: {{ downloadProgress.statusText }}</p>
                </div>
                <el-button v-if="downloadProgress.canCancel" type="danger" @click="cancelDownload">
                    Cancel Download
                </el-button>
            </el-card>

            <!-- Test Results -->
            <el-card v-if="testResults.length > 0" class="results-card">
                <h3>Test Results</h3>
                <div v-for="result in testResults" :key="result.id" class="test-result"
                    :class="result.success ? 'success' : 'error'">
                    <div class="result-header">
                        <el-icon>
                            <Check v-if="result.success" />
                            <Close v-else />
                        </el-icon>
                        <strong>{{ result.type }}</strong> - {{ result.fileName }}
                    </div>
                    <div class="result-details">
                        <p>Duration: {{ result.duration }}ms</p>
                        <p v-if="result.error">Error: {{ result.error }}</p>
                        <p v-if="result.size">Size: {{ formatBytes(result.size) }}</p>
                        <p v-if="result.speed">Speed: {{ result.speed.mbps }} MB/s ({{ result.speed.duration }}s)</p>
                    </div>
                </div>
            </el-card>

            <!-- Sample Files for Testing -->
            <el-card class="samples-card">
                <h3>Sample Test Files</h3>
                <p>Create these test files on your XRootD server for testing:</p>
                <div class="sample-files">
                    <div class="sample-file" v-for="sample in sampleFiles" :key="sample.path">
                        <code>{{ sample.path }}</code>
                        <span class="sample-size">{{ sample.size }}</span>
                        <el-button size="small" @click="downloadSample(sample)" :loading="downloading"
                            :disabled="downloading">
                            Download
                        </el-button>
                    </div>
                </div>
            </el-card>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Check, Warning, Close } from '@element-plus/icons-vue'
import { DownloadService } from '@/services/downloadService'
import { EnhancedDownloadService } from '@/services/enhancedDownloadService'

// Reactive data
const compatibility = ref({})
const testForm = ref({
    filePath: '/tmp/test.txt',
    fileName: 'test-download.txt',
    fileSize: 1024
})

const downloading = ref(false)
const downloadProgress = ref({
    active: false,
    percentage: 0,
    downloaded: 0,
    total: 0,
    speed: 0,
    status: '',
    statusText: '',
    canCancel: false,
    downloadId: null
})

const testResults = ref([])
const currentDownloadId = ref(null)

const sampleFiles = ref([
    { path: '/tmp/small.txt', size: '1KB', description: 'Small text file' },
    { path: '/tmp/medium.bin', size: '10MB', description: 'Medium binary file' },
    { path: '/tmp/large.iso', size: '1GB', description: 'Large ISO file' }
])

// Methods
const formatBytes = (bytes) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

const testBasicDownload = async () => {
    if (!testForm.value.filePath || !testForm.value.fileName) {
        ElMessage.error('Please fill in file path and name')
        return
    }

    downloading.value = true
    const startTime = Date.now()

    try {
        const result = await DownloadService.downloadFile(
            testForm.value.filePath,
            testForm.value.fileName,
            parseInt(testForm.value.fileSize) || 0
        )

        const duration = Date.now() - startTime

        testResults.value.unshift({
            id: Date.now(),
            type: 'Basic Download',
            fileName: testForm.value.fileName,
            success: result.success,
            error: result.error,
            duration,
            size: parseInt(testForm.value.fileSize) || 0,
            speed: result.speed
        })

        if (result.success) {
            let message = 'Basic download completed successfully'
            if (result.speed) {
                message += ` (${result.speed.mbps} MB/s)`
            }
            ElMessage.success(message)
        } else {
            ElMessage.error(`Basic download failed: ${result.error}`)
        }

    } catch (error) {
        ElMessage.error(`Download error: ${error.message}`)
        testResults.value.unshift({
            id: Date.now(),
            type: 'Basic Download',
            fileName: testForm.value.fileName,
            success: false,
            error: error.message,
            duration: Date.now() - startTime
        })
    } finally {
        downloading.value = false
    }
}

const testEnhancedDownload = async () => {
    if (!testForm.value.filePath || !testForm.value.fileName) {
        ElMessage.error('Please fill in file path and name')
        return
    }

    downloading.value = true
    downloadProgress.value.active = true
    downloadProgress.value.canCancel = true
    downloadProgress.value.statusText = 'Starting download...'

    const startTime = Date.now()

    try {
        const result = await EnhancedDownloadService.downloadFileEnhanced(
            testForm.value.filePath,
            testForm.value.fileName,
            parseInt(testForm.value.fileSize) || 0,
            {
                onProgress: (downloaded, total, percentage, speed) => {
                    downloadProgress.value.downloaded = downloaded
                    downloadProgress.value.total = total
                    downloadProgress.value.percentage = percentage
                    downloadProgress.value.speed = speed
                    downloadProgress.value.statusText = `Downloading... ${percentage}%`
                },
                onComplete: (downloadId) => {
                    downloadProgress.value.statusText = 'Download completed!'
                    downloadProgress.value.status = 'success'
                    downloadProgress.value.canCancel = false
                },
                onError: (downloadId, error) => {
                    downloadProgress.value.statusText = `Error: ${error.message}`
                    downloadProgress.value.status = 'exception'
                    downloadProgress.value.canCancel = false
                }
            }
        )

        currentDownloadId.value = result.downloadId
        const duration = Date.now() - startTime

        testResults.value.unshift({
            id: Date.now(),
            type: 'Enhanced Download',
            fileName: testForm.value.fileName,
            success: result.success,
            error: result.error,
            duration,
            size: parseInt(testForm.value.fileSize) || 0
        })

        if (result.success) {
            ElMessage.success('Enhanced download completed successfully')
        } else {
            ElMessage.error(`Enhanced download failed: ${result.error}`)
        }

    } catch (error) {
        ElMessage.error(`Download error: ${error.message}`)
        downloadProgress.value.statusText = `Error: ${error.message}`
        downloadProgress.value.status = 'exception'
    } finally {
        downloading.value = false
        setTimeout(() => {
            downloadProgress.value.active = false
            downloadProgress.value.canCancel = false
        }, 3000)
    }
}

const cancelDownload = () => {
    if (currentDownloadId.value) {
        EnhancedDownloadService.cancelDownload(currentDownloadId.value)
        downloadProgress.value.statusText = 'Download cancelled'
        downloadProgress.value.status = 'warning'
        downloadProgress.value.canCancel = false
        ElMessage.warning('Download cancelled')
    }
}

const downloadSample = (sample) => {
    testForm.value.filePath = sample.path
    testForm.value.fileName = sample.path.split('/').pop()
    testForm.value.fileSize = sample.size === '1KB' ? 1024 :
        sample.size === '10MB' ? 10485760 :
            sample.size === '1GB' ? 1073741824 : 0
    testBasicDownload()
}

// Initialize
onMounted(() => {
    compatibility.value = DownloadService.checkCompatibility()
})
</script>

<style scoped>
.download-test-page {
    min-height: 100vh;
    background: var(--el-bg-color-page);
    padding: 20px;
}

.container {
    max-width: 800px;
    margin: 0 auto;
}

.compatibility-card,
.test-card,
.progress-card,
.results-card,
.samples-card {
    margin-bottom: 20px;
}

.compatibility-info {
    display: flex;
    flex-direction: column;
    gap: 8px;
}

.compat-item {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px;
    border-radius: 4px;
}

.compat-item.success {
    background: var(--el-color-success-light-9);
    color: var(--el-color-success);
}

.compat-item.warning {
    background: var(--el-color-warning-light-9);
    color: var(--el-color-warning);
}

.warnings {
    margin-top: 16px;
    padding: 12px;
    background: var(--el-color-warning-light-9);
    border-radius: 4px;
}

.progress-info {
    margin: 16px 0;
}

.progress-info p {
    margin: 4px 0;
    font-size: 14px;
    color: var(--el-text-color-regular);
}

.test-result {
    border: 1px solid;
    border-radius: 8px;
    padding: 12px;
    margin-bottom: 12px;
}

.test-result.success {
    border-color: var(--el-color-success);
    background: var(--el-color-success-light-9);
}

.test-result.error {
    border-color: var(--el-color-error);
    background: var(--el-color-error-light-9);
}

.result-header {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
}

.result-details p {
    margin: 4px 0;
    font-size: 13px;
}

.sample-files {
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.sample-file {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
}

.sample-file code {
    flex: 1;
    background: var(--el-color-info-light-9);
    padding: 4px 8px;
    border-radius: 4px;
    font-family: 'Monaco', 'Consolas', monospace;
}

.sample-size {
    min-width: 60px;
    text-align: center;
    font-weight: 500;
    color: var(--el-color-primary);
}
</style>
