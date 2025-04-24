import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

/**
 * Gets the global version from Git tags with format "vX.Y.Z"
 * Returns format: X.Y.Z or X.Y.Z-githash if there are commits after the tag
 */
export function getGlobalVersion() {
    try {
        // Get the git describe output for global tags (v*)
        const gitDescribe = execSync('git describe --tags --match "v*" --abbrev=7', {
            encoding: 'utf8',
            stdio: ['pipe', 'pipe', 'ignore']
        }).trim();

        // Parse the git describe output (format: v0.5.0-46-g83e4762 or v0.5.0)
        const match = gitDescribe.match(/^v?(\d+\.\d+\.\d+)(?:-(\d+)-g([a-f0-9]+))?$/);

        if (match) {
            const [, version, commits, hash] = match;
            // If there are commits after the tag, append the short hash
            if (commits && hash) {
                return `${version}-${hash}`;
            }
            return version;
        }

        console.warn('Could not parse global version from:', gitDescribe);
        return getPackageVersion('./package.json'); // Fall back to root package.json

    } catch (error) {
        console.warn('Failed to get global version:', error.message);
        return getPackageVersion('./package.json'); // Fall back to root package.json
    }
}

/**
 * Gets the frontend version from Git tags with format "web/vX.Y.Z"
 * Returns format: X.Y.Z or X.Y.Z-githash if there are commits after the tag
 */
export function getFrontendVersion() {
    try {
        // Get the git describe output for frontend tags (web/v*)
        const gitDescribe = execSync('git describe --tags --match "web/v*" --abbrev=7', {
            encoding: 'utf8',
            stdio: ['pipe', 'pipe', 'ignore']
        }).trim();

        // Parse the git describe output (format: web/v0.5.0-46-g83e4762 or web/v0.5.0)
        const match = gitDescribe.match(/^web\/v?(\d+\.\d+\.\d+)(?:-(\d+)-g([a-f0-9]+))?$/);

        if (match) {
            const [, version, commits, hash] = match;
            // If there are commits after the tag, append the short hash
            if (commits && hash) {
                return `${version}-${hash}`;
            }
            return version;
        }

        console.warn('Could not parse frontend version from:', gitDescribe);
        return getPackageVersion('./web/package.json'); // Fall back to web/package.json

    } catch (error) {
        console.warn('Failed to get frontend version:', error.message);
        return getPackageVersion('./web/package.json'); // Fall back to web/package.json
    }
}

/**
 * Gets the backend version from Git tags with format "app/vX.Y.Z"
 * Returns format: X.Y.Z or X.Y.Z-githash if there are commits after the tag
 */
export function getBackendVersion() {
    try {
        // Get the git describe output for backend tags (app/v*)
        const gitDescribe = execSync('git describe --tags --match "app/v*" --abbrev=7', {
            encoding: 'utf8',
            stdio: ['pipe', 'pipe', 'ignore']
        }).trim();

        // Parse the git describe output (format: app/v0.5.0-46-g83e4762 or app/v0.5.0)
        const match = gitDescribe.match(/^app\/v?(\d+\.\d+\.\d+)(?:-(\d+)-g([a-f0-9]+))?$/);

        if (match) {
            const [, version, commits, hash] = match;
            // If there are commits after the tag, append the short hash
            if (commits && hash) {
                return `${version}-${hash}`;
            }
            return version;
        }

        console.warn('Could not parse backend version from:', gitDescribe);
        return getPackageVersion('./app/package.json') || '0.0.0'; // Try app/package.json or default

    } catch (error) {
        console.warn('Failed to get backend version:', error.message);
        return getPackageVersion('./app/package.json') || '0.0.0'; // Try app/package.json or default
    }
}

/**
 * Generic function for getting the version from Git
 * This is maintained for backward compatibility
 */
export function getGitVersion() {
    // By default return the global version
    return getGlobalVersion();
}

/**
 * Gets the version from package.json as a fallback
 * @param {string} packagePath - Path to the package.json file
 */
function getPackageVersion(packagePath = './package.json') {
    try {
        const packageJsonPath = path.resolve(process.cwd(), packagePath);
        const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
        return packageJson.version || '0.0.0';
    } catch (error) {
        console.warn(`Failed to read ${packagePath}:`, error.message);
        return '0.0.0';
    }
}

// When run directly, output versions
if (process.argv[1] === fileURLToPath(import.meta.url)) {
    const command = process.argv[2];

    if (command === 'global') {
        console.log(getGlobalVersion());
    } else if (command === 'frontend') {
        console.log(getFrontendVersion());
    } else if (command === 'backend') {
        console.log(getBackendVersion());
    } else {
        console.log(`Global version: ${getGlobalVersion()}`);
        console.log(`Frontend version: ${getFrontendVersion()}`);
        console.log(`Backend version: ${getBackendVersion()}`);
    }
}