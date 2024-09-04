import 'bulma/css/bulma.css';

import { createApp } from "vue";
import { createPinia } from "pinia";

import App from "./App.vue";
import router from "./router";
import colorPlugin from './plugin/colorPlugin';
import axios from "axios";
import VueAxios from "vue-axios";


const app = createApp(App);

app.use(createPinia());
app.use(router);
app.use(VueAxios, axios);

// Using predefine color constants of this app
app.use(colorPlugin);

// Make a human ridable representation of bytes.
// Alternatively a package pretty-bytes can be used.
// It's home: https://github.com/sindresorhus/pretty-bytes
// NPM: `npm install pretty-bytes`
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

app.mount("#app");