# File Download System

This document describes the streaming file download system implementation for DataHarbor, designed to efficiently handle large files over WAN connections.

## Overview

The download system leverages modern web streaming APIs to provide efficient file downloads directly from XRootD storage without consuming browser memory. The implementation is optimized for large files (KB to GB range) and provides real-time progress tracking with speed calculation.

## Technology Stack

### Current Implementation

- **StreamSaver.js** - Primary streaming download library providing broad browser compatibility
- **Web Streams API** - Native browser streaming with polyfill fallback for older browsers
- **Service Worker** - Background processing for optimal download performance
- **XRootD ReadAt API** - Backend streaming using native XRootD client for efficient data transfer

### Future Migration Path

**Native File System Access API**: The system is prepared for migration to the native File System Access API once Safari implements support. This will eliminate the need for StreamSaver.js and provide direct file system integration. An adapter layer will be implemented to maintain compatibility during the transition period.

## Architecture

### Core Components

1. **DownloadService** (`/src/services/downloadService.js`)
   - Primary download interface using StreamSaver.js
   - Browser compatibility checks and automatic fallback handling
   - Basic error handling and user feedback
   - Speed calculation and reporting

2. **EnhancedDownloadService** (`/src/services/enhancedDownloadService.js`)
   - Advanced download management with progress tracking
   - Retry mechanisms with exponential backoff
   - Cancellation support and cleanup
   - Prepared for resume functionality using Range requests

3. **DownloadManager** (`/src/components/DownloadManager.vue`)
   - UI component for managing multiple downloads
   - Progress visualization and user controls
   - Download queue management interface

4. **Stream Polyfill** (`/src/services/streamPolyfill.js`)
   - Automatic polyfill loading for browser compatibility
   - Graceful degradation for unsupported browsers

### Backend Integration

The frontend integrates with the existing XRootD API endpoints:

- **Download Endpoint**: `/api/v1/xrd/download` - Streaming file download with speed logging
- **Authentication**: Session-based authentication using HTTP-only cookies
- **Range Support**: Backend prepared for resume downloads with `Accept-Ranges: bytes` header

## Design Principles

### Memory Efficiency

The system maintains constant memory usage (~64KB) regardless of file size by using streaming APIs rather than buffering entire files in memory. This enables downloading files larger than available RAM.

### Browser Compatibility

- **Chrome/Edge 76+**: Native streaming support with optimal performance
- **Firefox 65+**: Streams API with polyfill for missing features
- **Safari 14+**: StreamSaver.js with Service Worker fallback
- **Older Browsers**: Graceful degradation with basic download functionality

### Security Model

- **Same-Origin Policy**: Downloads use existing authentication cookies
- **Path Validation**: Backend validates all file paths to prevent directory traversal
- **Rate Limiting**: Single download per user session to prevent resource abuse
- **XRootD Permissions**: File access controlled by existing XRootD permission system

## Performance Characteristics

### Expected Metrics

- **Memory Usage**: Constant ~64KB regardless of file size
- **Download Speed**: Network and XRootD server limited, with real-time calculation
- **CPU Usage**: Minimal overhead (< 5% during streaming)
- **Browser Responsiveness**: Non-blocking downloads with progress updates

### Speed Tracking

Both frontend and backend implement speed calculation:

- **Backend**: Logs download speed, duration, and throughput statistics
- **Frontend**: Displays real-time speed in UI messages upon completion
- **Format**: Speed displayed as MB/s with total transfer time and file size

## Error Handling Strategy

The system implements comprehensive error handling:

- **Network Issues**: Automatic retry with exponential backoff (configurable)
- **Authentication Failures**: Clear error messages with redirect to login
- **File Not Found**: User-friendly 404 handling with path validation
- **Cancellation**: Clean abort without partial file artifacts
- **Browser Compatibility**: Automatic fallback for unsupported features

## Future Enhancements

### Phase 2: Advanced Features (Prepared)

1. **Resume Downloads**
   - Backend already supports HTTP Range requests
   - Frontend prepared for resume capability using byte offsets
   - Automatic resume after network interruption

2. **Concurrent Downloads**
   - Multiple file downloads with configurable concurrency limits
   - Bandwidth management and priority queuing
   - Smart scheduling for optimal performance

3. **Native File System Integration**
   - Migration to File System Access API when Safari support available
   - Direct file system writes without Service Worker dependency
   - Improved performance and reduced complexity

### Phase 3: Enterprise Features

1. **Download Queue Management**
   - Priority-based download scheduling
   - Bandwidth throttling and fair usage policies
   - Download history and retry mechanisms

2. **Offline Support**
   - Background downloads using Service Worker
   - Download resume after browser restart
   - Persistent download queue across sessions

## Configuration

### StreamSaver Setup

The system uses the official StreamSaver.js MITM (Man-in-the-Middle) service worker for cross-browser compatibility. For production deployments, consider hosting the MITM worker locally for improved security and performance.

### Backend Optimization

The XRootD download endpoint provides optimal streaming performance through:

- **Chunked Transfer Encoding**: 512KB buffer size for efficient streaming
- **Content-Length Headers**: Accurate file size for progress calculation
- **Accept-Ranges Support**: Prepared for resume download functionality
- **Speed Logging**: Server-side performance monitoring and statistics

## Testing Strategy

### Browser Compatibility Testing

Regular testing across supported browsers ensures consistent functionality:

- Automated tests for core download functionality
- Manual testing for UI responsiveness and error handling
- Performance testing with various file sizes and network conditions

### Performance Benchmarks

- Small files (< 1MB): Verify overhead is minimal
- Large files (> 100MB): Confirm memory usage remains constant
- Very large files (> 1GB): Validate streaming performance over WAN
- Slow connections: Test progress reporting and timeout handling

## Troubleshooting

### Common Issues

1. **Service Worker Registration Failures**
   - Ensure HTTPS is enabled for optimal Service Worker support
   - Check browser console for specific registration errors
   - Verify Content Security Policy allows Service Worker execution

2. **Download Speed Issues**
   - Monitor backend logs for XRootD performance metrics
   - Check network latency and bandwidth to XRootD server
   - Verify no rate limiting is affecting transfer speeds

3. **Memory Usage Problems**
   - Confirm streaming is working (check browser task manager)
   - Verify polyfills are loading correctly for older browsers
   - Check for JavaScript errors preventing streaming activation

### Debug Information

Enable detailed logging in development mode to diagnose issues:

- Browser console shows detailed download progress and timing
- Backend logs include transfer speeds and XRootD performance metrics
- Network tab in browser dev tools shows streaming behavior
