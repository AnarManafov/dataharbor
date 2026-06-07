<template>
    <div class='home'>
        <section class='hero'>
            <div class='hero-body'>
                <div class='container'>
                    <h1 class='title'>
                        {{ isAuthenticated ? `Welcome back, ${greetingName}` : 'Welcome to DataHarbor' }}
                    </h1>
                    <h2 class='subtitle'>
                        {{ branding.heroSubtitle }}
                    </h2>
                    <div class='button-block'>
                        <router-link to="/browse" class='cta-button'>
                            <el-icon class="mr-2" aria-hidden="true">
                                <FolderOpened />
                            </el-icon>
                            {{ isAuthenticated ? 'Open File Browser' : 'Start Browsing Files' }}
                        </router-link>
                    </div>
                </div>
            </div>
        </section>

        <!-- Feature highlights -->
        <div class="features-section">
            <div class="container">
                <!-- Security spans the full width on top -->
                <div class="security-card">
                    <div class="security-header">
                        <div class="feature-icon">
                            <el-icon size="large" color="var(--el-color-info)" aria-hidden="true">
                                <Lock />
                            </el-icon>
                        </div>
                        <h3>Enterprise-Grade Security</h3>
                    </div>
                    <p class="security-intro">
                        Your identity and your data are protected at every step, so you can focus on your
                        research instead of security.
                    </p>
                    <div class="security-points">
                        <div class="security-point">
                            <el-icon aria-hidden="true">
                                <UserFilled />
                            </el-icon>
                            <div>
                                <strong>Sign in you can trust</strong>
                                <span>Login is handled by your own institution. DataHarbor never sees or stores your
                                    password.</span>
                            </div>
                        </div>
                        <div class="security-point">
                            <el-icon aria-hidden="true">
                                <Key />
                            </el-icon>
                            <div>
                                <strong>Your session stays private</strong>
                                <span>Your access keys are kept safely on the server, never in your browser, and your
                                    sign-in can't be read by other sites or scripts.</span>
                            </div>
                        </div>
                        <div class="security-point">
                            <el-icon aria-hidden="true">
                                <Connection />
                            </el-icon>
                            <div>
                                <strong>Encrypted in transit</strong>
                                <span>Every connection is protected with TLS encryption, from your browser to
                                    DataHarbor and on to the storage system.</span>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="features-grid">
                    <div class="feature-card">
                        <div class="feature-icon">
                            <el-icon size="large" color="var(--el-color-primary)" aria-hidden="true">
                                <FolderOpened />
                            </el-icon>
                        </div>
                        <h3>Browse</h3>
                        <p>Navigate directories, inspect metadata, and explore large-scale datasets with ease</p>
                    </div>

                    <div class="feature-card">
                        <div class="feature-icon">
                            <el-icon size="large" color="var(--el-color-success)" aria-hidden="true">
                                <Upload />
                            </el-icon>
                        </div>
                        <h3>Upload</h3>
                        <p>Drag and drop files into any directory. Chunked, SHA-256 verified, and resumable, so
                            interrupted transfers pick up where they left off</p>
                    </div>

                    <div class="feature-card">
                        <div class="feature-icon">
                            <el-icon size="large" color="var(--el-color-warning)" aria-hidden="true">
                                <Download />
                            </el-icon>
                        </div>
                        <h3>Download</h3>
                        <p>Stream multi-gigabyte files straight to your browser, or grab several at once as a single
                            .tar.gz archive</p>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { computed } from 'vue';
import { FolderOpened, Download, Upload, Lock, UserFilled, Key, Connection } from '@element-plus/icons-vue';
import { getConfig } from '@/config/config';
import useAuth from '@/composables/useAuth';

const branding = getConfig().branding || {};
const { isAuthenticated, user } = useAuth();

// Short, friendly first name for the personalized greeting.
// Mirrors the name-resolution order used in TopBar.vue.
const greetingName = computed(() => {
    const u = user.value;
    if (!u) return 'there';
    return u.given_name ||
        u.name?.split(' ')[0] ||
        u.preferred_username ||
        u.email?.split('@')[0] ||
        'there';
});
</script>


<style lang="scss" scoped>
.home {
    min-height: 100%;
    background: var(--el-bg-color-page);
}

.hero {
    text-align: center;
    background: var(--el-bg-color-page);
    min-height: 30vh;
    display: flex;
    align-items: center;

    .hero-body {
        padding: 3rem 1.5rem 1rem;
        width: 100%;

        .container {
            max-width: 800px;
            margin: 0 auto;
        }
    }
}

