<script>
export default {
    created() {
        const token = this.$route.query.token;
        if (token) {
            // Store the token securely
            localStorage.setItem('authToken', token);
            console.log('Token: ', token);
            
            // Redirect to the main app
            this.$router.push('/browse');
        } else {
            // Handle error
            console.error('No token found in URL');
        }
    }
}
</script>