// colorPlugin.js
export default {
    install(app, options) {
        // Define your custom colors
        const appColors = {
            // offline - red
            offline: 'var(--el-color-danger)',
            // online - green 
            online: 'var(--el-color-success)',
        };

        // Make the colors available globally
        app.config.globalProperties.$app_colors = appColors;
    }
};
