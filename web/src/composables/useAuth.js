// src/composables/useAuth.js
import { ref, provide, inject } from 'vue';

const authSymbol = Symbol();

export function provideAuth() {
    const isLoggedIn = ref(false);

    const login = () => {
        isLoggedIn.value = true;
    };

    const logout = () => {
        isLoggedIn.value = false;
    };

    provide(authSymbol, {
        isLoggedIn,
        login,
        logout
    });
}

export function useAuth() {
    const auth = inject(authSymbol);
    if (!auth) {
        throw new Error('useAuth must be used within a provideAuth');
    }
    return auth;
}
