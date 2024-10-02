<template>
    <!--FIXME: is-transparent doesn't work. Find out why. -->
    <nav class="navbar is-transparent" role="navigation" aria-label="main navigation">
        <div class="navbar-brand no-hover">
            <a class="navbar-item is-size-4 has-text-weight-bold" href="/">
                <img src="../../assets/brand.png">
                <!--strong class="is-size-4">Data Lake UI</strong-->
            </a>
            <a role="button" class="navbar-burger burger" aria-label="menu" aria-expanded="true"
                data-target="navbarBasicExample">
                <span aria-hidden="true"></span>
                <span aria-hidden="true"></span>
                <span aria-hidden="true"></span>
            </a>
        </div>
        <div id="navbarBasicExample" class="navbar-menu">
            <div class="navbar-start">
                <router-link to="/browse" class="navbar-item">Browse files and folders</router-link>
                <router-link to="/documentation" class="navbar-item">Documentation</router-link>
                <div class="navbar-item has-dropdown is-hoverable">
                    <a class="navbar-link">
                        More
                    </a>
                    <div class="navbar-dropdown">
                        <router-link to="/about" class="navbar-item">
                            About
                        </router-link>
                        <a class="navbar-item">
                            Contact
                        </a>
                        <hr class="navbar-divider">
                        <a class="navbar-item">
                            Report an issue
                        </a>
                        <div class="navbar-item">
                            Version: {{ appVersion }}
                        </div>
                    </div>
                </div>
            </div>


            <div class="navbar-end">
                <div class="navbar-item">
                    <div class="buttons">
                        <router-link v-if="!isLoggedIn" to="/login" class="button is-dark is-outlined">
                            Log In
                        </router-link>
                        <span v-else>Welcome, {{ userName }}!</span>
                    </div>
                </div>
            </div>
        </div>
    </nav>
</template>
<script>
import { version } from '../../../package.json';
import { useAuth } from '../../composables/useAuth';

export default {
    name: 'Nav',
    setup() {
        const { isLoggedIn, userName } = useAuth();
        return {
            isLoggedIn,
            userName
        };
    },
    data() {
        return {
            appVersion: version
        };
    },
    mounted() {
        // Get all "navbar-burger" elements
        const $navbarBurgers = Array.prototype.slice.call(document.querySelectorAll('.navbar-burger'), 0);

        // Add a click event on each of them
        $navbarBurgers.forEach(el => {
            el.addEventListener('click', () => {

                // Get the target from the "data-target" attribute
                const target = el.dataset.target;
                const $target = document.getElementById(target);

                // Toggle the "is-active" class on both the "navbar-burger" and the "navbar-menu"
                el.classList.toggle('is-active');
                $target.classList.toggle('is-active');

            });
        });
    }
};
</script>
<style lang="scss" scoped>
nav {
    // margin-top: 10px;
    // margin-bottom: 10px;

    a {
        color: var(--el-color-text-primary);
        text-decoration: none;

        &.router-link-exact-active {
            background-color: transparent;
            font-weight: bold;
            color: var(--el-color-warning);
        }
    }
}

.no-hover .navbar-item:hover {
    background-color: transparent;
    /* Prevents background color change on hover */
    cursor: default;
    /* Changes cursor to default arrow */
}
</style>
