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
                        <!-- Show login button if not authenticated -->
                        <el-button v-if="!isAuthenticated" type="primary" @click="handleLogin">
                            <el-icon class="el-icon--left">
                                <User />
                            </el-icon>
                            <span>Log In</span>
                        </el-button>

                        <!-- Show user profile if authenticated -->
                        <div v-else class="user-profile-container">
                            <div class="user-profile">
                                <el-tooltip v-if="userEmail" effect="dark" :content="userEmail"
                                    placement="bottom-start">
                                    <el-avatar :size="40" :src="userAvatar" :icon="User" class="user-avatar"
                                        :aria-label="'User avatar for ' + (userFullName || 'unknown user')">
                                        {{ userInitials }}
                                    </el-avatar>
                                </el-tooltip>

                                <el-avatar v-else :size="40" :icon="User" class="user-avatar"
                                    :aria-label="'User avatar for ' + (userFullName || 'unknown user')">
                                    {{ userInitials }}
                                </el-avatar>

                                <div class="user-info">
                                    <div class="user-login">{{ userLogin }}</div>
                                    <div class="user-name">{{ userFullName }}</div>
                                </div>

                                <el-button type="danger" size="small" @click="showLogoutConfirmation"
                                    class="logout-btn">
                                    Logout
                                </el-button>
                            </div>
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
import { User } from '@element-plus/icons-vue';

export default {
    name: 'Nav',
    components: {
        User
    },
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

        // Get user login (username or ID)
        const userLogin = computed(() => {
            if (!user.value) return 'User';

            // Preferred username is the ideal login value
            if (user.value.preferred_username) {
                return user.value.preferred_username;
            }

            // Email username as a fallback
            if (user.value.email) {
                return user.value.email.split('@')[0]; // Just the part before @
            }

            // Subject ID as last resort
            if (user.value.sub) {
                return user.value.sub;
            }

            return 'User';
        });

        // Get user's full name
        const userFullName = computed(() => {
            if (!user.value) return '';

            // Full name is the best option for display name
            if (user.value.name) {
                return user.value.name;
            }

            // Combine given name and family name if available
            if (user.value.given_name && user.value.family_name) {
                return `${user.value.given_name} ${user.value.family_name}`;
            }

            // Given name only
            if (user.value.given_name) {
                return user.value.given_name;
            }

            // Fall back to username if no name is available
            if (user.value.preferred_username) {
                return user.value.preferred_username;
            }

            return '';
        });

        // Get user's email for tooltip
        const userEmail = computed(() => {
            if (!user.value) return '';
            return user.value.email || '';
        });

        // Get user's initials for avatar fallback
        const userInitials = computed(() => {
            if (!user.value) return '';

            // From full name
            if (user.value.name) {
                const names = user.value.name.split(' ');
                if (names.length >= 2) {
                    return (names[0][0] + names[1][0]).toUpperCase();
                } else if (names.length === 1 && names[0].length > 0) {
                    return names[0][0].toUpperCase();
                }
            }

            // From given name and family name
            if (user.value.given_name && user.value.family_name) {
                return (user.value.given_name[0] + user.value.family_name[0]).toUpperCase();
            }

            // From preferred username
            if (user.value.preferred_username) {
                return user.value.preferred_username[0].toUpperCase();
            }

            // From email
            if (user.value.email) {
                return user.value.email[0].toUpperCase();
            }

            return '';
        });

        // Get user's avatar URL if available
        const userAvatar = computed(() => {
            if (!user.value) return '';
            return user.value.picture || '';  // OpenID standard claim for avatar
        });

        return {
            isAuthenticated,
            showLogoutConfirmation,
            handleLogin,
            User,
            userLogin,
            userFullName,
            userEmail,
            userInitials,
            userAvatar,
            // Keep for backward compatibility
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

.user-profile-container {
    display: flex;
    align-items: center;
}

.user-profile {
    display: flex;
    align-items: center;
    padding: 4px 8px;
    border-radius: 4px;
    transition: background-color 0.2s;

    &:hover {
        background-color: rgba(0, 0, 0, 0.05);
    }
}

.user-avatar {
    margin-right: 12px;
    background-color: var(--el-color-primary);
    color: #fff;
    cursor: pointer;
}

.user-info {
    display: flex;
    flex-direction: column;
    margin-right: 16px;
}

.user-login {
    font-weight: bold;
    font-size: 14px;
}

.user-name {
    font-size: 12px;
    color: var(--el-text-color-secondary);
}

.logout-btn {
    margin-left: 8px;
}

/* Scope element-plus button styles to this component only */
.navbar-end .el-button {
    display: flex;
    align-items: center;
}

.navbar-end .el-icon--left {
    display: flex;
    margin-right: 4px;
}
</style>
