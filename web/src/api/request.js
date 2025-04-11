import axios from "axios";
import { getConfig } from '../config/config';
import router from '../router';

const config = getConfig();

// Create a centralized axios instance with authentication support
// TODO: Consider consolidating with api.js to avoid duplication of HTTP client setup
const instance = axios.create({
    baseURL: config.apiBaseUrl,
    timeout: config.apiTimeout || 30000,
    // Enable credentials for cross-domain authentication
    withCredentials: true
});

// Handle authentication failures and session expiration gracefully
instance.interceptors.response.use(
    response => response,
    async error => {
        const originalRequest = error.config;

        // Detect unauthorized requests to initiate re-authentication
        // Prevents multiple redirects for the same authentication failure
        if (error.response?.status === 401 && !originalRequest._redirect) {
            originalRequest._redirect = true;

            // Notify application-wide listeners about authentication issues
            window.dispatchEvent(new CustomEvent('auth:token-expired'));

            // Preserve current navigation context for post-login return
            const currentPath = router.currentRoute.value.fullPath;
            if (currentPath !== '/login') {
                router.push({
                    path: '/login',
                    query: { redirect: currentPath }
                });
            }
        }

        return Promise.reject(error);
    }
);

// Provide simplified access to common HTTP methods
export const get = (url, config = {}) => {
    return instance.get(url, config);
};

export const post = (data, url, config = {}) => {
    return instance.post(url, data, config);
};

export const put = (data, url, config = {}) => {
    return instance.put(url, data, config);
};

export const del = (url, config = {}) => {
    return instance.delete(url, config);
};

export default instance;
