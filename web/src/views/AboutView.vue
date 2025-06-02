<template>
    <div class="about">
        <div class="page-header">
            <div class="container">
                <h1 class="page-title">About DataHarbor</h1>
                <p class="page-subtitle">Learn more about our mission and capabilities</p>
            </div>
        </div>

        <div class="content-section">
            <div class="container">
                <el-row :gutter="32">
                    <el-col :span="24" :md="12">
                        <el-card class="info-card" shadow="never">
                            <h2>Our Mission</h2>
                            <p>
                                DataHarbor provides a seamless interface for browsing and managing GSI Lustre file
                                systems.
                                We aim to make file system navigation intuitive and efficient for researchers and
                                scientists.
                            </p>
                        </el-card>
                    </el-col>

                    <el-col :span="24" :md="12">
                        <el-card class="info-card" shadow="never">
                            <h2>Key Features</h2>
                            <ul class="feature-list">
                                <li><el-icon>
                                        <Check />
                                    </el-icon> Secure file browsing</li>
                                <li><el-icon>
                                        <Check />
                                    </el-icon> User authentication</li>
                                <li><el-icon>
                                        <Check />
                                    </el-icon> Real-time monitoring</li>
                                <li><el-icon>
                                        <Check />
                                    </el-icon> Efficient downloads</li>
                            </ul>
                        </el-card>
                    </el-col>
                </el-row>

                <el-row :gutter="32" class="tech-section">
                    <el-col :span="24">
                        <el-card class="tech-card" shadow="hover">
                            <h2>Technology Stack</h2>
                            <div class="tech-grid">
                                <div class="tech-item">
                                    <div class="tech-icon">
                                        <el-icon size="24">
                                            <Platform />
                                        </el-icon>
                                    </div>
                                    <h3>Vue.js 3</h3>
                                    <p>Modern reactive frontend framework</p>
                                </div>
                                <div class="tech-item">
                                    <div class="tech-icon">
                                        <el-icon size="24">
                                            <Monitor />
                                        </el-icon>
                                    </div>
                                    <h3>Element Plus</h3>
                                    <p>Enterprise-class UI components</p>
                                </div>
                                <div class="tech-item">
                                    <div class="tech-icon">
                                        <el-icon size="24">
                                            <Connection />
                                        </el-icon>
                                    </div>
                                    <h3>RESTful API</h3>
                                    <p>Secure backend integration</p>
                                </div>
                            </div>
                        </el-card>
                    </el-col>
                </el-row>
            </div>
        </div>
    </div>
</template>

<script>
import { Check, Platform, Monitor, Connection } from '@element-plus/icons-vue';

export default {
    name: 'AboutView',
    components: {
        Check,
        Platform,
        Monitor,
        Connection
    }
};
</script>
<style lang="scss" scoped>
.about {
    min-height: 100vh;
    background: #f8f9fa;
}

.page-header {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 4rem 0;
    text-align: center;

    .container {
        max-width: 1200px;
        margin: 0 auto;
        padding: 0 2rem;
    }
}

.page-title {
    font-size: 3rem;
    font-weight: 700;
    margin-bottom: 1rem;
    text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.3);
}

.page-subtitle {
    font-size: 1.25rem;
    font-weight: 300;
    opacity: 0.9;
}

.content-section {
    padding: 4rem 0;

    .container {
        max-width: 1200px;
        margin: 0 auto;
        padding: 0 2rem;
    }
}

.info-card {
    height: 100%;
    border-radius: 12px;
    border: none;

    h2 {
        color: var(--el-text-color-primary);
        font-size: 1.5rem;
        font-weight: 600;
        margin-bottom: 1.5rem;
    }

    p {
        color: var(--el-text-color-regular);
        line-height: 1.6;
        font-size: 1rem;
    }
}

.feature-list {
    list-style: none;
    padding: 0;

    li {
        display: flex;
        align-items: center;
        margin-bottom: 0.75rem;
        color: var(--el-text-color-regular);

        .el-icon {
            color: var(--el-color-success);
            margin-right: 0.75rem;
        }
    }
}

.tech-section {
    margin-top: 3rem;
}

.tech-card {
    border-radius: 12px;
    border: none;
    transition: all 0.3s ease;

    h2 {
        color: var(--el-text-color-primary);
        font-size: 1.75rem;
        font-weight: 600;
        margin-bottom: 2rem;
        text-align: center;
    }
}

.tech-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
    gap: 2rem;
}

.tech-item {
    text-align: center;
    padding: 1.5rem;

    .tech-icon {
        display: flex;
        justify-content: center;
        align-items: center;
        width: 60px;
        height: 60px;
        border-radius: 50%;
        background: var(--el-color-primary-light-9);
        margin: 0 auto 1rem;
        color: var(--el-color-primary);
    }

    h3 {
        font-size: 1.25rem;
        font-weight: 600;
        margin-bottom: 0.5rem;
        color: var(--el-text-color-primary);
    }

    p {
        color: var(--el-text-color-regular);
        font-size: 0.9rem;
    }
}

@media (max-width: 768px) {
    .page-title {
        font-size: 2rem;
    }

    .page-subtitle {
        font-size: 1rem;
    }

    .content-section {
        padding: 2rem 0;
    }

    .tech-grid {
        grid-template-columns: 1fr;
        gap: 1.5rem;
    }
}
</style>