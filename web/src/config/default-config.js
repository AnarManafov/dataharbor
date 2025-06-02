/**
 * Default configuration values used when external config is not available
 * Provides fallback values for development or when deployment-specific config is missing
 */
export default {
    // Base URL for API calls
    apiBaseUrl: '/api',

    // API request timeout in milliseconds (30 seconds)
    apiTimeout: 30000,

    // Authentication settings
    auth: {
        // Fallback to direct endpoint for BFF pattern
        redirectUrl: '/api/auth/callback',
    },

    // Feature flags
    features: {
        enableDocumentation: true,
        enableFileDownload: true,
    },

    // UI customization
    ui: {
        appTitle: 'DataHarbor',
        initialPageSize: 100,
    }
};