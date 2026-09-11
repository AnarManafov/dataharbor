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

// Loop guard for automatic sign-in. If the IdP sends us back and the session
// still doesn't validate (cookie blocked, clock skew, backend restart), a page
// that auto-starts login would bounce forever. Remember when we last started an
// automatic attempt so the login page can fall back to a manual button.
const AUTO_LOGIN_KEY = 'dh_auto_login_started_at';
const AUTO_LOGIN_COOLDOWN_MS = 60 * 1000;

const markAutoLoginStarted = () => {
    try { sessionStorage.setItem(AUTO_LOGIN_KEY, String(Date.now())); } catch { /* storage unavailable */ }
};
const clearAutoLoginMark = () => {
    try { sessionStorage.removeItem(AUTO_LOGIN_KEY); } catch { /* storage unavailable */ }
};
// True when an automatic attempt was started recently and never completed.
export const autoLoginRecentlyFailed = () => {
    try {
        const at = Number(sessionStorage.getItem(AUTO_LOGIN_KEY) || 0);
        return at > 0 && Date.now() - at < AUTO_LOGIN_COOLDOWN_MS;
    } catch {
        return false;
    }
};

// Only same-origin paths may be used as a post-login destination. The backend
// enforces the same rule; this keeps the UI from ever asking for anything else.
export const sanitizeReturnPath = (path) => {
    if (typeof path !== 'string' || !path.startsWith('/') || path.startsWith('//')) return '/';
    return path;
};

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
                clearAutoLoginMark();

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

    // Where to land after login when the caller didn't say: the current page,
    // or — on the login page itself — whatever it was asked to return to.
    const defaultReturnPath = () => {
        const current = router?.currentRoute?.value;
        if (!current) return '/';
        if (current.path === '/login') return current.query?.redirect || '/';
        return current.fullPath;
    };

    // Initiate the OIDC flow. This is the one and only "sign in" action: it
    // sends the browser straight to the identity provider, which owns the
    // credentials UI. There is deliberately no in-app login form in between.
    // @param {string} [returnPath] in-app path to come back to after login
    // @param {object} [opts] { automatic: true } when started without a click
    const login = async (returnPath, opts = {}) => {
        isLoading.value = true;
        error.value = null;

        try {
            const redirectPath = sanitizeReturnPath(returnPath ?? defaultReturnPath());

            // Backend generates the proper auth URL with correct parameters
            const response = await apiLogin(redirectPath);

            if (response && response.data && response.data.auth_url) {
                console.log('Redirecting to auth URL:', response.data.auth_url);
                if (opts.automatic) markAutoLoginStarted();
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
            // Land on the public home page. Sending a user who just signed out
            // to a "sign in" page reads as if the sign-out didn't take.
            router.push('/');
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
