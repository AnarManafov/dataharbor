<template>
    <div class="documentation">
        <div class="page-header">
            <div class="container">
                <h1 class="page-title">Documentation</h1>
                <p class="page-subtitle">A quick guide to browsing and downloading your storage data</p>
            </div>
        </div>

        <div class="content-section">
            <div class="container">
                <el-card class="doc-section" shadow="never">
                    <h2><el-icon>
                            <Reading />
                        </el-icon> Getting Started</h2>
                    <p>
                        DataHarbor lets you browse and download files on your storage cluster
                        directly from your web browser. No special client software is required.
                    </p>

                    <ol class="step-list">
                        <li>
                            <strong>Sign in</strong> — click <em>{{ branding.loginButtonText }}</em> on the
                            login page. You will be redirected to your institutional identity provider (OIDC/SSO).
                        </li>
                        <li>
                            <strong>Open the File Browser</strong> — after login you are taken to your
                            initial directory on the cluster automatically.
                        </li>
                        <li>
                            <strong>Navigate</strong> — click any folder to open it. Use the breadcrumb
                            bar at the top or the <code>..</code> entry in the table to go back up.
                        </li>
                        <li>
                            <strong>Download</strong> — click the download button next to a file. A
                            confirmation dialog appears, then the file streams directly to your disk.
                        </li>
                    </ol>
                </el-card>

                <el-card class="doc-section" shadow="never">
                    <h2><el-icon>
                            <FolderOpened />
                        </el-icon> File Browser</h2>
                    <p>
                        The file browser shows the contents of your current directory in a sortable table.
                    </p>

                    <h3>Toolbar</h3>
                    <ul class="feature-list">
                        <li>
                            <strong>Breadcrumb navigation</strong> — each path segment is clickable,
                            letting you jump to any parent folder instantly.
                        </li>
                        <li>
                            <strong>Home button</strong> — returns you to your initial directory.
                            A colored indicator shows whether the backend service is online.
                        </li>
                        <li>
                            <strong>Storage statistics</strong> — the toolbar displays free space and
                            current utilization of the storage nodes. Utilization is color-coded: green
                            (&lt;70%), yellow (70-89%), and red (90%+).
                        </li>
                        <li>
                            <strong>Page stats</strong> — a second row shows the number of folders, files,
                            and total size for both the current page and the full directory.
                        </li>
                        <li>
                            <strong>Network performance</strong> — a third row displays live connection
                            metrics: XRD server latency, average download speed (based on recent
                            transfers), and the last directory query time. These update automatically
                            at regular intervals.
                        </li>
                    </ul>

                    <h3>File Table</h3>
                    <ul class="feature-list">
                        <li>
                            <strong>Columns</strong> — Name, Size, Date modified, and Type. Click any
                            column header to sort ascending or descending.
                        </li>
                        <li>
                            <strong>Pagination</strong> — large directories are split into pages of 500
                            items. Use the page controls at the bottom to move between pages or jump to a
                            specific one.
                        </li>
                        <li>
                            <strong>Parent directory (..) </strong> — always appears at the top of the
                            table (except when you are already at your initial directory).
                        </li>
                    </ul>
                </el-card>

                <el-card class="doc-section" shadow="never">
                    <h2><el-icon>
                            <Download />
                        </el-icon> Downloads</h2>
                    <p>
                        DataHarbor streams files directly from the XRootD storage layer to your browser
                        using chunked transfer. This means even multi-gigabyte files download without
                        consuming browser memory.
                    </p>

                    <ul class="feature-list">
                        <li>
                            <strong>Starting a download</strong> — click the download icon in the
                            <em>Actions</em> column, then confirm in the dialog that appears. If previous
                            downloads have been completed, the button tooltip shows an estimated download
                            time and speed for that file.
                        </li>
                        <li>
                            <strong>Download Manager</strong> — open the drawer from the top bar to see
                            all active and completed downloads, including speed, progress, and total size.
                        </li>
                        <li>
                            <strong>Completion</strong> — once finished, a notification shows the
                            transfer speed, duration, and file size.
                        </li>
                        <li>
                            <strong>Errors</strong> — if a download fails (network issue, expired
                            session, missing file), an error message explains the cause. You can retry
                            from the Download Manager.
                        </li>
                    </ul>
                </el-card>

                <el-card class="doc-section" shadow="never">
                    <h2><el-icon>
                            <Lock />
                        </el-icon> Authentication</h2>
                    <p>
                        DataHarbor uses OpenID Connect (OIDC) single sign-on. Your institutional
                        identity provider handles the login — DataHarbor never sees your password.
                    </p>

                    <ul class="feature-list">
                        <li>
                            <strong>Sign in</strong> — you are redirected to your identity provider, then
                            back to DataHarbor. A secure, HTTP-only session cookie keeps you logged in.
                        </li>
                        <li>
                            <strong>Session management</strong> — your session is maintained
                            automatically. If it expires, you will be prompted to sign in again.
                        </li>
                        <li>
                            <strong>Sign out</strong> — use the user menu in the top-right corner to log
                            out at any time. Your session is cleared immediately.
                        </li>
                    </ul>
                </el-card>

                <div class="quick-links-row">
                    <router-link to="/browse" class="quick-link">
                        <el-icon>
                            <FolderOpened />
                        </el-icon> File Browser
                    </router-link>
                    <router-link to="/about" class="quick-link">
                        <el-icon>
                            <InfoFilled />
                        </el-icon> About
                    </router-link>
                    <router-link to="/" class="quick-link">
                        <el-icon>
                            <House />
                        </el-icon> Home
                    </router-link>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import { Reading, FolderOpened, Lock, Download, InfoFilled, House } from '@element-plus/icons-vue';
