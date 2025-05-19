import 'element-plus/dist/index.css';
import './styles/theme.css';

// Load stream polyfill for browser compatibility
import './services/streamPolyfill.js';

import { createApp } from "vue";
import { createPinia } from "pinia";
import store from './store';

import App from "./App.vue";
import router from "./router";
import colorPlugin from './plugins/colorPlugin';
import axios from "axios";
import VueAxios from "vue-axios";
import { setConfig } from './config/config';

const app = createApp(App);

app.use(createPinia());
app.use(store);
app.use(router);
app.use(VueAxios, axios);

// Register color constants for consistent theming across components
app.use(colorPlugin);

app.config.errorHandler = (err, vm, info) => {
    console.error('Error:', err);
    console.error('Vue component:', vm);
    console.error('Additional info:', info);
};

// Allow configuration to be customized per environment
const configPath = import.meta.env.VITE_CONFIG_FILE_PATH || '/config.json';
console.log('Loading configuration from:', configPath);

// Try environment-specific config first, with fallback to default
fetch(configPath)
    .then(response => {
        if (!response.ok) {
            throw new Error(`Failed to load config from ${configPath}: ${response.status}`);
        }
        return response.json();
    })
    .catch(error => {
        console.warn(`Error loading config from ${configPath}:`, error);
        console.log('Falling back to default config at /config.json');
        // Fallback to default path
        return fetch('/config.json').then(response => response.json());
    })
    .then(config => {
        console.log('Config loaded successfully:', { ...config, oidc: { ...config.oidc, clientSecret: config.oidc?.clientSecret ? '***' : undefined } });

        // Apply the loaded configuration
        setConfig(config);
        app.config.globalProperties.$config = config;

        // Mount the app immediately - router will handle authentication checks
        app.mount("#app");

        // Register global filters for formatting data consistently across the app
        app.config.globalProperties.$filters = {
            prettyBytes(num) {
                if (typeof num !== 'number' || isNaN(num)) {
                    throw new TypeError('Expected a number');
                }
                const units = ['B', 'kB', 'MB', 'GB', 'TB', 'PB', 'EB', 'ZB', 'YB'];
                const exponent = num === 0 ? 0 : Math.floor(Math.log(num) / Math.log(1000));
                const size = (num / Math.pow(1000, exponent)).toFixed(2);
                return `${size} ${units[exponent]}`;
            }
        };
    })
    .catch(error => {
        console.error('Fatal error loading configuration:', error);
        document.body.innerHTML = `<div style="padding: 20px; color: red;">
            <h1>Configuration Error</h1>
            <p>Failed to load application configuration. Please contact support.</p>
            <pre>${error.message}</pre>
        </div>`;
    });