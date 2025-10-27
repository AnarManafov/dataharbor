import { fileURLToPath, URL } from "node:url";
import path from "node:path";
import { execSync } from "child_process";
import fs from "fs";

import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";
import { getHttpsConfig } from "./cert-config.js";

// Helper functions to get version information directly (no external imports)
function getVersion(tagPattern, fallbackPath) {
    try {
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
            return commits && hash ? `${version}-${hash}` : version;
        }

        console.warn(`Could not parse version from: ${gitDescribe}`);
        return getPackageVersion(fallbackPath);
    } catch (error) {
        console.warn(`Failed to get version for ${tagPattern}: ${error.message}`);
        return getPackageVersion(fallbackPath);
    }
}

function getPackageVersion(packagePath) {
    try {
        const packageJsonPath = path.resolve(process.cwd(), packagePath);
        const packageJson = JSON.parse(fs.readFileSync(packageJsonPath, 'utf8'));
        return packageJson.version || '0.0.0';
    } catch (error) {
        console.warn(`Failed to read ${packagePath}: ${error.message}`);
        return '0.0.0';
    }
}

// Get versions
const globalVersion = getVersion("v*", "../package.json");
const frontendVersion = getVersion("web/v*", "./package.json");
const backendVersion = getVersion("app/v*", "../app/package.json");

console.log(`Building with versions:
- Global: ${globalVersion}
- Frontend: ${frontendVersion}
- Backend: ${backendVersion}`);

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
    resolve: {
        alias: {
            "@": fileURLToPath(new URL("./src", import.meta.url)),
        },
    },
    css: {
        preprocessorOptions: {
            scss: {
                // this resolves the warning: The legacy JS API is deprecated and will be removed in Dart Sass 2.0.0.
                api: 'modern-compiler', // or 'modern'
            },
        },
    },
    define: {
        '__APP_VERSION__': JSON.stringify(frontendVersion),
        '__GLOBAL_VERSION__': JSON.stringify(globalVersion),
        '__BACKEND_VERSION__': JSON.stringify(backendVersion),
    },
    server: {
        host: '0.0.0.0', // Bind to all interfaces for Docker
        port: 5173,
        https: getHttpsConfig(),
        proxy: {
            // Proxy all /api requests to backend server (now HTTPS)
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
    }
});
