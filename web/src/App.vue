<template>
    <div id="app">
        <Nav />
        <div class="container is-fluid">
            <router-view />
        </div>
    </div>
</template>

<script>
import Nav from './components/partials/Nav.vue';
import useAuth from './composables/useAuth';

export default {
    name: 'App',
    components: {
        Nav
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
}

.container {
    margin-top: 20px;
}

.centered {
    display: flex;
    justify-content: center;
    align-items: center;
}
</style>