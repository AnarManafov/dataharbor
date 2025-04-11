<template>
    <div class="callback-container">
        <div class="spinner-container">
            <div class="spinner-border" role="status">
                <span class="visually-hidden">Loading...</span>
            </div>
            <p class="mt-3">Processing authentication callback...</p>
            <p v-if="error" class="text-danger mt-2">{{ error }}</p>
        </div>
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuth } from '../../composables/useAuth';

const router = useRouter();
const route = useRoute();
const { handleCallback } = useAuth();
const error = ref('');

onMounted(async () => {
    try {
        console.log("Processing callback with params:", route.query);

        // Extract auth code and state parameters from URL to complete the OIDC flow
        const code = route.query.code;
        const state = route.query.state;

        if (!code) {
            console.error("No authorization code in callback");
            error.value = "Missing authorization code";
            return;
        }

        // Pass the auth parameters to the backend to complete the token exchange
        // This must happen server-side to keep client_secret secure
        await handleCallback(code, state, router);

        console.log("Callback being processed by backend");
    } catch (err) {
        console.error('Callback processing error:', err);
        error.value = err.message || 'Authentication failed';

        // Redirect to login after a brief delay to allow error message to be seen
        setTimeout(() => router.push('/login'), 3000);
    }
});
</script>

<style scoped>
.callback-container {
    display: flex;
    justify-content: center;
    align-items: center;
    height: 70vh;
    /* Use most of viewport height for visual balance */
}

.spinner-container {
    text-align: center;
}
</style>
