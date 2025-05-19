import { ref, computed } from 'vue';
import { useRouter } from 'vue-router';
import axios from 'axios';
import { getConfig } from '../config/config';
import { getUserInfo, login as apiLogin, logout as apiLogout } from '../api/api';

// Create singleton state to maintain auth state across components
// This prevents auth status from resetting when components remount
const isAuthenticated = ref(false);
const user = ref(null);
const isLoading = ref(false);
const error = ref(null);

// Enable cookie-based auth across domains
axios.defaults.withCredentials = true;

// Main composable function for authentication
export default function useAuth() {
    const router = useRouter();
    const config = getConfig();

    // Determine if user has required permissions for specific features
    const hasRole = (role) => {
        if (!user.value || !user.value.roles) return false;
        return user.value.roles.includes(role);
    };

    // Validate current session and refresh user data
    const checkAuth = async () => {
        isLoading.value = true;
        error.value = null;

        try {
            const response = await getUserInfo();

            if (response && response.data) {
                isAuthenticated.value = true;
                user.value = response.data;

                // Debug user data for troubleshooting permissions issues
                console.log('User data from auth response:', response.data);
            } else {
                isAuthenticated.value = false;
                user.value = null;
            }
        } catch (err) {
            isAuthenticated.value = false;
            user.value = null;
            error.value = err.message || 'Failed to check authentication status';
            console.error('Auth check error:', err);
        } finally {
            isLoading.value = false;
        }

        return isAuthenticated.value;
    };

    // Initiate OIDC authentication flow while preserving intended destination
    const login = async () => {
        isLoading.value = true;
        error.value = null;

        try {
            // Preserve navigation context for post-login redirection
            const currentPath = router?.currentRoute?.value?.path || '/';
            const redirectPath = currentPath !== '/login' ? currentPath : '/';

            // Backend generates the proper auth URL with correct parameters
            const response = await apiLogin(redirectPath);

            if (response && response.data && response.data.auth_url) {
                console.log('Redirecting to auth URL:', response.data.auth_url);
                window.location.href = response.data.auth_url;
            } else {
                console.error('Invalid login response', response);
                error.value = 'Failed to initialize login flow';
                isLoading.value = false;
            }
        } catch (err) {
            console.error('Login error:', err);
            error.value = err.message || 'Failed to start authentication';
            isLoading.value = false;
            throw err; // Re-throw to allow proper error handling in the calling component
        }
    };

    // Terminate user session and clear application state
    const logout = async () => {
        isLoading.value = true;

        try {
            await apiLogout();
            isAuthenticated.value = false;
            user.value = null;
            router.push('/login');
        } catch (err) {
            error.value = 'Logout failed';
            console.error('Logout error:', err);
        } finally {
            isLoading.value = false;
        }
    };

    return {
        isAuthenticated: computed(() => isAuthenticated.value),
        user: computed(() => user.value),
        isLoading: computed(() => isLoading.value),
        error: computed(() => error.value),
        login,
        logout,
        checkAuth,
        hasRole
    };
}

// Also export as a named export for components that prefer this syntax
export { useAuth };
