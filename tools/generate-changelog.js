#!/usr/bin/env node
/**
 * Generates a changelog based on Git commits between the current and previous version.
 * This script can be used as part of the release process to automatically document changes.
 * 
 * Usage: node tools/generate-changelog.js [tag] [previous-tag]
 */

import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const ROOT_DIR = path.resolve(__dirname, '..');

// Parse command line arguments
const newTag = process.argv[2] || 'HEAD';
let previousTag = process.argv[3];

if (!previousTag) {
    // If no previous tag is specified, get the latest tag
    try {
        previousTag = execSync('git describe --tags --abbrev=0', {
            encoding: 'utf8',
            stdio: ['pipe', 'pipe', 'ignore']
        }).trim();
    } catch (error) {
        console.error('Error getting previous tag:', error.message);
        process.exit(1);
    }
}

// Type definitions for categorizing commits
const COMMIT_TYPES = {
    feat: { title: '✨ New Features', order: 1 },
    fix: { title: '🐛 Bug Fixes', order: 2 },
    perf: { title: '⚡️ Performance Improvements', order: 3 },
    refactor: { title: '♻️ Code Refactoring', order: 4 },
    style: { title: '💄 UI and Style Changes', order: 5 },
    docs: { title: '📝 Documentation', order: 6 },
    test: { title: '✅ Tests', order: 7 },
    build: { title: '👷 Build System', order: 8 },
    ci: { title: '🔧 CI Configuration', order: 9 },
    chore: { title: '🔨 Chores and Maintenance', order: 10 },
    other: { title: '🔄 Other Changes', order: 11 }
};

/**
 * Gets commits between two Git references
 */
function getCommits(from, to = 'HEAD') {
    try {
        const gitLogCommand = `git log ${from}..${to} --pretty=format:"%h|%s|%an" --no-merges`;
        const output = execSync(gitLogCommand, {
            encoding: 'utf8',
            stdio: ['pipe', 'pipe', 'ignore']
        }).trim();

        if (!output) return [];

        return output.split('\n').map(line => {
            const [hash, subject, author] = line.split('|');

            // Try to parse the conventional commit type
            let type = 'other';
            const match = subject.match(/^(\w+)(?:\(([^\)]+)\))?:\s*(.+)$/);
            if (match) {
                const [, commitType] = match;
                if (COMMIT_TYPES[commitType]) {
                    type = commitType;
                }
            }

            return { hash, subject, author, type };
        });
    } catch (error) {
        console.error('Error getting commits:', error.message);
        return [];
    }
}

/**
 * Generates a changelog as markdown
 */
function generateChangelog(commits) {
    if (commits.length === 0) {
        return '> No changes found for this version.';
    }

    // Group commits by type
    const groupedCommits = {};
    Object.keys(COMMIT_TYPES).forEach(type => {
        groupedCommits[type] = [];
    });

    commits.forEach(commit => {
        if (groupedCommits[commit.type]) {
            groupedCommits[commit.type].push(commit);
        } else {
            groupedCommits.other.push(commit);
        }
    });

    // Generate markdown
    let markdown = '';

    // Sort types by order
    const sortedTypes = Object.keys(COMMIT_TYPES)
        .sort((a, b) => COMMIT_TYPES[a].order - COMMIT_TYPES[b].order)
        .filter(type => groupedCommits[type].length > 0);

    sortedTypes.forEach(type => {
        if (groupedCommits[type].length === 0) return;

        markdown += `\n## ${COMMIT_TYPES[type].title}\n\n`;

        groupedCommits[type].forEach(commit => {
            const shortHash = commit.hash.substring(0, 7);
            markdown += `- ${commit.subject} ([${shortHash}](${getCommitUrl(shortHash)}))\n`;
        });

        markdown += '\n';
    });

    return markdown.trim();
}

/**
 * Gets the commit URL for the current repo
 */
function getCommitUrl(hash) {
    try {
        const remoteUrl = execSync('git config --get remote.origin.url', { encoding: 'utf8' }).trim();
        const githubMatch = remoteUrl.match(/github\.com[:/]([^\/]+)\/([^\/\.]+)/);

        if (githubMatch) {
            const [, owner, repo] = githubMatch;
            return `https://github.com/${owner}/${repo.replace('.git', '')}/commit/${hash}`;
        }

        return `#${hash}`;
    } catch (error) {
        return `#${hash}`;
    }
}

// Main execution
console.log(`Generating changelog from ${previousTag} to ${newTag}...`);
const commits = getCommits(previousTag, newTag);
console.log(`Found ${commits.length} commits.`);

const changelog = generateChangelog(commits);

// If we're generating for a specific tag release
if (newTag !== 'HEAD') {
    // Strip the 'v' prefix if present
    const version = newTag.startsWith('v') ? newTag.substring(1) : newTag;
    const today = new Date().toISOString().split('T')[0]; // YYYY-MM-DD

    const header = `# ${newTag} (${today})\n`;
    const fullChangelog = header + '\n' + changelog;

    console.log('\nChangelog:');
    console.log(fullChangelog);

    // You can uncomment these lines to automatically update CHANGELOG.md
    /*
    const changelogPath = path.join(ROOT_DIR, 'CHANGELOG.md');
    let existingChangelog = '';
    
    try {
      existingChangelog = fs.readFileSync(changelogPath, 'utf8');
    } catch (error) {
      // File doesn't exist yet, create it
      existingChangelog = '# Changelog\n\nAll notable changes to this project will be documented in this file.\n\n';
    }
    
    // Add the new changelog at the top, after the header
    const newChangelog = existingChangelog.replace(/# Changelog\n\n/, `# Changelog\n\n${fullChangelog}\n\n`);
    
    fs.writeFileSync(changelogPath, newChangelog, 'utf8');
    console.log(`\nChangelog written to ${changelogPath}`);
    */
} else {
    // Just output to console
    console.log('\nChangelog:');
    console.log(changelog);
}

// Output the changelog so it can be captured by GitHub Actions
console.log('\nGENERATED_CHANGELOG<<EOF');
console.log(changelog);
console.log('EOF');