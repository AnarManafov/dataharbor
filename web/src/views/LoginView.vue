<!-- src/views/LoginView.vue -->
<template>
    <div class="login-view">
        <div class="login-container">
            <div class="login-header">
                <div class="logo-section">
                    <img src="/assets/dataharbor-logo.svg" alt="DataHarbor" class="logo" />
                    <h1>DataHarbor</h1>
                </div>
                <p class="login-subtitle">Sign in to access your files</p>
            </div>

            <el-card class="login-card" shadow="hover">
                <div v-if="loading" class="loading-section">
                    <el-icon class="loading-icon">
                        <Loading />
                    </el-icon>
                    <p class="loading-text">Authenticating...</p>
                </div>
                <div v-else class="login-content">
                    <h2>Welcome Back</h2>
                    <p class="login-description">
                        Please sign in with your GSI account to access the file browser and other protected resources.
                    </p>

                    <div class="login-form">
                        <el-button type="primary" :loading="loading" size="large" class="login-button"
                            @click="handleLogin">
                            <el-icon class="button-icon">
                                <User />
                            </el-icon>
                            Sign in with GSI Account
                        </el-button>

                        <el-alert v-if="errorMessage" :title="errorMessage" type="error" show-icon class="error-alert"
                            :closable="false" />
                    </div>

                    <div class="help-section">
                        <p>Need help? <a href="#" class="help-link">Contact Support</a></p>
                    </div>
                </div>
            </el-card>

            <div class="login-footer">
                <p>&copy; 2025 DataHarbor. All rights reserved.</p>
            </div>
        </div>
    </div>
</template>

<script>
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import useAuth from '../composables/useAuth';
import { Loading, User } from '@element-plus/icons-vue';

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
            loading
        };
    }
}
</script>

<style scoped>
.login-view {
    min-height: 100vh;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 2rem;
}

.login-container {
    width: 100%;
    max-width: 450px;
}

.login-header {
    text-align: center;
    margin-bottom: 2rem;

    .logo-section {
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 1rem;
        margin-bottom: 1rem;

        .logo {
            width: 48px;
            height: 48px;
        }

        h1 {
            color: white;
            font-size: 2rem;
            font-weight: 700;
            margin: 0;
            text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.3);
        }
    }

    .login-subtitle {
        color: rgba(255, 255, 255, 0.9);
        font-size: 1.1rem;
        font-weight: 300;
        margin: 0;
    }
}

.login-card {
    border-radius: 16px;
    border: none;
    box-shadow: 0 20px 40px rgba(0, 0, 0, 0.1);
    overflow: hidden;
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
    padding: 3rem 2rem;

    h2 {
        color: var(--el-text-color-primary);
        font-size: 1.75rem;
        font-weight: 600;
        text-align: center;
        margin-bottom: 1rem;
    }

    .login-description {
        color: var(--el-text-color-regular);
        text-align: center;
        line-height: 1.6;
        margin-bottom: 2rem;
    }
}

.login-form {
    margin-bottom: 2rem;

    .login-button {
        width: 100%;
        height: 48px;
        font-size: 1rem;
        font-weight: 500;
        border-radius: 8px;
        transition: all 0.3s ease;

        .button-icon {
            margin-right: 0.5rem;
        }

        &:hover {
            transform: translateY(-2px);
            box-shadow: 0 8px 20px rgba(0, 0, 0, 0.15);
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

.login-footer {
    text-align: center;
    margin-top: 2rem;

    p {
        color: rgba(255, 255, 255, 0.7);
        font-size: 0.9rem;
        margin: 0;
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
        .logo-section h1 {
            font-size: 1.5rem;
        }

        .login-subtitle {
            font-size: 1rem;
        }
    }

    .login-content {
        padding: 2rem 1.5rem;

        h2 {
            font-size: 1.5rem;
        }
    }
}
</style>