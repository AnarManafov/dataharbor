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

app.mount("#app");