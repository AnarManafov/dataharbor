// src/composables/useAuth.js
import { ref, provide, inject } from 'vue';

const authSymbol = Symbol();

export function provideAuth() {
    const isLoggedIn = ref(false);
    const userName = ref('');

    const login = () => {
        isLoggedIn.value = true;
    };

    const logout = () => {
        isLoggedIn.value = false;
        userName.value = '';
    };

    const setUserName = (name) => {
        userName.value = name;
    };

    provide(authSymbol, {
        isLoggedIn,
        userName,
        login,
        logout,
        setUserName
    });
}

export function useAuth() {
    const auth = inject(authSymbol);
    if (!auth) {
        throw new Error('useAuth must be used within a provideAuth');
    }
    return auth;
}
