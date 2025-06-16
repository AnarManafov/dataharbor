import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import BrowseXrdView from '../views/BrowseXrdView.vue'
import AboutView from '../views/AboutView.vue'
import DocumentationView from '../views/DocumentationView.vue'
import LoginView from '../views/LoginView.vue'
import OidcCallbackComponent from '../components/partials/OidcCallbackComponent.vue'
import OidcCallbackError from '../components/partials/OidcCallbackError.vue'
import { useAuth } from '../composables/useAuth'

// Routes configuration with metadata for access control
const routes = [
    {
        path: '/',
        name: 'home',
        component: HomeView,
        // Public routes allow access even when not authenticated
        meta: { isPublic: true }
    },
    {
        path: '/browse/:path(.*)*',
        name: 'browse',
        component: BrowseXrdView,
        // Protected routes will redirect to login when user isn't authenticated
        meta: { requiresAuth: true },
        props: true
    },
    {
        path: '/about',
        name: 'about',
        component: AboutView,
        meta: { isPublic: true }
    },
    {
        path: '/docs',
        name: 'docs',
        component: DocumentationView,
        meta: { isPublic: true }
    },
    // Secondary path for documentation to support existing bookmarks and links
    {
        path: '/documentation',
        redirect: '/docs',
        meta: { isPublic: true }
    },
    {
        path: '/login',
        name: 'login',
        component: LoginView,
        meta: { isPublic: true }
    },
    // Authentication callback routes for OIDC flow
    {
        path: '/oidc-callback',
        name: 'oidcCallback',
        component: OidcCallbackComponent,
        meta: { isPublic: true }
    },
    {
        path: '/oidc-callback-error',
        name: 'oidcCallbackError',
        component: OidcCallbackError,
        meta: { isPublic: true }
    }
]

const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes
})

// Create auth instance before router guards run to ensure authentication state is available
const { checkAuth } = useAuth();

// Global navigation guard to enforce authentication requirements
router.beforeEach(async (to, from, next) => {
    const isPublic = to.matched.some(record => record.meta.isPublic);
    const requiresAuth = to.matched.some(record => record.meta.requiresAuth);
    // Skip auth check for OAuth callback routes to prevent authentication loops
    const isOidcCallback = to.path.includes('/oidc-callback');

    // OIDC callbacks must proceed without auth checks
    if (isOidcCallback) {
        return next();
    }

    // Public routes don't need authentication verification
    if (isPublic && !requiresAuth) {
        return next();
    }

    try {
        // For protected routes, verify user is authenticated
        if (requiresAuth) {
            const isAuthenticated = await checkAuth();

            if (!isAuthenticated) {
                // Redirect to login with return URL to bring user back after authentication
                return next({
                    path: '/login',
                    query: { redirect: to.fullPath }
                });
            }
        }

        // Continue to requested route
        next();
    } catch (error) {
        console.error('Navigation guard error:', error);

        // Safely handle auth check failures
        if (requiresAuth) {
            next({
                path: '/login',
                query: { redirect: to.fullPath }
            });
        } else {
            next();
        }
    }
});

export default router
