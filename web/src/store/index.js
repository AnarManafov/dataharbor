// Updated Vuex store implementation for BFF pattern
import * as Vuex from 'vuex';

// Simple store without OIDC module
const store = Vuex.createStore({
    state: {
        // Global app state
        isAuthenticated: false,
        user: null
    },
    getters: {
        isAuthenticated: state => state.isAuthenticated,
        user: state => state.user
    },
    mutations: {
        setAuthenticated(state, isAuthenticated) {
            state.isAuthenticated = isAuthenticated;
        },
        setUser(state, user) {
            state.user = user;
        }
    },
    actions: {
        // Actions for global state management
        updateAuthState({ commit }, { isAuthenticated, user }) {
            commit('setAuthenticated', isAuthenticated);
            commit('setUser', user);
        }
    },
    modules: {
        // Empty oidcStore module to prevent errors in components that 
        // might still reference it during the transition to BFF pattern
        oidcStore: {
            namespaced: true,
            state: {
                user: null,
                is_authenticated: false
            },
            getters: {
                oidcUser: () => null,
                isAuthenticated: () => false
            },
            actions: {
                signInRedirect: () => Promise.resolve(),
                signInOidc: () => Promise.resolve(),
                signInRedirectOidc: () => Promise.resolve(),
                signInSilent: () => Promise.resolve(),
                signInCallback: () => Promise.resolve(),
                signOut: () => Promise.resolve(),
                authenticateOidcSilent: () => Promise.resolve()
            }
        }
    }
});

export default store;
