<!-- src/views/LoginView.vue -->
<template>
    <div class="login-view">
        <div class="container py-5">
            <div class="row justify-content-center">
                <div class="col-md-6">
                    <div class="card shadow">
                        <div class="card-header bg-primary text-white">
                            <h3 class="mb-0">Login</h3>
                        </div>
                        <div class="card-body">
                            <div v-if="loading" class="text-center py-3">
                                <div class="spinner-border" role="status">
                                    <span class="visually-hidden">Loading...</span>
                                </div>
                                <p class="mt-2">Authenticating...</p>
                            </div>
                            <div v-else>
                                <p class="card-text">
                                    Please login to access protected resources.
                                </p>
                                <button class="btn btn-primary w-100" @click="handleLogin" :disabled="loading">
                                    Login with GSI Account
                                </button>
                                <div v-if="errorMessage" class="alert alert-danger mt-3">
                                    {{ errorMessage }}
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import useAuth from '../composables/useAuth';

export default {
    name: 'LoginView',
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
    /* Account for header/footer space */
    padding: 40px 0;
    display: flex;
    align-items: center;
}
</style>