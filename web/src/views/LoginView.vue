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
        // a fake token for demonstration purposes. Generated using http://jwtbuilder.jamiekurtz.com
        window.location.href = 'callback?token=eyJ0eXAiOiJKV1QiLCJhbGciOiJIUzI1NiJ9.eyJpc3MiOiJPbmxpbmUgSldUIEJ1aWxkZXIiLCJpYXQiOjE3Mjc2OTc5MzgsImV4cCI6MTc1OTIzMzkzOCwiYXVkIjoid3d3LmV4YW1wbGUuY29tIiwic3ViIjoianJvY2tldEBleGFtcGxlLmNvbSIsIkZpcnN0TmFtZSI6IkpvaG5ueSIsIkxhc3ROYW1lIjoiUm9ja2V0IiwiRW1haWwiOiJqcm9ja2V0QGV4YW1wbGUuY29tIiwiUm9sZSI6WyJNYW5hZ2VyIiwiUHJvamVjdCBBZG1pbmlzdHJhdG9yIl19._CxtUaVCN7zycnfXEzQBQxgsrwj7LZPA5nPl8IA338o';
    }
};
</script>