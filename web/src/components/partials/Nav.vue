<template>
    <!--FIXME: is-transparent doesn't work. Find out why. -->
    <nav class="navbar is-transparent" role="navigation" aria-label="main navigation">
        <div class="navbar-brand no-hover">
            <a class="navbar-item is-size-4 has-text-weight-bold" href="/">
                <img src="/assets/brand.png">
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
                        <el-button v-if="!isAuthenticated" type="primary" @click="handleLogin">
                            Log In
                        </el-button>
                        <div v-else class="user-profile">
                            <span class="welcome-message">Welcome, {{ userName }}!</span>
                            <el-button type="danger" size="small" @click="showLogoutConfirmation">
                                Logout
                            </el-button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </nav>
</template>
<script>
import { version } from '../../../package.json';
import { useAuth } from '../../composables/useAuth';
import { computed } from 'vue';
import { ElMessageBox } from 'element-plus';
import { useRouter } from 'vue-router';

export default {
    name: 'Nav',
    setup() {
        const { isAuthenticated, user, logout, login } = useAuth();
        const router = useRouter();

        const handleLogin = () => {
            router.push('/login');
        };

        const showLogoutConfirmation = () => {
            ElMessageBox.confirm(
                'Are you sure you want to log out?',
                'Confirm Logout',
                {
                    confirmButtonText: 'Logout',
                    cancelButtonText: 'Cancel',
                    type: 'warning'
                }
            ).then(async () => {
                try {
                    await logout();
                } catch (error) {
                    console.error('Logout failed:', error);
                }
            }).catch(() => {
                // User canceled the logout action, do nothing
            });
        };

        return {
            isAuthenticated,
            showLogoutConfirmation,
            handleLogin,
            // Try multiple standard OIDC claims for the user's name
            // Order of preference: given_name, name, preferred_username, email, sub
            userName: computed(() => {
                if (!user.value) return 'User';

                // Check for given name first (first name)
                if (user.value.given_name) {
                    return user.value.given_name;
                }

                // Full name is next best option
                if (user.value.name) {
                    return user.value.name;
                }

                // Username is third choice
                if (user.value.preferred_username) {
                    return user.value.preferred_username;
                }

                // Email as a fallback
                if (user.value.email) {
                    return user.value.email.split('@')[0]; // Just the part before @
                }

                // Subject ID as last resort
                if (user.value.sub) {
                    return user.value.sub;
                }

                // If nothing is available, use "User"
                return 'User';
            })
        };
    },
    data() {
        return {
            appVersion: version
        };
    },
    mounted() {
        // Find all burger menu toggles for responsive design
        const $navbarBurgers = Array.prototype.slice.call(document.querySelectorAll('.navbar-burger'), 0);

        // Attach click handlers to each burger menu
        $navbarBurgers.forEach(el => {
            el.addEventListener('click', () => {

                // Retrieve the target menu from the data-target attribute
                const target = el.dataset.target;
                const $target = document.getElementById(target);

                // Toggle active state on both the burger button and the menu
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

.user-profile {
    display: flex;
    align-items: center;
}

.welcome-message {
    margin-right: 8px;
}
</style>