import { getConfig } from '@/config/config';

export default {
    name: 'DocumentationView',
    components: {
        Reading,
        FolderOpened,
        Lock,
        Download,
        InfoFilled,
        House,
    },
    data() {
        const config = getConfig();
        return {
            branding: config.branding || {},
        };
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
        max-width: 860px;
        margin: 0 auto;
        padding: 0 2rem;
    }
}

.doc-section {
    margin-bottom: 1.5rem;
    border-radius: 12px;
    border: none;

    h2 {
        color: var(--el-text-color-primary);
        font-size: 1.35rem;
        font-weight: 600;
        margin-bottom: 0.75rem;
        display: flex;
        align-items: center;
        gap: 0.5rem;
    }

    h3 {
        color: var(--el-text-color-primary);
        font-size: 1.1rem;
        font-weight: 600;
        margin: 1.25rem 0 0.75rem 0;
    }

    p {
        color: var(--el-text-color-regular);
        line-height: 1.7;
        margin-bottom: 0.75rem;
        font-size: 0.975rem;
    }

    code {
        background: var(--el-fill-color-light);
        padding: 0.1em 0.4em;
        border-radius: 4px;
        font-size: 0.9em;
    }
}

.step-list {
    color: var(--el-text-color-regular);
    padding-left: 1.5rem;
    font-size: 0.975rem;

    li {
        margin-bottom: 0.6rem;
        line-height: 1.65;

        strong {
            color: var(--el-text-color-primary);
        }
    }
}

.feature-list {
    color: var(--el-text-color-regular);
    padding-left: 1.5rem;
    font-size: 0.975rem;

    li {
        margin-bottom: 0.6rem;
        line-height: 1.65;

        strong {
            color: var(--el-text-color-primary);
        }
    }
}

.quick-links-row {
    display: flex;
    justify-content: center;
    gap: 1rem;
    margin-top: 0.5rem;
}

.quick-link {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    padding: 0.5rem 1.25rem;
    border-radius: 8px;
    font-size: 0.925rem;
    color: var(--el-text-color-primary);
    text-decoration: none;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    transition: background-color 0.2s ease, color 0.2s ease;

    &:hover {
        background-color: var(--el-color-primary-light-9);
        color: var(--el-color-primary);
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

    .quick-links-row {
        flex-direction: column;
        align-items: center;
    }
}
</style>
