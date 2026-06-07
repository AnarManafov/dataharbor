<!-- src/views/LoginView.vue -->
<template>
    <div class="login-view">
        <div class="login-container">
            <div class="login-header">
                <div class="logo-section">
                    <img src="/assets/dataharbor-logo.svg" alt="DataHarbor" class="logo" />
                </div>
            </div>

            <div v-if="loading" class="loading-section">
                <el-icon class="loading-icon">
                    <Loading />
                </el-icon>
                <p class="loading-text">Authenticating...</p>
            </div>
            <div v-else class="login-content">
                <p class="login-description">
                    {{ branding.loginDescription }}
                </p>

                <div class="login-form">
                    <el-button type="primary" :loading="loading" size="large" class="login-button"
                        @click="handleLogin">
                        <el-icon class="button-icon">
                            <User />
                        </el-icon>
                        Sign in to access your files
                    </el-button>

                    <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon class="error-alert"
                        :closable="false" />
                </div>

                <div class="help-section">
                    <p>Need help? <a href="https://github.com/AnarManafov/dataharbor/issues" target="_blank"
                            rel="noopener noreferrer" class="help-link">Report an issue</a></p>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import useAuth from '../composables/useAuth';
import { Loading, User } from '@element-plus/icons-vue';
import { getConfig } from '@/config/config';

export default {
    name: 'LoginView',
    components: {
        Loading,
        User
    },
    setup() {
        const { login, isAuthenticated, error, isLoading } = useAuth();
        const route = useRoute();
        const router = useRouter();
        const config = getConfig();
        const branding = config.branding || {};

        // Store intended destination for post-login redirect
        const redirectPath = ref(route.query.redirect || '/');
        const errorMessage = ref('');
        const loading = ref(false);

        // Handle login button click
        const handleLogin = async () => {
            console.log('Login button clicked');
            loading.value = true;
            errorMessage.value = '';

            try {
                // Call the login method from useAuth composable
                await login();
                // The login method will redirect the browser, so we don't need to do anything else here
            } catch (err) {
                console.error('Login failed:', err);
                errorMessage.value = 'Failed to initiate login. Please try again.';
                loading.value = false;
            }
        };

        onMounted(() => {
            // If already authenticated, redirect to intended destination
            if (isAuthenticated.value) {
                router.push(redirectPath.value);
            }
        });

        return {
            handleLogin,
            errorMessage,
            loading,
            branding
        };
    }
}
</script>

<style scoped>
/* Fill the main content area (viewport minus the top bar) without exceeding it,
   so the global footer stays visible and the page doesn't scroll. */
.login-view {
    min-height: 100%;
    background: var(--el-bg-color-page);
    display: flex;
    justify-content: center;
    padding: 2rem;
    box-sizing: border-box;
}

/* Full-height column: logo pinned to top, button centered in the middle
   (via auto margins on .login-content), footer at the bottom. */
.login-container {
    width: 100%;
    max-width: 450px;
    display: flex;
    flex-direction: column;
    align-items: center;
}

.login-header {
    text-align: center;
    width: 100%;

    .logo-section {
        display: flex;
        align-items: center;
        justify-content: center;

        .logo {
            width: 400px;
            height: auto;
            max-width: 100%;
        }
    }
}

.loading-section {
    text-align: center;
    padding: 3rem 2rem;

    .loading-icon {
        font-size: 3rem;
        color: var(--el-color-primary);
        margin-bottom: 1rem;
        animation: rotating 2s linear infinite;
    }

    .loading-text {
        color: var(--el-text-color-regular);
        font-size: 1.1rem;
        margin: 0;
    }
}

.login-content {
    width: 100%;
    margin: auto 0;

    .login-description {
        color: var(--el-text-color-regular);
        text-align: center;
        line-height: 1.6;
        margin-bottom: 2rem;
    }
}

.login-form {
    margin-bottom: 2rem;
    text-align: center;

    /* Centered primary action styled to match the home "Open File Browser"
       CTA: blue gradient from the app palette, pill shape, theme-aware shadow. */
    .login-button {
        height: 52px;
        padding: 0 2.5rem;
        font-size: 1.1rem;
        font-weight: 600;
        border-radius: 50px;
        background: linear-gradient(135deg, var(--el-color-primary) 0%, var(--el-color-primary-dark-2) 100%);
        border: none;
        color: var(--el-color-white);
        box-shadow: var(--dh-shadow-md);
        transition: all 0.3s ease;

        .button-icon {
            margin-right: 0.6rem;
            font-size: 1.35em;
        }

        &:hover,
        &:focus {
            background: linear-gradient(135deg, var(--el-color-primary) 0%, var(--el-color-primary-dark-2) 100%);
            color: var(--el-color-white);
            transform: translateY(-2px);
            box-shadow: var(--dh-shadow-lg);
        }

        &:active {
            transform: translateY(0);
        }
    }

    .error-alert {
        margin-top: 1rem;
        border-radius: 8px;
    }
}

.help-section {
    text-align: center;
    padding-top: 1.5rem;
    border-top: 1px solid var(--el-border-color-light);

    p {
        color: var(--el-text-color-regular);
        font-size: 0.9rem;
        margin: 0;
    }

    .help-link {
        color: var(--el-color-primary);
        text-decoration: none;
        font-weight: 500;

        &:hover {
            text-decoration: underline;
        }
    }
}

@keyframes rotating {
    from {
        transform: rotate(0deg);
    }

    to {
        transform: rotate(360deg);
    }
}

@media (max-width: 768px) {
    .login-view {
        padding: 1rem;
    }

    .login-header {
        .logo-section .logo {
            width: 280px;
            height: auto;
        }
    }
}
</style>
