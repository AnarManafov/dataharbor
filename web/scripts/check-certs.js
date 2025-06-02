#!/usr/bin/env node

/**
 * Certificate Setup Script for Data Lake UI Development
 * 
 * This script helps developers set up SSL certificates for local development
 * by detecting and configuring certificate paths across different platforms.
 */

import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { getCertPaths } from '../cert-config.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

console.log('🔍 Data Lake UI - SSL Certificate Configuration\n');

// Check current certificate status
const certPaths = getCertPaths();

if (certPaths) {
    console.log('✅ SSL certificates found and ready to use!\n');
    console.log('🚀 You can now run:');
    console.log('   npm run dev        (regular development)');
    console.log('   npm run sandbox    (sandbox mode)\n');
} else {
    console.log('❌ No SSL certificates found.\n');
    console.log('📋 Setup options:\n');

    console.log('1️⃣  Use environment variables (recommended for cross-platform):');
    console.log('   export VITE_SSL_KEY="/path/to/your/server.key"');
    console.log('   export VITE_SSL_CERT="/path/to/your/server.crt"');
    console.log('   npm run dev\n');

    console.log('2️⃣  Use PKM-specific scripts (for your current setup):');
    console.log('   npm run dev:pkm-certs');
    console.log('   npm run sandbox:pkm-certs\n');

    console.log('3️⃣  Copy certificates to local app/config:');
    console.log('   cp /path/to/your/server.key ../app/config/');
    console.log('   cp /path/to/your/server.crt ../app/config/');
    console.log('   npm run dev\n');

    console.log('4️⃣  Create your own setup script:');
    console.log('   See web/scripts/setup-certs-example.sh for a template\n');

    // Exit with non-zero status to indicate failure in CI environments
    process.exit(1);
}

console.log('🔧 For more information, see: web/docs/ssl-setup.md');
