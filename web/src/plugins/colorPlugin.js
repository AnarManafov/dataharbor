// colorPlugin.js
export default {
    install(app, options) {
        // Define your custom colors
        const appColors = {
            // offline - red
            offline: '#C23A3A',
            // online - green 
            online: '#67C23A',
        };

        // Make the colors available globally
        app.config.globalProperties.$app_colors = appColors;
    }
};
