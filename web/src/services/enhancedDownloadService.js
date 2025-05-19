import streamSaver from 'streamsaver'
import { DownloadService } from './downloadService'

/**
 * Enhanc      const response = await fetch(`/api/v1/xrd/download?path=${encodeURIComponent(filePath)}`, {d download service with progress tracking, retry, and resume capabilities
 * Designed for production use with large files over WAN
 */
export class EnhancedDownloadService {
    static downloadManager = null
    static activeDownloads = new Map()

    /**
     * Set the download manager component reference
     */
    static setDownloadManager(manager) {
        this.downloadManager = manager
    }

    /**
     * Download file with full progress tracking and retry support
     * Prepared for future resume functionality
     */
    static async downloadFileEnhanced(filePath, fileName, fileSize = 0, options = {}) {
        const downloadId = `download_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`

        const {
            maxRetries = 3,
            retryDelay = 2000,
            resumeSupport = false, // Future feature
            onProgress = null,
            onComplete = null,
            onError = null
        } = options

        // Add to download manager if available
        if (this.downloadManager) {
            this.downloadManager.addDownload({
                id: downloadId,
                fileName,
                filePath,
                size: fileSize
            })
        }

        // Track the download
        const downloadInfo = {
            id: downloadId,
            filePath,
            fileName,
            size: fileSize,
            status: 'preparing',
            abortController: new AbortController(),
            retryCount: 0
        }

        this.activeDownloads.set(downloadId, downloadInfo)

        try {
            const result = await this._executeDownload(downloadInfo, onProgress)

            if (result.success) {
                this._updateDownloadStatus(downloadId, {
                    status: 'completed',
                    progress: 100
                })
                onComplete?.(downloadId)
                return { success: true, downloadId }
            } else {
                throw new Error(result.error)
            }
        } catch (error) {
            this._updateDownloadStatus(downloadId, {
                status: 'error',
                error: error.message
            })
            onError?.(downloadId, error)
            return { success: false, error: error.message, downloadId }
        }
    }

