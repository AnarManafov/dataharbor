import path from 'path';
import fs from 'fs';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

/**
 * Certificate configuration utility for development/testing
 * Handles multiple potential locations for self-signed certificates
 */

/**
 * Helper function to generate certificate path objects
 * @param {string} basePath - The base path to resolve from
 * @param {string} keyPath - Relative or absolute path to the key file
 * @param {string} certPath - Relative or absolute path to the cert file
 * @param {string} source - Description of the certificate source
 * @returns {Object} Certificate path object
 */
function generateCertPaths(basePath, keyPath, certPath, source) {
    return {
        key: path.resolve(basePath, keyPath),
        cert: path.resolve(basePath, certPath),
        source: source
    };
}

/**
 * Find certificates in various potential locations
 * @returns {Object} Object with key and cert paths
 */
export function getCertPaths() {
    // Priority order for certificate locations:
    // 1. Environment variables (highest priority)
    // 2. PKM workspace location (for documentation/testing)
    // 3. Local app/config (current fallback)
    // 4. Relative PKM paths (for different PKM locations)

    const potentialLocations = [
        // Environment variables - highest priority
        {
            key: process.env.VITE_SSL_KEY,
            cert: process.env.VITE_SSL_CERT,
            source: 'environment variables'
        },

        // PKM workspace location (your specific case)
        generateCertPaths(__dirname, '../../pkm/docs/gsi/data-lake-ui/test/cert/server.key', '../../pkm/docs/gsi/data-lake-ui/test/cert/server.crt', 'PKM workspace (relative)'),

        // Common PKM locations on different platforms
        generateCertPaths(process.env.HOME || process.env.USERPROFILE || '', 'Documents/workspace/pkm/docs/gsi/data-lake-ui/test/cert/server.key', 'Documents/workspace/pkm/docs/gsi/data-lake-ui/test/cert/server.crt', 'PKM in user home'),

        // Original fallback location
        generateCertPaths(__dirname, '../app/config/server.key', '../app/config/server.crt', 'local app/config')
    ];

    for (const location of potentialLocations) {
        // Skip if environment variables are not set
        if (location.source === 'environment variables' && (!location.key || !location.cert)) {
            continue;
        }

        // Check if both files exist
        if (location.key && location.cert &&
            fs.existsSync(location.key) && fs.existsSync(location.cert)) {
            console.log(`🔒 Using SSL certificates from: ${location.source}`);
            console.log(`    Key: ${location.key}`);
            console.log(`    Cert: ${location.cert}`);
            return {
                key: location.key,
                cert: location.cert
            };
        }
    }

    // No certificates found
    console.warn('⚠️  No SSL certificates found in any of the expected locations:');
    potentialLocations.forEach((loc, index) => {
        if (loc.key && loc.cert) {
            console.warn(`   ${index + 1}. ${loc.source}: ${loc.key}`);
        }
    });
    console.warn('   Consider setting VITE_SSL_KEY and VITE_SSL_CERT environment variables');

    return null;
}

/**
 * Create HTTPS configuration object for Vite
 * @returns {Object|false} HTTPS config object or false to disable HTTPS
 */
export function getHttpsConfig() {
    const certPaths = getCertPaths();

    if (!certPaths) {
        console.warn('🔓 HTTPS disabled - no certificates found');
        return false;
    }

    return {
        key: certPaths.key,
        cert: certPaths.cert
    };
}
