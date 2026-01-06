<template>
    <div class='home'>
        <section class='hero is-dark'>
            <div class='hero-body'>
                <div class='container'>
                    <h1 class='title'>Welcome to DataHarbor</h1>
                    <h2 class='subtitle'>
                        Browsing GSI Lustre made easy
                    </h2>
                    <div class='button-block'>
                        <router-link to="/browse" class='button is-primary is-rounded'>
                            <el-icon class="mr-2" size="large">
                                <FolderOpened />
                            </el-icon>
                            Start Browsing Files
                        </router-link>
                    </div>
                </div>
            </div>
        </section>

        <!-- Feature Cards Section -->
        <div class="features-section">
            <div class="container">
                <div class="features-grid">
                    <el-card class="feature-card" shadow="hover">
                        <div class="feature-icon">
                            <el-icon size="large" color="var(--el-color-primary)">
                                <FolderOpened />
                            </el-icon>
                        </div>
                        <h3>Intuitive File Browser</h3>
                        <p>Navigate directories, inspect metadata, and explore large-scale datasets with ease</p>
                    </el-card>

                    <el-card class="feature-card" shadow="hover">
                        <div class="feature-icon">
                            <el-icon size="large" color="var(--el-color-success)">
                                <Download />
                            </el-icon>
                        </div>
                        <h3>Efficient Large File Downloads</h3>
                        <p>Stream multi-gigabyte files directly to your browser with chunked transfer and adaptive
                            buffering</p>
                    </el-card>

                    <el-card class="feature-card" shadow="hover">
                        <div class="feature-icon">
                            <el-icon size="large" color="var(--el-color-warning)">
                                <Lock />
                            </el-icon>
                        </div>
                        <h3>Enterprise-Grade Security</h3>
                        <p>OpenID Connect authentication with server-side token management and TLS-encrypted XRootD
                            connections</p>
                    </el-card>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import { FolderOpened, Download, Lock } from '@element-plus/icons-vue';

export default {
    name: 'HomeView',
    components: {
        FolderOpened,
        Download,
        Lock
    },
};
</script>


<style lang="scss" scoped>
.home {
    min-height: 100vh;
}

.hero {
    text-align: center;
    background: linear-gradient(135deg, var(--el-color-primary) 0%, var(--el-color-primary-dark-2) 100%);
    min-height: 35vh;
    display: flex;
    align-items: center;
    position: relative;
    overflow: hidden;

    &::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: url("data:image/svg+xml,%3Csvg width='60' height='60' viewBox='0 0 60 60' xmlns='http://www.w3.org/2000/svg'%3E%3Cg fill='none' fill-rule='evenodd'%3E%3Cg fill='%23ffffff' fill-opacity='0.1'%3E%3Ccircle cx='30' cy='30' r='2'/%3E%3C/g%3E%3C/g%3E%3C/svg%3E") repeat;
        opacity: 0.3;
    }

    &::after {
        content: '';
        position: absolute;
        bottom: 0;
        left: 0;
        right: 0;
        height: 100px;
        background: linear-gradient(to bottom, transparent, rgba(248, 249, 250, 0.1));
    }

    .hero-body {
        position: relative;
        z-index: 1;
        padding: 3rem 1.5rem;
        width: 100%;

        .container {
            max-width: 800px;
            margin: 0 auto;
        }
    }
}

.title {
    color: white;
    font-size: 3rem;
    font-weight: 700;
    margin-bottom: 1rem;
    text-shadow: 0 2px 10px rgba(0, 0, 0, 0.3);
    line-height: 1.2;
}

.subtitle {
    color: rgba(255, 255, 255, 0.9);
    font-size: 1.3rem;
    font-weight: 400;
    margin-bottom: 2.5rem;
    text-shadow: 0 1px 5px rgba(0, 0, 0, 0.2);
    line-height: 1.4;
}

.button-block {
    margin-top: 2.5rem;

    .button {
        display: inline-flex;
        align-items: center;
        padding: 1rem 2.5rem;
        font-size: 1.1rem;
        font-weight: 600;
        border-radius: 50px;
        background: rgba(255, 255, 255, 0.95);
        color: var(--el-color-primary);
        border: 2px solid rgba(255, 255, 255, 0.2);
        transition: all 0.3s ease;
        text-decoration: none;
        backdrop-filter: blur(10px);
        box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);

        &:hover {
            transform: translateY(-3px);
            box-shadow: 0 8px 25px rgba(0, 0, 0, 0.2);
            background: white;
            color: var(--el-color-primary-dark-2);
        }
    }
}

.features-section {
    padding: 4rem 0;
    background: var(--el-bg-color-page);

    .container {
        max-width: 1200px;
        margin: 0 auto;
        padding: 0 2rem;
    }
}

.features-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 2rem;
    margin-top: 2rem;
}

.feature-card {
    text-align: center;
    padding: 2rem 1.5rem;
    border-radius: 12px;
    transition: all 0.3s ease;
    border: none;

    &:hover {
        transform: translateY(-5px);
        box-shadow: 0 10px 30px rgba(0, 0, 0, 0.1);
    }

    .feature-icon {
        margin-bottom: 1.5rem;
        display: flex;
        justify-content: center;
        align-items: center;
        width: 60px;
        height: 60px;
        border-radius: 50%;
        background: var(--el-color-primary-light-9);
        margin: 0 auto 1.5rem;
    }

    h3 {
        font-size: 1.5rem;
        font-weight: 600;
        margin-bottom: 1rem;
        color: var(--el-text-color-primary);
    }

    p {
        color: var(--el-text-color-regular);
        line-height: 1.6;
    }
}

.mr-2 {
    margin-right: 0.5rem;
}

@media (max-width: 768px) {
    .hero {
        min-height: 30vh;

        .hero-body {
            padding: 2rem 1rem;
        }
    }

    .title {
        font-size: 2.2rem;
    }

    .subtitle {
        font-size: 1.1rem;
        margin-bottom: 2rem;
    }

    .button-block .button {
        padding: 0.8rem 2rem;
        font-size: 1rem;
    }

    .features-grid {
        grid-template-columns: 1fr;
        gap: 1.5rem;
    }

    .features-section {
        padding: 2rem 0;

        .container {
            padding: 0 1rem;
        }
    }
}
</style>