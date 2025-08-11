#!/usr/bin/env node
/**
 * This script synchronizes versions from Git tags to package.json files.
 * 
 * It will:
 * 1. Get version information from Git tags
 * 2. Update the respective package.json files with versions from tags
 * 
 * Usage: node sync-versions.js
 */

import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT_DIR = path.resolve(__dirname, '..');

/**
 * Gets version information for a specific tag pattern
 * @param {string} tagPattern - The pattern to match tags (e.g., "v*" or "web/v*")
 * @returns {Object} Version info object { version, hasCommitsAfterTag }
 */
function getVersionInfo(tagPattern) {
    try {
        // Get the git describe output
        const gitDescribe = execSync(`git describe --tags --match "${tagPattern}" --abbrev=7`, {
            encoding: 'utf8',
            stdio: ['pipe', 'pipe', 'ignore']
        }).trim();

        // Parse the git describe output
        const prefix = tagPattern.replace('*', '');
        const pattern = new RegExp(`^${prefix}?([0-9]+\\.[0-9]+\\.[0-9]+)(?:-([0-9]+)-g([a-f0-9]+))?$`);
        const match = gitDescribe.match(pattern);

        if (match) {
            const [, version, commits, hash] = match;
            const hasCommitsAfterTag = Boolean(commits && hash);

            return {
                version,
                hasCommitsAfterTag,
                rawOutput: gitDescribe
            };
        }

        console.warn(`Could not parse version from: ${gitDescribe}`);
        return { version: null, hasCommitsAfterTag: true, rawOutput: gitDescribe };
    } catch (error) {
        console.warn(`Failed to get version for ${tagPattern}: ${error.message}`);
        return { version: null, hasCommitsAfterTag: true, error: error.message };
    }
}

/**
 * Updates a package.json file with the new version if needed
 * @param {string} packagePath - Path to the package.json file
 * @param {string} version - Version to set
 * @returns {boolean} True if file was updated, false otherwise
 */
function updatePackageJson(packagePath, version) {
    if (!version) {
        console.log(`No valid version found for ${packagePath}, skipping update.`);
        return false;
    }

    try {
        const packageJsonPath = path.resolve(ROOT_DIR, packagePath);
        const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));

        if (packageJson.version === version) {
            console.log(`Package ${packagePath} already has correct version ${version}, no update needed.`);
            return false;
        }

        // Update version and write to file
        packageJson.version = version;
        fs.writeFileSync(packageJsonPath, JSON.stringify(packageJson, null, 2) + '\n');
        console.log(`Updated ${packagePath} to version ${version}`);
        return true;
    } catch (error) {
        console.error(`Failed to update ${packagePath}: ${error.message}`);
        return false;
    }
}

/**
 * Main execution function
 */
function main() {
    // Get version info for different components
    const globalVersionInfo = getVersionInfo('v*');
    const frontendVersionInfo = getVersionInfo('web/v*');
    const backendVersionInfo = getVersionInfo('app/v*');

    console.log('\nVersion information:');
    console.log(`- Global: ${globalVersionInfo.version || 'unknown'}${globalVersionInfo.hasCommitsAfterTag ? '' : ' (clean tag)'}`);
    console.log(`- Frontend: ${frontendVersionInfo.version || 'unknown'}${frontendVersionInfo.hasCommitsAfterTag ? '' : ' (clean tag)'}`);
    console.log(`- Backend: ${backendVersionInfo.version || 'unknown'}${backendVersionInfo.hasCommitsAfterTag ? '' : ' (clean tag)'}`);

    const updates = [];

    // Always update package.json files if we have a valid version, regardless of commits after tag
    if (globalVersionInfo.version) {
        const updated = updatePackageJson('package.json', globalVersionInfo.version);
        if (updated) updates.push('Root package.json');
    }

    if (frontendVersionInfo.version) {
        const updated = updatePackageJson('web/package.json', frontendVersionInfo.version);
        if (updated) updates.push('Frontend package.json');
    }

    // Update package-lock.json files if any package.json was updated
    if (updates.length) {
        console.log(`\n[INFO] Updated ${updates.length} package.json files: ${updates.join(', ')}`);
        console.log('\n[INFO] Updating package-lock.json files...');

        try {
            // Run npm install to update package-lock.json files
            execSync('npm install', { encoding: 'utf8', stdio: 'inherit' });
            console.log('[SUCCESS] Successfully updated package-lock.json files');
        } catch (error) {
            console.error('[ERROR] Failed to update package-lock.json files:', error.message);
        }
    } else {
        console.log(`\n[INFO] No package.json files needed updating.`);
    }
}

// Run the main function
main();