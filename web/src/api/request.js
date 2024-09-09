import axios from "axios";
import { getConfig } from '../config';

const config = getConfig();

const instance = axios.create({
    baseURL: config.apiBaseUrl,
    timeout: config.apiTimeout,
});

instance.interceptors.request.use(
    function (config) {
        return config;
    },
    function (error) {
        return Promise.reject(error);
    }
);

export default instance;
