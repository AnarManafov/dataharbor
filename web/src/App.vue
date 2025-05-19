<template>
    <div id="app">
        <GlobalSidebar>
            <router-view />
        </GlobalSidebar>
    </div>
</template>

<script>
import GlobalSidebar from './components/GlobalSidebar.vue';
import useAuth from './composables/useAuth';
import { onMounted } from 'vue';

export default {
    name: 'App',
    components: {
        GlobalSidebar
    },
    setup() {
        // Initialize auth at the root level to ensure auth state is available throughout the application
        const { checkAuth } = useAuth();

        // Check auth status when the app is mounted
        // Skip initial auth check on login page to avoid unnecessary API calls and potential redirect loops
        onMounted(() => {
            if (window.location.pathname !== '/login') {
                checkAuth();
            }
        });

        // No need to return auth as we're using it for initialization only
        // For components needing auth functionality, they should call useAuth() directly
        return {};
    }
};
</script>

<style lang="scss">
#app {
    font-family: var(--dh-font-family);
    font-size: var(--dh-font-size-base);
    line-height: var(--dh-line-height-normal);
    font-weight: var(--dh-font-weight-normal);
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    color: #2c3e50;
    height: 100vh;
    overflow: hidden;
}

.centered {
    display: flex;
    justify-content: center;
    align-items: center;
}
</style>