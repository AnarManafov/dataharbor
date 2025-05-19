import streamSaver from 'streamsaver'

// Configure StreamSaver for your HTTPS environment
// Since you use HTTPS even in dev, we can use the official mitm
streamSaver.mitm = 'https://jimmywarting.github.io/StreamSaver.js/mitm.html'

/**
 * Service for handling file downloads using StreamSaver.js
 * Optimized for large files (KB to GB) over WAN connections
 */
export class DownloadService {
    /**
     * Download a file with basic streaming
     * @param {string} filePath - Full path to the file on XRootD
     * @param {string} fileName - Name for the downloaded file
     * @param {number} estimatedSize - File size in bytes for progress indication
     * @returns {Promise<{success: boolean, error?: string, speed?: {mbps: number, bytesPerSec: number, duration: number}}>}
     */
    static async downloadFile(filePath, fileName, estimatedSize = 0) {
        try {
            console.log(`Starting download: ${fileName} (${estimatedSize} bytes)`)

            // Track download timing for speed calculation
            const startTime = Date.now()
            let bytesReceived = 0

            // Create writable stream with size for progress bar
            const fileStream = streamSaver.createWriteStream(fileName, {
                size: estimatedSize > 0 ? estimatedSize : undefined, // Only set if we have size
                writableStrategy: undefined, // Use default
                readableStrategy: undefined  // Use default
            })

            // Fetch from your existing streaming endpoint
            const downloadUrl = `/api/v1/xrd/download?path=${encodeURIComponent(filePath)}`

            console.log(`Fetching from: ${downloadUrl}`)

            const response = await fetch(downloadUrl, {
                method: 'GET',
                credentials: 'include', // Include cookies for authentication
                headers: {
                    'Accept': 'application/octet-stream',
                }
            })

            if (!response.ok) {
                throw new Error(`Download failed: ${response.status} ${response.statusText}`)
            }

            // Check if response has content-length header
            const contentLength = response.headers.get('content-length')
            if (contentLength && estimatedSize === 0) {
                console.log(`Server provided content-length: ${contentLength}`)
                bytesReceived = parseInt(contentLength)
            } else {
                bytesReceived = estimatedSize
            }

            // Pipe response directly to StreamSaver
            // This creates a true streaming download without loading into memory
            await response.body.pipeTo(fileStream)

            // Calculate download speed
            const endTime = Date.now()
            const duration = (endTime - startTime) / 1000 // seconds
            const bytesPerSec = bytesReceived / duration
            const mbps = bytesPerSec / (1024 * 1024)

            const speedInfo = {
                mbps: parseFloat(mbps.toFixed(2)),
                bytesPerSec: Math.round(bytesPerSec),
                duration: parseFloat(duration.toFixed(2)),
                totalBytes: bytesReceived
            }

            console.log(`Download completed successfully: ${fileName}`, speedInfo)
            return { success: true, speed: speedInfo }

        } catch (error) {
            console.error('Download failed:', error)

            // Provide user-friendly error messages
            let userMessage = error.message
            if (error.name === 'AbortError') {
                userMessage = 'Download was cancelled'
            } else if (error.message.includes('Failed to fetch')) {
                userMessage = 'Network error - please check your connection'
            } else if (error.message.includes('404')) {
                userMessage = 'File not found on server'
            } else if (error.message.includes('403')) {
                userMessage = 'Access denied - please check your permissions'
            }

            return { success: false, error: userMessage }
        }
    }

