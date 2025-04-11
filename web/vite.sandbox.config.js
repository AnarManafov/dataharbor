import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';
import path from 'path';
import { fileURLToPath } from 'url';
import AutoImport from "unplugin-auto-import/vite";
import Components from "unplugin-vue-components/vite";
import { ElementPlusResolver } from "unplugin-vue-components/resolvers";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

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
        // Proxy API requests to your backend during development
        proxy: {
            '/api': {
                target: 'http://localhost:22000',
                changeOrigin: true,
                secure: false,
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
