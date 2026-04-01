<template>
    <div class="about">
        <div class="page-header">
            <div class="container">
                <h1 class="page-title">About DataHarbor</h1>
                <p class="page-subtitle">Secure, browser-based access to your data</p>
            </div>
        </div>

        <div class="content-section">
            <div class="container">
                <el-card class="mission-card" shadow="never">
                    <div class="mission-content">
                        <div class="mission-section">
                            <h2>Our Mission</h2>
                            <p>
                                DataHarbor bridges the gap between high-performance computing storage and
                                the web browser. By integrating directly with your storage via
                                the XRootD protocol, it gives researchers fast, secure access to their data
                                from anywhere — without specialized client software.
                            </p>
                        </div>
                        <div class="divider" />
                        <div class="features-section">
                            <h2>Key Features</h2>
                            <ul class="feature-list">
                                <li>
                                    <el-icon>
                                        <Check />
                                    </el-icon>
                                    <span><strong>OIDC / SSO login</strong> — authenticate with your institutional
                                        identity provider</span>
                                </li>
                                <li>
                                    <el-icon>
                                        <Check />
                                    </el-icon>
                                    <span><strong>XRootD file access</strong> — direct, high-performance access to
                                        cluster storage</span>
                                </li>
                                <li>
                                    <el-icon>
                                        <Check />
                                    </el-icon>
                                    <span><strong>Streaming downloads</strong> — transfer large files without browser
                                        memory limits</span>
                                </li>
                                <li>
                                    <el-icon>
                                        <Check />
                                    </el-icon>
                                    <span><strong>Paginated directory listing</strong> — navigate deep file hierarchies
                                        with ease</span>
                                </li>
                                <li>
                                    <el-icon>
                                        <Check />
                                    </el-icon>
                                    <span><strong>Zero client software</strong> — access your data from any modern web
                                        browser</span>
                                </li>
                                <li>
                                    <el-icon>
                                        <Check />
                                    </el-icon>
                                    <span><strong>Container-ready</strong> — production-grade Docker Compose deployment
                                        included</span>
                                </li>
                            </ul>
                        </div>
                    </div>
                </el-card>

                <el-card class="tech-card" shadow="never">
                    <div class="tech-card-header">
                        <h2>Technology Stack</h2>
                        <p>Powering DataHarbor from storage layer to browser</p>
                    </div>
                    <div class="tech-groups">
                        <div v-for="(items, layer) in techStack" :key="layer" class="tech-group">
                            <h3 class="group-label">{{ layerLabels[layer] }}</h3>
                            <div class="tech-col">
                                <div v-for="item in items" :key="item.name" class="tech-item"
                                    :style="{ '--accent-color': item.color }">
                                    <div class="tech-item-name">{{ item.name }}</div>
                                    <div class="tech-item-desc">{{ item.desc }}</div>
                                </div>
                            </div>
                        </div>
                    </div>
                </el-card>
            </div>
        </div>
    </div>
</template>

<script>
import { Check } from '@element-plus/icons-vue';

export default {
    name: 'AboutView',
    components: { Check },
    data() {
        return {
            layerLabels: {
                backend: 'Backend',
                frontend: 'Frontend',
                infrastructure: 'Infrastructure & Security',
            },
            techStack: {
                backend: [
                    { name: 'Go', desc: 'Language & runtime', color: '#00ADD8' },
                    { name: 'Gin', desc: 'HTTP web framework', color: '#4DB7E5' },
                    { name: 'XRootD', desc: 'High-performance data access', color: '#E85D04' },
                ],
                frontend: [
                    { name: 'Vue.js', desc: 'Reactive UI framework', color: '#4FC08D' },
                    { name: 'Vite', desc: 'Build tool & dev server', color: '#646CFF' },
                    { name: 'Element Plus', desc: 'UI component library', color: '#409EFF' },
                ],
                infrastructure: [
                    { name: 'Docker', desc: 'Containerization & orchestration', color: '#2496ED' },
                    { name: 'Nginx', desc: 'Reverse proxy & TLS termination', color: '#009900' },
                    { name: 'OIDC', desc: 'Federated identity & SSO', color: '#EB5424' },
                ],
            },
        };
    },
};
</script>
<style lang="scss" scoped>
.about {
    min-height: 100%;
    background: #f8f9fa;
}

