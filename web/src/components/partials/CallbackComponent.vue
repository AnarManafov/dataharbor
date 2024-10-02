<script>
import { jwtDecode } from 'jwt-decode';
import { useAuth } from '../../composables/useAuth';

//
// A Login Process
//
// A login callback component that extracts the token from the URL and stores it securely. 
export default {
    created() {
        const { login, setUserName } = useAuth();
        const token = this.$route.query.token;
        if (token) {
            localStorage.setItem('authToken', token);
            console.log('Token: ', token);

            const decodedToken = jwtDecode(token);
            const firstName = decodedToken.FirstName;
            const lastName = decodedToken.LastName;
            const expirationDate = new Date(decodedToken.exp * 1000);

            console.log('Name: %s %s', firstName, lastName);
            console.log('Expiration Date: ', expirationDate);

            // Update the user name
            setUserName(`${firstName} ${lastName}`);
            // Update the login state
            login();

            this.$router.push('/browse');
        } else {
            console.error('No token found in URL');
        }
    }
}
</script>