.title {
    color: var(--el-text-color-primary);
    font-size: 3rem;
    font-weight: 700;
    margin-bottom: 1rem;
    line-height: 1.2;
}

.subtitle {
    color: var(--el-text-color-regular);
    font-size: 1.3rem;
    font-weight: 400;
    margin-bottom: 2.5rem;
    line-height: 1.4;
}

.button-block {
    margin-top: 2.5rem;

    // Blue gradient drawn from the app's primary palette so it tracks the
    // active theme (light/dark) once a theme toggle is added.
    .cta-button {
        display: inline-flex;
        align-items: center;
        padding: 1rem 2.5rem;
        font-size: 1.1rem;
        font-weight: 600;
        border-radius: 50px;
        background: linear-gradient(135deg, var(--el-color-primary) 0%, var(--el-color-primary-dark-2) 100%);
        color: var(--el-color-white);
        border: none;
        transition: all 0.3s ease;
        text-decoration: none;
        box-shadow: var(--dh-shadow-md);

        &:hover {
            transform: translateY(-3px);
            box-shadow: var(--dh-shadow-lg);
        }
    }
}

.features-section {
    padding: 2rem 0 4rem;
    background: var(--el-bg-color-page);

    .container {
        max-width: 1100px;
        margin: 0 auto;
        padding: 0 2rem;
    }
}

/* ── Security card (full width, centered header) ───────────── */
.security-card {
    padding: 2rem;
    border-radius: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    box-shadow: var(--dh-shadow-sm);
    transition: all 0.3s ease;

    &:hover {
        box-shadow: var(--dh-shadow-lg);
        border-color: var(--el-color-primary-light-5);
    }
}

.security-header {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 0.75rem;
    margin-bottom: 0.75rem;

    .feature-icon {
        width: 48px;
        height: 48px;
        margin: 0;
    }

    h3 {
        font-size: 1.5rem;
        font-weight: 600;
        margin: 0;
        color: var(--el-text-color-primary);
    }
}

.security-intro {
    text-align: center;
    max-width: 640px;
    margin: 0 auto 2rem;
    color: var(--el-text-color-regular);
    line-height: 1.6;
}

.security-points {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1.5rem;
}

.security-point {
    display: flex;
    align-items: flex-start;
    gap: 0.75rem;

    .el-icon {
        color: var(--el-color-primary);
        font-size: 1.25rem;
        flex-shrink: 0;
        margin-top: 0.15rem;
    }

    strong {
        display: block;
        color: var(--el-text-color-primary);
        font-weight: 600;
        font-size: 0.975rem;
        margin-bottom: 0.25rem;
    }

    span {
        color: var(--el-text-color-regular);
        font-size: 0.9rem;
        line-height: 1.55;
    }
}

/* ── Action cards ──────────────────────────────────────────── */
.features-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1.75rem;
    margin-top: 1.75rem;
}

.feature-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    padding: 2rem 1.5rem;
    border-radius: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    box-shadow: var(--dh-shadow-sm);
    transition: all 0.3s ease;

    &:hover {
        transform: translateY(-5px);
        box-shadow: var(--dh-shadow-lg);
        border-color: var(--el-color-primary-light-5);
    }

    h3 {
        font-size: 1.5rem;
        font-weight: 600;
        margin-bottom: 0.75rem;
        color: var(--el-text-color-primary);
    }

    p {
        color: var(--el-text-color-regular);
        line-height: 1.6;
        margin: 0;
    }
}

.feature-icon {
    margin: 0 auto 1.5rem;
    display: flex;
    justify-content: center;
    align-items: center;
    width: 60px;
    height: 60px;
    border-radius: 50%;
    background: var(--el-color-primary-light-9);
    flex-shrink: 0;
}

.mr-2 {
    margin-right: 0.5rem;
}

@media (max-width: 900px) {
    .security-points {
        grid-template-columns: 1fr;
    }

    .features-grid {
        grid-template-columns: 1fr;
        gap: 1.5rem;
    }
}

@media (max-width: 768px) {
    .hero {
        min-height: 25vh;

        .hero-body {
            padding: 2rem 1rem 0.5rem;
        }
    }

    .title {
        font-size: 2.2rem;
    }

    .subtitle {
        font-size: 1.1rem;
        margin-bottom: 2rem;
    }

    .button-block .cta-button {
        padding: 0.8rem 2rem;
        font-size: 1rem;
    }

    .security-header {
        flex-direction: column;
        text-align: center;
    }

    .features-section {
        padding: 1rem 0 2rem;

        .container {
            padding: 0 1rem;
        }
    }
}
</style>
