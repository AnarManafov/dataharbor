import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import path from 'path';
import { fileURLToPath } from 'url';
import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";
import fs from 'fs';
import { getHttpsConfig } from "./cert-config.js";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Function to ensure sandbox has all required files
function ensureSandboxFiles() {
    const publicDir = path.resolve(__dirname, 'public');
    const sandboxDir = path.resolve(__dirname, '../sandbox/public');

    // Create sandbox directory if it doesn't exist
    if (!fs.existsSync(sandboxDir)) {
        fs.mkdirSync(sandboxDir, { recursive: true });
    }

    // Create assets directory if it doesn't exist
    const sandboxAssetsDir = path.join(sandboxDir, 'assets');
    if (!fs.existsSync(sandboxAssetsDir)) {
        fs.mkdirSync(sandboxAssetsDir, { recursive: true });
    }

    // Copy required files from public to sandbox if they don't exist
    const filesToEnsure = [
        'config.json',
        'silent-renew.html',
        'assets/brand.png',
        'assets/favicon.ico',
        'assets/norway-4970080_1280.jpg'
    ];

    filesToEnsure.forEach(file => {
        const sourcePath = path.join(publicDir, file);
        const destPath = path.join(sandboxDir, file);

        if (fs.existsSync(sourcePath) && !fs.existsSync(destPath)) {
            // Make sure the directory exists
            const destDir = path.dirname(destPath);
            if (!fs.existsSync(destDir)) {
                fs.mkdirSync(destDir, { recursive: true });
            }

            // Copy the file
            fs.copyFileSync(sourcePath, destPath);
            console.log(`Copied ${file} to sandbox`);
        }
    });

    console.log('Sandbox environment is ready');
}

// Ensure sandbox is prepared before starting the dev server
ensureSandboxFiles();

// https://vitejs.dev/config/
export default defineConfig({
    plugins: [
        vue(),
        AutoImport({
            resolvers: [ElementPlusResolver()],
        }),
        Components({
            resolvers: [ElementPlusResolver()],
        }),
    ],
    // Use the sandbox's public directory
    publicDir: path.resolve(__dirname, '../sandbox/public'),
    // Configure the development server
    server: {
        port: 5173,
        https: getHttpsConfig(),
        // Proxy API requests to your backend during development
        proxy: {
            '/api': {
                target: 'https://localhost:22000',
                changeOrigin: true,
                secure: false, // Allow self-signed certificates in development
                ws: true,
                xfwd: true,
                // Allow cookies to be sent cross-domain
                cookieDomainRewrite: {
                    '*': '' // Remove domain restrictions from cookies
                },
                // Preserve cookies from backend to frontend
                configure: (proxy, _options) => {
                    proxy.on('proxyRes', (proxyRes, req, res) => {
                        const cookies = proxyRes.headers['set-cookie'];
                        if (cookies) {
                            // Rewrite the set-cookie headers to work with the frontend domain
                            const newCookies = cookies.map(cookie =>
                                cookie
                                    .replace(/Domain=[^;]+/, '') // Remove domain restriction
                                    .replace(/SameSite=None/, 'SameSite=Lax') // Switch to Lax mode for development
                            );
                            proxyRes.headers['set-cookie'] = newCookies;
                        }
                    });
                },
            }
        }
    },
    // Add resolve aliases to match your main Vite config
    resolve: {
        alias: {
            '@': path.resolve(__dirname, './src'),
            // Add any other aliases your project might be using
        }
    },
    css: {
        preprocessorOptions: {
            scss: {
                // Resolve the warning: The legacy JS API is deprecated
                api: 'modern-compiler',
            },
        },
    }
});
