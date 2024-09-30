import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'

const router = createRouter({
    history: createWebHistory(import.meta.env.BASE_URL),
    routes: [
        {
            path: '/',
            name: 'home',
            component: HomeView
        },
        {
            path: '/about',
            name: 'about',
            // route level code-splitting
            // this generates a separate chunk (About.[hash].js) for this route
            // which is lazy-loaded when the route is visited.
            component: () => import('../views/AboutView.vue')
        },
        {
            path: '/browse/:path*',
            name: 'browse',
            component: () => import('../views/BrowseXrdView.vue'),
            props: route => ({ path: Array.isArray(route.params.path) ? route.params.path.join('/') : route.params.path })
        },
        {
            path: '/documentation',
            name: 'documentation',
            component: () => import('../views/DocumentationView.vue')
        },
        {
            path: '/login',
            name: 'login',
            component: () => import('../views/LoginView.vue')
        },
        {
            // The Authentication process
            // - User Clicks Login Button:
            //   When the user clicks the login button on your SPA, redirect them to the third-party authentication service.
            // - Redirect to Authentication Service: 
            //   The user is redirected to the authentication service’s login page where they enter their credentials.
            // - Authentication and Token Generation: 
            //   After successful authentication, the service generates a token (usually a JWT).
            // - Redirect Back to Your App: 
            //   The authentication service redirects the user back to your app with the token included in the URL as a query parameter (e.g., https://yourapp.com/callback?token=xyz).
            // 
            // The Authentication Callback URL
            path: '/callback',
            component: () => import('../components/partials/CallbackComponent.vue')
        }
    ]
})

export default router
