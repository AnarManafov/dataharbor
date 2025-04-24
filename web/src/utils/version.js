/**
 * Version utilities for accessing dynamic version information
 * These values are injected by Vite at build time from Git tags
 */
import { APP_VERSION, GLOBAL_VERSION, BACKEND_VERSION } from '../scripts/version-info';

/**
 * Get the frontend application version (from web/v* tags)
 * @returns {string} Frontend version in format X.Y.Z or X.Y.Z-hash
 */
export function getAppVersion() {
    return APP_VERSION;
}

/**
 * Get the global project version (from v* tags)
 * @returns {string} Global version in format X.Y.Z or X.Y.Z-hash
 */
export function getGlobalVersion() {
    return GLOBAL_VERSION;
}

/**
 * Get the backend application version (from app/v* tags)
 * @returns {string} Backend version in format X.Y.Z or X.Y.Z-hash
 */
export function getBackendVersion() {
    return BACKEND_VERSION;
}

/**
 * Get all version information as an object
 * @returns {Object} Object containing all version types
 */
export function getAllVersions() {
    return {
        app: getAppVersion(),
        global: getGlobalVersion(),
        backend: getBackendVersion()
    };
}