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
import { useRouter } from 'vue-router';
import useAuth from '../../composables/useAuth';

const router = useRouter();
const { handleCallback, error: authError } = useAuth();
const error = ref('');

onMounted(async () => {
    try {
        // Process the auth callback using vue-oidc-client
        await handleCallback();

        // The handleCallback function handles the redirect internally
    } catch (err) {
        console.error('Callback processing error:', err);
        error.value = authError.value || 'Authentication failed';

        // Redirect to login after a delay
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
}

.spinner-container {
    text-align: center;
}
</style>