.page-header {
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    color: white;
    padding: 3rem 0;
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
    margin-bottom: 0.75rem;
    text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.3);
}

.page-subtitle {
    font-size: 1.1rem;
    font-weight: 300;
    opacity: 0.85;
}

.content-section {
    padding: 2.5rem 0 3rem;

    .container {
        max-width: 1200px;
        margin: 0 auto;
        padding: 0 2rem;
        display: flex;
        flex-direction: column;
        gap: 1.5rem;
    }
}

/* ── Mission + Features combined card ─────────────────────── */
.mission-card {
    border-radius: 12px;
    border: none;
}

.mission-content {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    gap: 0 2.5rem;
    align-items: start;
}

.mission-section,
.features-section {
    h2 {
        color: var(--el-text-color-primary);
        font-size: 1.35rem;
        font-weight: 600;
        margin: 0 0 1rem 0;
    }

    p {
        color: var(--el-text-color-regular);
        line-height: 1.7;
        font-size: 0.975rem;
        margin: 0;
    }
}

.divider {
    width: 1px;
    align-self: stretch;
    background: var(--el-border-color-lighter);
}

.feature-list {
    list-style: none;
    padding: 0;
    margin: 0;

    li {
        display: flex;
        align-items: baseline;
        gap: 0.6rem;
        padding: 0.35rem 0;
        color: var(--el-text-color-regular);
        font-size: 0.925rem;
        line-height: 1.5;

        .el-icon {
            flex-shrink: 0;
            color: var(--el-color-success);
            position: relative;
            top: 1px;
        }

        strong {
            color: var(--el-text-color-primary);
        }
    }
}

/* ── Technology Stack card ─────────────────────────────────── */
.tech-card {
    border-radius: 12px;
    border: none;
}

.tech-card-header {
    text-align: center;
    margin-bottom: 2rem;

    h2 {
        color: var(--el-text-color-primary);
        font-size: 1.5rem;
        font-weight: 600;
        margin-bottom: 0.35rem;
    }

    p {
        color: var(--el-text-color-secondary);
        font-size: 0.95rem;
        margin: 0;
    }
}

.tech-groups {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 0 2rem;
}

.group-label {
    font-size: 0.7rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.1em;
    color: var(--el-text-color-secondary);
    margin: 0 0 0.875rem 0;
    padding-bottom: 0.5rem;
    border-bottom: 1px solid var(--el-border-color-lighter);
}

.tech-col {
    display: flex;
    flex-direction: column;
    gap: 0.625rem;
}

.tech-item {
    padding: 0.75rem 1rem;
    border-radius: 8px;
    border: 1px solid var(--el-border-color-light);
    border-left: 3px solid var(--accent-color, var(--el-color-primary));
    background: var(--el-bg-color);
    transition: box-shadow 0.2s ease, transform 0.2s ease;

    &:hover {
        box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
        transform: translateX(2px);
    }
}

.tech-item-name {
    font-size: 0.9rem;
    font-weight: 600;
    color: var(--el-text-color-primary);
    margin-bottom: 0.2rem;
}

.tech-item-desc {
    font-size: 0.775rem;
    color: var(--el-text-color-secondary);
    line-height: 1.4;
}

@media (max-width: 900px) {
    .mission-content {
        grid-template-columns: 1fr;
        gap: 1.5rem 0;
    }

    .divider {
        width: 100%;
        height: 1px;
        align-self: auto;
    }

    .tech-groups {
        grid-template-columns: 1fr;
        gap: 1.5rem 0;
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
}
</style>
