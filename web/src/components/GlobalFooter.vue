<template>
    <footer class="global-footer">
        <span>&copy; 2026 GSI Helmholtzzentrum f&uuml;r Schwerionenforschung GmbH</span>
        <span class="separator">&middot;</span>
        <a href="https://www.gsi.de/impressum" target="_blank" rel="noopener noreferrer">Impressum</a>
        <span class="separator">&middot;</span>
        <a href="https://www.gsi.de/en/data-privacy-protection" target="_blank" rel="noopener noreferrer">Data privacy
            protection</a>
    </footer>
</template>

<style scoped>
.global-footer {
    padding: 1rem 1.5rem;
    text-align: center;
    font-size: 0.75rem;
    color: var(--el-text-color-placeholder);
    flex-shrink: 0;
}

.global-footer a {
    color: var(--el-text-color-placeholder);
    text-decoration: none;
}

.global-footer a:hover {
    color: var(--el-color-primary);
    text-decoration: underline;
}

.separator {
    margin: 0 0.4rem;
}
</style>
