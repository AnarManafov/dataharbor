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

export default {
    name: 'App',
    components: {
        GlobalSidebar
    },
    setup() {
        // Initialize auth at the root level to ensure auth state is available throughout the application
        const auth = useAuth();

        // No need to return auth as we're using it for initialization only
        // For components needing auth functionality, they should call useAuth() directly
        return {};
    }
};
</script>

<style lang="scss">
#app {
    font-family: Avenir, Helvetica, Arial, sans-serif;
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