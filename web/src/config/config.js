import axios from 'axios';
import defaultConfig from './default-config';

// Configuration is a shared singleton across the application
// to ensure consistency and prevent duplicate loading
let config = null;

/**
 * Loads configuration from an external JSON file or uses default values
 * External config allows deployment-specific settings without rebuilding
 */
export async function loadConfig() {
    try {
        // Public directory is the right place for deployment-specific config
        // as it doesn't get bundled during build time
        const response = await axios.get('/config.json');
        console.log('Config loaded successfully:', response.data);
        config = response.data;
    } catch (error) {
        console.warn(
            'Could not load external configuration. Using defaults.',
            error.message
        );
        // Fall back to embedded defaults for development or when config isn't available
        config = { ...defaultConfig };
    }

    return config;
}

/**
 * Sets configuration explicitly, useful for testing or external control
 */
export function setConfig(newConfig) {
    config = newConfig;
    console.log('Setting config:', config);
}

/**
 * Returns the current application configuration
 * Creates default config if none has been loaded yet to prevent null errors
 */
export function getConfig() {
    if (!config) {
        config = { ...defaultConfig };
    }
    return config;
}
