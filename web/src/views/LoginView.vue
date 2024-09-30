<!-- src/views/LoginView.vue -->
<template>
    <div>
        <h1>Login</h1>
        <!-- Add your login form here -->
    </div>
</template>

<script>
export default {
    name: 'LoginView',
    mounted() {
        window.location.href = 'callback?token=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJPbmxpbmUgSldUIEJ1aWxkZXIiLCJpYXQiOjE3Mjc2OTY4NTAsImV4cCI6MTc1OTIzMjg1MCwiYXVkIjoid3d3LmV4YW1wbGUuY29tIiwic3ViIjoianJvY2tldEBleGFtcGxlLmNvbSIsIkdpdmVuTmFtZSI6IkpvaG5ueSIsIlN1cm5hbWUiOiJSb2NrZXQiLCJFbWFpbCI6Impyb2NrZXRAZXhhbXBsZS5jb20iLCJSb2xlIjpbIk1hbmFnZXIiLCJQcm9qZWN0IEFkbWluaXN0cmF0b3IiXX0.tQEp51BqOHK6gBfrMtygdLNn-8Q99is8xaWffnYd3A4';
    }
};
</script>