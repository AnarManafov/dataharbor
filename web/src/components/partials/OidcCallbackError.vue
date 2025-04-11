<template>
    <div class="error-container">
        <div class="error-content">
            <h2 class="text-danger">Authentication Error</h2>
            <p>There was a problem processing your login.</p>
            <p class="error-message">The authentication server encountered an error or rejected the request.</p>
            <button class="btn btn-primary mt-3" @click="goToLogin">Try Again</button>
        </div>
    </div>
</template>

<script setup>
import { useRouter } from 'vue-router';

const router = useRouter();

const goToLogin = () => {
    router.push('/login');
};
</script>

<style scoped>
.error-container {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 70vh;
}

.error-content {
    text-align: center;
    max-width: 400px;
    padding: 2rem;
    border-radius: 8px;
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
}

.error-message {
    padding: 1rem;
    background-color: #f8d7da;
    border-radius: 4px;
    margin-top: 1rem;
}
</style>
