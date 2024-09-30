<template>
    <div id="app">
        <Nav />
        <router-view />
    </div>
</template>

<script>
import Nav from './components/partials/Nav.vue';
import { provideAuth } from './composables/useAuth';

export default {
    name: 'app',
    components: {
        Nav,
    },
    setup() {
        provideAuth();
    }
};
</script>

<style lang="scss">
#app {
    font-family: 'Avenir', Helvetica, Arial, sans-serif;
    -webkit-font-smoothing: antialiased;
    -moz-osx-font-smoothing: grayscale;
    color: #2c3e50;
}
</style>