    /**
     * Execute the actual download with streaming and progress tracking
     */
    static async _executeDownload(downloadInfo, onProgress) {
        const { id, filePath, fileName, size, abortController } = downloadInfo

        try {
            this._updateDownloadStatus(id, { status: 'downloading', progress: 0 })

            // Create StreamSaver stream
            const fileStream = streamSaver.createWriteStream(fileName, {
                size: size > 0 ? size : undefined
            })

            // Prepare headers for potential resume support
            const headers = {
                'Accept': 'application/octet-stream',
            }

            // Future: Add resume support
            // if (resumeSupport && downloadInfo.resumeFrom > 0) {
            //   headers['Range'] = `bytes=${downloadInfo.resumeFrom}-`
            // }

            const response = await fetch(`/api/xrd/download?path=${encodeURIComponent(filePath)}`, {
                method: 'GET',
                credentials: 'include',
                headers,
                signal: abortController.signal
            })

            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`)
            }

            // Get actual file size
            const contentLength = response.headers.get('content-length')
            const totalSize = contentLength ? parseInt(contentLength) : size

            // Update download info with actual size
            this._updateDownloadStatus(id, { size: totalSize })

            // Check if we can use simple streaming (no progress needed)
            if (!onProgress && !this.downloadManager) {
                await response.body.pipeTo(fileStream)
                return { success: true }
            }

            // Manual streaming for progress tracking
            const reader = response.body.getReader()
            const writer = fileStream.getWriter()

            let downloaded = 0
            let lastProgressTime = Date.now()
            const startTime = Date.now()

            try {
                while (true) {
                    // Check for cancellation
                    if (abortController.signal.aborted) {
                        throw new Error('Download cancelled by user')
                    }

                    const { done, value } = await reader.read()

                    if (done) break

                    // Write chunk
                    await writer.write(value)
                    downloaded += value.length

                    // Calculate progress
                    const progress = totalSize > 0 ? Math.round((downloaded / totalSize) * 100) : 0

                    // Update progress (throttled)
                    const now = Date.now()
                    if (now - lastProgressTime > 250) { // Update every 250ms
                        const elapsed = (now - startTime) / 1000
                        const speed = elapsed > 0 ? downloaded / elapsed : 0

                        this._updateDownloadStatus(id, {
                            downloaded,
                            progress,
                            speed
                        })

                        onProgress?.(downloaded, totalSize, progress, speed)
                        lastProgressTime = now
                    }
                }

                // Final progress update
                this._updateDownloadStatus(id, {
                    downloaded,
                    progress: 100,
                    speed: 0
                })

            } finally {
                await writer.close()
            }

            return { success: true }

        } catch (error) {
            if (error.name === 'AbortError' || error.message.includes('cancelled')) {
                this._updateDownloadStatus(id, { status: 'cancelled' })
                return { success: false, error: 'Download cancelled' }
            }
            throw error
        }
    }

    /**
     * Cancel an active download
     */
    static cancelDownload(downloadId) {
        const download = this.activeDownloads.get(downloadId)
        if (download && download.abortController) {
            download.abortController.abort()
            this._updateDownloadStatus(downloadId, { status: 'cancelled' })
            this.activeDownloads.delete(downloadId)
            return true
        }
        return false
    }

    /**
     * Retry a failed download
     */
    static async retryDownload(downloadId) {
        const download = this.activeDownloads.get(downloadId)
        if (download && download.status === 'error') {
            download.retryCount++
            if (download.retryCount <= 3) {
                download.abortController = new AbortController()
                return this._executeDownload(download)
            }
        }
        return { success: false, error: 'Maximum retries exceeded' }
    }

    /**
     * Update download status in manager
     */
    static _updateDownloadStatus(downloadId, updates) {
        const download = this.activeDownloads.get(downloadId)
        if (download) {
            Object.assign(download, updates)
        }

        if (this.downloadManager) {
            this.downloadManager.updateDownload(downloadId, updates)
        }
    }

    /**
     * Get all active downloads
     */
    static getActiveDownloads() {
        return Array.from(this.activeDownloads.values())
    }

    /**
     * Check if a file is currently being downloaded
     */
    static isFileBeingDownloaded(filePath) {
        return Array.from(this.activeDownloads.values()).some(
            download => download.filePath === filePath && download.status === 'downloading'
        )
    }

    /**
     * Download multiple files sequentially (for future use)
     */
    static async downloadMultipleFiles(files, options = {}) {
        const results = []

        for (const file of files) {
            if (options.concurrent) {
                // Future: implement concurrent downloads
                results.push(this.downloadFileEnhanced(file.path, file.name, file.size, options))
            } else {
                // Sequential downloads
                const result = await this.downloadFileEnhanced(file.path, file.name, file.size, options)
                results.push(result)

                if (!result.success && options.stopOnError) {
                    break
                }
            }
        }

        return options.concurrent ? Promise.all(results) : results
    }

    /**
     * Resume download capability (prepared for future implementation)
     */
    static async resumeDownload(downloadId, resumeFrom = 0) {
        // Future implementation
        console.log(`Resume download ${downloadId} from byte ${resumeFrom}`)
        return { success: false, error: 'Resume functionality not yet implemented' }
    }

    /**
     * Get download statistics
     */
    static getDownloadStats() {
        const downloads = this.getActiveDownloads()

        return {
            total: downloads.length,
            active: downloads.filter(d => d.status === 'downloading').length,
            completed: downloads.filter(d => d.status === 'completed').length,
            failed: downloads.filter(d => d.status === 'error').length,
            cancelled: downloads.filter(d => d.status === 'cancelled').length,
            totalBytes: downloads.reduce((sum, d) => sum + (d.size || 0), 0),
            downloadedBytes: downloads.reduce((sum, d) => sum + (d.downloaded || 0), 0)
        }
    }
}

export default EnhancedDownloadService