    /**
     * Download file with progress tracking and retry capability
     * Prepared for future resume download functionality
     * @param {string} filePath - Full path to the file on XRootD
     * @param {string} fileName - Name for the downloaded file
     * @param {number} estimatedSize - File size in bytes
     * @param {function} onProgress - Progress callback (downloaded, total)
     * @param {object} options - Additional options
     * @returns {Promise<{success: boolean, error?: string, speed?: {mbps: number, bytesPerSec: number, duration: number}}>}
     */
    static async downloadFileWithProgress(filePath, fileName, estimatedSize = 0, onProgress = null, options = {}) {
        const {
            maxRetries = 3,
            retryDelay = 1000,
            chunkSize = 64 * 1024, // 64KB chunks for progress reporting
            resumeSupport = false   // Prepared for future implementation
        } = options

        let attempt = 0

        while (attempt < maxRetries) {
            // Track download timing for speed calculation
            const startTime = Date.now()

            try {
                console.log(`Download attempt ${attempt + 1}/${maxRetries}: ${fileName}`)

                // Create the file stream
                const fileStream = streamSaver.createWriteStream(fileName, {
                    size: estimatedSize > 0 ? estimatedSize : undefined
                })

                // Prepare headers for potential resume support
                const headers = {
                    'Accept': 'application/octet-stream',
                }

                // Future: Add Range header for resume functionality
                // if (resumeSupport && startByte > 0) {
                //   headers['Range'] = `bytes=${startByte}-`
                // }

                const response = await fetch(`/api/v1/xrd/download?path=${encodeURIComponent(filePath)}`, {
                    method: 'GET',
                    credentials: 'include',
                    headers
                })

                if (!response.ok) {
                    throw new Error(`HTTP ${response.status}: ${response.statusText}`)
                }

                // Get actual file size from response
                const contentLength = response.headers.get('content-length')
                const totalSize = contentLength ? parseInt(contentLength) : estimatedSize

                // If no progress callback, use simple pipe
                if (!onProgress) {
                    await response.body.pipeTo(fileStream)

                    // Calculate download speed for simple pipe
                    const endTime = Date.now()
                    const duration = (endTime - startTime) / 1000
                    const bytesPerSec = totalSize / duration
                    const mbps = bytesPerSec / (1024 * 1024)

                    const speedInfo = {
                        mbps: parseFloat(mbps.toFixed(2)),
                        bytesPerSec: Math.round(bytesPerSec),
                        duration: parseFloat(duration.toFixed(2)),
                        totalBytes: totalSize
                    }

                    return { success: true, speed: speedInfo }
                }

                // Manual streaming for progress tracking
                const reader = response.body.getReader()
                const writer = fileStream.getWriter()

                let downloaded = 0
                let lastProgressTime = Date.now()

                try {
                    while (true) {
                        const { done, value } = await reader.read()

                        if (done) break

                        // Write chunk
                        await writer.write(value)
                        downloaded += value.length

                        // Report progress (throttled to avoid UI spam)
                        const now = Date.now()
                        if (now - lastProgressTime > 100) { // Update every 100ms max
                            onProgress(downloaded, totalSize)
                            lastProgressTime = now
                        }
                    }

                    // Final progress update
                    onProgress(downloaded, totalSize)

                } finally {
                    writer.close()
                }

                // Calculate download speed for manual streaming
                const endTime = Date.now()
                const duration = (endTime - startTime) / 1000
                const bytesPerSec = downloaded / duration
                const mbps = bytesPerSec / (1024 * 1024)

                const speedInfo = {
                    mbps: parseFloat(mbps.toFixed(2)),
                    bytesPerSec: Math.round(bytesPerSec),
                    duration: parseFloat(duration.toFixed(2)),
                    totalBytes: downloaded
                }

                console.log(`Download completed: ${fileName} (${downloaded} bytes)`, speedInfo)
                return { success: true, speed: speedInfo }

            } catch (error) {
                attempt++
                console.error(`Download attempt ${attempt} failed:`, error)

                if (attempt >= maxRetries) {
                    return {
                        success: false,
                        error: `Download failed after ${maxRetries} attempts: ${error.message}`
                    }
                }

                // Wait before retry
                await new Promise(resolve => setTimeout(resolve, retryDelay * attempt))
            }
        }
    }

    /**
     * Check browser compatibility and setup
     * @returns {object} Compatibility info
     */
    static checkCompatibility() {
        const isSecure = window.isSecureContext
        const hasServiceWorker = 'serviceWorker' in navigator
        const hasStreams = 'ReadableStream' in window && 'WritableStream' in window

        return {
            isSecure,
            hasServiceWorker,
            hasStreams,
            optimal: isSecure && hasServiceWorker && hasStreams,
            warnings: [
                ...(!isSecure ? ['Not in secure context - may impact performance'] : []),
                ...(!hasServiceWorker ? ['Service Worker not available - reduced performance'] : []),
                ...(!hasStreams ? ['Streams not available - polyfill active'] : [])
            ]
        }
    }

    /**
     * Cancel an ongoing download (for future use)
     * @param {string} downloadId - Download identifier
     */
    static cancelDownload(downloadId) {
        // Future implementation for download cancellation
        console.log(`Cancel download: ${downloadId}`)
    }
}

// Check compatibility on load
const compatibility = DownloadService.checkCompatibility()
if (compatibility.warnings.length > 0) {
    console.warn('StreamSaver compatibility warnings:', compatibility.warnings)
}

export default DownloadService
