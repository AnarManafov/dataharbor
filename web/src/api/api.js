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
    if (data && data.message) {
        errorMessage = data.message;
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
    return apiClient.get('/xrd/initialDir')
        .catch(handleApiError);
}

/**
 * List directory contents for user navigation
 * @param {string} path - Directory to explore
 */
export function getItemsInDir(path) {
    return apiClient.post('/xrd/ls', { path })
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
    return apiClient.post('/xrd/ls/paged', {
        path,
        page,
        pageSize
    }).catch(handleApiError);
}

/**
 * Request file preparation for user download
 * Makes XRootD data accessible through HTTP by staging to web-accessible storage
 * @param {string} path - File to prepare for download
 */
export function getFileStagedForDownload(path) {
    return apiClient.post('/xrd/stage', { path })
        .catch(handleApiError);
}

/**
 * Retrieve storage system identification for user context awareness
 * Helps users understand which system they're currently accessing
 */
export function getHostName() {
    return apiClient.get('/xrd/hostname')
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