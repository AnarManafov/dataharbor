<template>
    <div class="callback-container">
        <p v-if="loading">Processing authentication...</p>
    </div>
</template>

<script>
import { jwtDecode } from 'jwt-decode';
import { useAuth } from '../../composables/useAuth';

export default {
    name: 'CallbackComponent',
    props: {
        token: {
            type: String,
            default: null
        }
    },
    data() {
        return {
            loading: true
        }
    },
    mounted() {
        const { login, setUserName } = useAuth();
        // Handle the token from the query parameter
        if (this.token) {
            // Store the token in localStorage
            localStorage.setItem('authToken', this.token);
            console.log('Authentication token stored');

            const decodedToken = jwtDecode(this.token);
            const firstName = decodedToken.FirstName;
            const lastName = decodedToken.LastName;
            const expirationDate = new Date(decodedToken.exp * 1000);

            console.log('Name: %s %s', firstName, lastName);
            console.log('Expiration Date: ', expirationDate);

            // Update the user name
            setUserName(`${firstName} ${lastName}`);
            // Update the login state
            login();

            // Redirect to the protected route or home
            this.$router.push({ name: 'browse', params: { path: '' } });
        } else {
            // No token found, redirect to login
            console.error('No authentication token received');
            this.$router.push({ name: 'login' });
        }
    }
}
</script>
