#!/usr/bin/env node
/**
 * This script synchronizes versions from Git tags to package.json files
 * when there are no commits after the last tag.
 * 
 * It will:
 * 1. Check if there are commits after the last tag
 * 2. If not, update the respective package.json files with versions from tags
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

        console.log(`Updating ${packagePath} version from ${packageJson.version} to ${version}`);
        packageJson.version = version;

        fs.writeFileSync(
            packageJsonPath,
            JSON.stringify(packageJson, null, 4) + '\n'
        );
        return true;
    } catch (error) {
        console.error(`Failed to update ${packagePath}: ${error.message}`);
        return false;
    }
}

// Main execution
function main() {
    console.log('Checking Git versions and package.json files...');

    // Get version info for different components
    const globalVersionInfo = getVersionInfo('v*');
    const frontendVersionInfo = getVersionInfo('web/v*');
    const backendVersionInfo = getVersionInfo('app/v*');

    console.log('\nVersion information:');
    console.log(`- Global: ${globalVersionInfo.version || 'N/A'} (${globalVersionInfo.hasCommitsAfterTag ? 'has commits after tag' : 'clean tag'})`);
    console.log(`- Frontend: ${frontendVersionInfo.version || 'N/A'} (${frontendVersionInfo.hasCommitsAfterTag ? 'has commits after tag' : 'clean tag'})`);
    console.log(`- Backend: ${backendVersionInfo.version || 'N/A'} (${backendVersionInfo.hasCommitsAfterTag ? 'has commits after tag' : 'clean tag'})`);

    // Only update if there are no commits after tags
    const updates = [];

    if (!globalVersionInfo.hasCommitsAfterTag && globalVersionInfo.version) {
        const updated = updatePackageJson('package.json', globalVersionInfo.version);
        if (updated) updates.push('Root package.json');
    }

    if (!frontendVersionInfo.hasCommitsAfterTag && frontendVersionInfo.version) {
        const updated = updatePackageJson('web/package.json', frontendVersionInfo.version);
        if (updated) updates.push('Frontend package.json');
    }

    if (updates.length > 0) {
        console.log(`\n✅ Successfully updated versions in: ${updates.join(', ')}`);
    } else {
        console.log('\n🔄 No package.json files needed updating.');
    }
}

// Execute the main function
main();