import axios from "axios";

const instance = axios.create({
    // baseURL: import.meta.env.VITE_BASE_URL,
    baseURL: "http://localhost:22000/",
    timeout: 5000,
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
