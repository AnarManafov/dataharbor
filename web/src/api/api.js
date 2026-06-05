import axios from 'axios';

// Support multiple deployment environments through configurable endpoint
const baseURL = import.meta.env.VITE_API_BASE_URL || '/api';

// Standardize API interactions with consistent configuration
// to prevent timeout and content-type inconsistencies
const apiClient = axios.create({
  baseURL,
  timeout: 30000, // 30s timeout prevents UI hanging during network issues
  headers: {
    'Content-Type': 'application/json',
    'Accept': 'application/json'
  }
});

/**
 * Normalize error handling across the application
 * to provide consistent user feedback regardless of error source
 */
function handleApiError(error) {
  // Handle network failures separately as they don't have response objects
  if (!error.response) {
    return Promise.reject({
      message: 'Network error - please check your connection',
      status: 0,
      data: null
    });
  }

  // Extract useful information from backend responses
  const { status, data } = error.response;

  let errorMessage = 'Unknown error occurred';

  // Handle 403 Forbidden (authorization errors)
  if (status === 403) {
    errorMessage = data?.message || data?.error || 'You are not authorized to access this resource. Please check your permissions.';
  }
  // Handle 401 Unauthorized (authentication errors)
  else if (status === 401) {
    errorMessage = data?.message || data?.error || 'Authentication required. Please log in.';
  }
  // Handle other errors
  else if (data && data.message) {
    errorMessage = data.message;
  } else if (data && data.error) {
    errorMessage = data.error;
  } else if (data && typeof data === 'string') {
    errorMessage = data;
  }

  return Promise.reject({
    message: errorMessage,
    status,
    data
  });
}

// XRootD file operations API

/**
 * Get server-configured starting directory to anchor user navigation
 * Ensures users begin browsing from an accessible and relevant location
 */
export function getInitialDirPath() {
  return apiClient.get('/v1/xrd/initialDir')
    .catch(handleApiError);
}

/**
 * List directory contents for user navigation
 * @param {string} path - Directory to explore
 */
export function getItemsInDir(path) {
  return apiClient.post('/v1/xrd/ls/paged', { path, page: 1, pageSize: 500 })
    .catch(handleApiError);
}

/**
 * Retrieve directory contents in chunks to optimize frontend performance
 * Essential for handling large directories without browser memory issues
 * @param {string} path - Directory to explore
 * @param {number} page - Pagination index starting at 1
 * @param {number} pageSize - Items per page
 */
export function getPagedItemsInDir(path, page, pageSize) {
  return apiClient.post('/v1/xrd/ls/paged', {
    path,
    page,
    pageSize
  }).catch(handleApiError);
}

/**
 * Download file directly from XRootD via streaming endpoint
 * Returns the streaming download URL without requiring file staging
 * @param {string} path - File path to download
 */
export function getStreamingDownloadUrl(path) {
  // Return the URL for the streaming download endpoint
  return `${baseURL}/v1/xrd/download?path=${encodeURIComponent(path)}`;
}

/**
 * Get the batch download endpoint URL for multi-file tar.gz streaming
 * Used by downloadService.js with fetch() for streaming
 */
export function getBatchDownloadUrl() {
  return `${baseURL}/v1/xrd/download/batch`;
}

// -----------------------------------------------------------------------
// Upload API
// -----------------------------------------------------------------------

/**
 * Fetch configured transfer limits (upload + download).
 * Populates the Transfer Limits info popover in the toolbar.
 */
export function getTransferLimits() {
  return apiClient.get('/v1/xrd/upload/limits').catch(handleApiError);
}

/**
 * Create a multi-file upload session. The server allocates one uploadId per
 * accepted file and returns conflict status for each.
 *
 * @param {{destDir:string, files:Array<{relPath:string,size:number,sha256:string,onConflict?:string}>}} payload
 */
export function createUploadSession(payload) {
  return apiClient.post('/v1/xrd/upload/session', payload).catch(handleApiError);
}

/**
 * Upload a single chunk of bytes for a given uploadId.
 *
 * Uses a longer timeout than the default apiClient timeout because chunks can
 * be up to ~8 MiB and travel over WAN. A per-request AbortController signal
 * must be supplied so the uploadService can pause/cancel in-flight transfers.
 *
 * @param {string} uploadId
 * @param {number} offset
 * @param {Blob|ArrayBuffer} chunk
 * @param {{signal?:AbortSignal, onUploadProgress?:Function}} [opts]
 */
export function uploadChunk(uploadId, offset, chunk, opts = {}) {
  return apiClient.put(
    `/v1/xrd/upload/${encodeURIComponent(uploadId)}/chunk`,
    chunk,
    {
      params: { offset },
      headers: { 'Content-Type': 'application/octet-stream' },
      // Chunks can be large; disable the short default timeout.
      timeout: 0,
      signal: opts.signal,
      onUploadProgress: opts.onUploadProgress,
      // Let the browser set Content-Length from the Blob/ArrayBuffer.
      transformRequest: [(data) => data],
    }
  ).catch(handleApiError);
}

/**
 * Finalize an upload: the server verifies SHA-256 and atomically publishes the
 * file under its destination path.
 */
export function completeUpload(uploadId) {
  return apiClient.post(
    `/v1/xrd/upload/${encodeURIComponent(uploadId)}/complete`,
    {}
  ).catch(handleApiError);
}

/** Abort a single in-progress upload (cleans up temp file and slot). */
export function abortUpload(uploadId) {
  return apiClient.delete(
    `/v1/xrd/upload/${encodeURIComponent(uploadId)}`
  ).catch(handleApiError);
}

/** Query the server-committed byte offset. Used by the resume flow. */
export function getUploadStatus(uploadId) {
  return apiClient.get(
    `/v1/xrd/upload/${encodeURIComponent(uploadId)}/status`
  ).catch(handleApiError);
}

/**
 * Retrieve storage system identification for user context awareness
 * Helps users understand which system they're currently accessing
 */
export function getHostName() {
  return apiClient.get('/v1/xrd/hostname')
    .catch(handleApiError);
}

/**
 * Retrieve virtual filesystem statistics (storage utilization, free space, etc.)
 * @param {string} path - Path prefix for filtering server/partition stats
 */
export function getVirtualFSStat(path = '/') {
  return apiClient.get('/v1/xrd/vstat', { params: { path } })
    .catch(handleApiError);
}

/**
 * Ping XRD server to measure round-trip latency
 * Returns latencyMs, status, and server hostname
 */
export function pingXrd() {
  return apiClient.get('/v1/xrd/ping')
    .catch(handleApiError);
}

/**
 * Verify backend service availability
 * Used for status indicators and service monitoring
 */
export function getBackendHealth() {
  return apiClient.get('/health')
    .catch(handleApiError);
}

// Authentication API

/**
 * Retrieve current user session data
 * Verifies authentication status and provides user context
 */
export function getUserInfo() {
  return apiClient.get('/auth/user', { withCredentials: true })
    .catch(handleApiError);
}

/**
 * Begin OAuth/OIDC authentication flow
 * Preserves user's navigation intent for post-login continuation
 * @param {string} redirectUri - Destination after successful login
 */
export function login(redirectUri) {
  return apiClient.get('/auth/login', {
    params: { redirect_uri: redirectUri },
    withCredentials: true
  }).catch(handleApiError);
}

/**
 * Terminate user session
 * Cleans up server session state and client-side authentication data
 */
export function logout() {
  return apiClient.post('/auth/logout', {}, { withCredentials: true })
    .catch(handleApiError);
}

export default apiClient;
