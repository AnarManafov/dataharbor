<!-- src/views/LoginView.vue -->
<template>
    <div class="login-view">
        <el-card class="login-card">
            <template #header>
                <div class="card-header">
                    <h3>Login</h3>
                </div>
            </template>

            <div v-if="loading" class="text-center">
                <el-icon class="loading-icon">
                    <Loading />
                </el-icon>
                <p class="loading-text">Authenticating...</p>
            </div>
            <div v-else>
                <p class="card-text">
                    Please login to access protected resources.
                </p>
                <el-button type="primary" :loading="loading" size="large" class="login-button" @click="handleLogin">
                    Login with GSI Account
                </el-button>
                <el-alert v-if="errorMessage" title="" :description="errorMessage" type="error" show-icon
                    class="mt-3" />
            </div>
        </el-card>
    </div>
</template>

<script>
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import useAuth from '../composables/useAuth';
import { Loading } from '@element-plus/icons-vue';

export default {
    name: 'LoginView',
    components: {
        Loading
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
    min-height: calc(100vh - 120px);
    padding: 40px 0;
    display: flex;
    align-items: center;
    justify-content: center;
}

.login-card {
    width: 400px;
    max-width: 90%;
}

.card-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.text-center {
    text-align: center;
}

.loading-icon {
    font-size: 2rem;
    margin-bottom: 1rem;
    animation: rotating 2s linear infinite;
}

.loading-text {
    margin-top: 1rem;
}

.login-button {
    width: 100%;
    margin-top: 1rem;
}

.mt-3 {
    margin-top: 1rem;
}

@keyframes rotating {
    from {
        transform: rotate(0deg);
    }

    to {
        transform: rotate(360deg);
    }
}
</style>