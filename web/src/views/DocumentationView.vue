<template>
    <div class="documentation">
        <div class="page-header">
            <div class="container">
                <h1 class="page-title">Documentation</h1>
                <p class="page-subtitle">Learn how to use DataHarbor effectively</p>
            </div>
        </div>

        <div class="content-section">
            <div class="container">
                <el-row :gutter="32">
                    <el-col :span="24" :lg="18">
                        <div class="main-content">
                            <el-card class="doc-section" shadow="never">
                                <h2><el-icon>
                                        <Reading />
                                    </el-icon> Getting Started</h2>
                                <p>Welcome to DataHarbor! This guide will help you get started with browsing and
                                    managing your files.</p>

                                <h3>Quick Start</h3>
                                <ol class="step-list">
                                    <li>Log in with your credentials</li>
                                    <li>Navigate to the File Browser</li>
                                    <li>Browse through your directories</li>
                                    <li>Download files as needed</li>
                                </ol>
                            </el-card>

                            <el-card class="doc-section" shadow="never">
                                <h2><el-icon>
                                        <FolderOpened />
                                    </el-icon> File Browser</h2>
                                <p>The file browser provides an intuitive interface for navigating your file system:</p>

                                <ul class="feature-list">
                                    <li><strong>Navigation:</strong> Click on folders to navigate deeper into the
                                        directory structure</li>
                                    <li><strong>File Preview:</strong> View file details and metadata</li>
                                    <li><strong>Search:</strong> Use the search functionality to quickly find files</li>
                                    <li><strong>Download:</strong> Securely download files with proper authentication
                                    </li>
                                </ul>
                            </el-card>

                            <el-card class="doc-section" shadow="never">
                                <h2><el-icon>
                                        <Lock />
                                    </el-icon> Authentication</h2>
                                <p>DataHarbor uses secure authentication to protect your data:</p>

                                <ul class="feature-list">
                                    <li><strong>Secure Login:</strong> Your credentials are encrypted and protected</li>
                                    <li><strong>Session Management:</strong> Automatic session handling for seamless
                                        experience</li>
                                    <li><strong>Access Control:</strong> Only authorized users can access specific
                                        directories</li>
                                </ul>
                            </el-card>
                        </div>
                    </el-col>

                    <el-col :span="24" :lg="6">
                        <div class="sidebar-content">
                            <el-card class="quick-links" shadow="hover">
                                <h3>Quick Links</h3>
                                <ul class="link-list">
                                    <li><router-link to="/browse"><el-icon>
                                                <FolderOpened />
                                            </el-icon> File Browser</router-link></li>
                                    <li><router-link to="/about"><el-icon>
                                                <InfoFilled />
                                            </el-icon> About</router-link></li>
                                    <li><router-link to="/"><el-icon>
                                                <House />
                                            </el-icon> Home</router-link></li>
                                </ul>
                            </el-card>

                            <el-card class="help-section" shadow="hover">
                                <h3>Need Help?</h3>
                                <p>If you encounter any issues or have questions, please contact our support team.</p>
                                <el-button type="primary" size="small" style="width: 100%; margin-top: 1rem;">
                                    <el-icon>
                                        <Message />
                                    </el-icon>
                                    Contact Support
                                </el-button>
                            </el-card>
                        </div>
                    </el-col>
                </el-row>
            </div>
        </div>
    </div>
</template>

<script>
import { Reading, FolderOpened, Lock, InfoFilled, House, Message } from '@element-plus/icons-vue';

export default {
    name: 'DocumentationView',
    components: {
        Reading,
        FolderOpened,
        Lock,
        InfoFilled,
        House,
        Message
    }
};
</script>
<style lang="scss" scoped>
.documentation {
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

.main-content {
    .doc-section {
        margin-bottom: 2rem;
        border-radius: 12px;
        border: none;

        h2 {
            color: var(--el-text-color-primary);
            font-size: 1.5rem;
            font-weight: 600;
            margin-bottom: 1rem;
            display: flex;
            align-items: center;
            gap: 0.5rem;
        }

        h3 {
            color: var(--el-text-color-primary);
            font-size: 1.25rem;
            font-weight: 600;
            margin: 1.5rem 0 1rem 0;
        }

        p {
            color: var(--el-text-color-regular);
            line-height: 1.6;
            margin-bottom: 1rem;
        }
    }
}

.step-list {
    color: var(--el-text-color-regular);
    padding-left: 1.5rem;

    li {
        margin-bottom: 0.5rem;
        line-height: 1.6;
    }
}

.feature-list {
    color: var(--el-text-color-regular);
    padding-left: 1.5rem;

    li {
        margin-bottom: 0.75rem;
        line-height: 1.6;

        strong {
            color: var(--el-text-color-primary);
        }
    }
}

.sidebar-content {

    .quick-links,
    .help-section {
        margin-bottom: 2rem;
        border-radius: 12px;
        border: none;

        h3 {
            color: var(--el-text-color-primary);
            font-size: 1.25rem;
            font-weight: 600;
            margin-bottom: 1rem;
        }
    }

    .help-section p {
        color: var(--el-text-color-regular);
        font-size: 0.9rem;
        line-height: 1.5;
    }
}

.link-list {
    list-style: none;
    padding: 0;

    li {
        margin-bottom: 0.75rem;

        a {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            color: var(--el-text-color-primary);
            text-decoration: none;
            padding: 0.5rem;
            border-radius: 6px;
            transition: background-color 0.2s ease;

            &:hover {
                background-color: var(--el-color-primary-light-9);
                color: var(--el-color-primary);
            }
        }
    }
}

@media (max-width: 992px) {
    .sidebar-content {
        margin-top: 2rem;
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