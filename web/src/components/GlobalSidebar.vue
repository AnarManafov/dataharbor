<template>
    <div class="layout-container">
        <!-- Top Bar -->
        <TopBar :is-collapsed="isCollapsed" @toggle-sidebar="toggleSidebar" />

        <!-- Main Layout -->
        <div class="main-layout">
            <!-- Sidebar -->
            <el-aside :width="isCollapsed ? '56px' : '200px'" class="sidebar" :class="{ 'collapsed': isCollapsed }">
                <div class="sidebar-content">
                    <!-- Navigation Menu -->
                    <div class="navigation-section">
                        <el-menu :default-active="activeMenu" :collapse="isCollapsed" :collapse-transition="false"
                            router class="sidebar-menu">
                            <!-- Main Navigation -->
                            <el-menu-item index="/" @click="navigateTo('/')">
                                <el-icon>
                                    <House />
                                </el-icon>
                                <template #title>Home</template>
                            </el-menu-item>

                            <el-menu-item index="/browse" @click="navigateTo('/browse')">
                                <el-icon>
                                    <FolderOpened />
                                </el-icon>
                                <template #title>File Browser</template>
                            </el-menu-item>

                            <el-menu-item index="/docs" @click="navigateTo('/docs')">
                                <el-icon>
                                    <Document />
                                </el-icon>
                                <template #title>Documentation</template>
                            </el-menu-item>

                            <el-menu-item index="/about" @click="navigateTo('/about')">
                                <el-icon>
                                    <InfoFilled />
                                </el-icon>
                                <template #title>About</template>
                            </el-menu-item>

                            <!-- Divider -->
                            <el-divider v-if="!isCollapsed" />

                            <!-- Quick Actions (Disabled for now) -->
                            <el-sub-menu index="quick" v-if="!isCollapsed" disabled>
                                <template #title>
                                    <el-icon>
                                        <Star />
                                    </el-icon>
                                    <span>Quick Actions</span>
                                </template>
                                <el-menu-item index="favorites" disabled>
                                    <el-icon>
                                        <StarFilled />
                                    </el-icon>
                                    <template #title>Favorites</template>
                                </el-menu-item>
                                <el-menu-item index="recent" disabled>
                                    <el-icon>
                                        <Clock />
                                    </el-icon>
                                    <template #title>Recent Items</template>
                                </el-menu-item>
                            </el-sub-menu>
                        </el-menu>
                    </div>
                </div>

                <!-- Version label at bottom -->
                <div class="sidebar-footer" v-if="!isCollapsed">
                    <span class="version-label">v{{ appVersion }}</span>
                </div>
            </el-aside>

            <!-- Main Content Area -->
            <el-main class="main-content">
                <div class="main-content-inner">
                    <slot />
                </div>
                <GlobalFooter />
            </el-main>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import TopBar from './TopBar.vue';
import GlobalFooter from './GlobalFooter.vue';
import { getAppVersion } from '@/utils/version';
import {
    House,
    FolderOpened,
    Document,
    InfoFilled,
    Star,
    StarFilled,
    Clock
} from '@element-plus/icons-vue';

const router = useRouter();
const route = useRoute();

// Sidebar state
const isCollapsed = ref(false);
const appVersion = getAppVersion();

// Active menu computation
const activeMenu = computed(() => {
    return route.path;
});

// Methods
const toggleSidebar = () => {
    isCollapsed.value = !isCollapsed.value;
};

const navigateTo = (path) => {
    router.push(path);
};

// Store sidebar state in localStorage
onMounted(() => {
    const savedState = localStorage.getItem('sidebar-collapsed');
    if (savedState !== null) {
        isCollapsed.value = JSON.parse(savedState);
    }
});

// Watch for sidebar state changes and save to localStorage
watch(isCollapsed, (newValue) => {
    localStorage.setItem('sidebar-collapsed', JSON.stringify(newValue));
});
</script>

<style scoped>
.layout-container {
    display: flex;
    flex-direction: column;
    height: 100vh;
}

.main-layout {
    display: flex;
    flex: 1;
    min-height: 0;
}

.sidebar {
    background: var(--el-bg-color);
    border-right: 1px solid var(--el-border-color-light);
    transition: width 0.3s ease;
    position: relative;
    z-index: 999;
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.sidebar-content {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    padding: 0;
    overflow: hidden;
}

.navigation-section {
    flex: 1;
    overflow-y: auto;
    padding: 12px 0;
}

.sidebar-menu {
    border: none;
    background: transparent;
}

.sidebar-menu .el-menu-item {
    margin: 1px 6px;
    border-radius: 6px;
    height: 40px;
    line-height: 40px;
    font-size: var(--dh-font-size-sm);
    font-weight: var(--dh-font-weight-normal);
}

.sidebar-menu .el-menu-item:hover {
    background-color: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
}

.sidebar-menu .el-menu-item.is-active {
    background-color: var(--el-color-primary-light-8);
    color: var(--el-color-primary);
    font-weight: var(--dh-font-weight-medium);
}

.sidebar-menu .el-sub-menu .el-sub-menu__title {
    margin: 1px 6px;
    border-radius: 6px;
    height: 40px;
    line-height: 40px;
    font-size: var(--dh-font-size-sm);
    font-weight: var(--dh-font-weight-normal);
}

.sidebar-menu .el-menu-item[disabled] {
    opacity: 0.5;
    cursor: not-allowed;
}

.sidebar-footer {
    padding: 8px 12px;
    text-align: center;
    border-top: 1px solid var(--el-border-color-lighter);
    flex-shrink: 0;
}

.version-label {
    font-size: 10px;
    color: var(--el-text-color-placeholder);
}

.main-content {
    flex: 1;
    padding: 0;
    overflow: auto;
    background: var(--el-bg-color-page);
    display: flex;
    flex-direction: column;
}

.main-content-inner {
    flex: 1 0 auto;
}

/* Responsive adjustments */
@media (max-width: 768px) {
    .sidebar {
        position: fixed;
        height: calc(100vh - 52px);
        top: 52px;
        z-index: 2000;
    }

    .sidebar.collapsed {
        transform: translateX(-100%);
    }

    .main-content {
        margin-left: 0;
    }
}

/* Dark mode support */
.dark .sidebar {
    background: var(--el-bg-color);
    border-right-color: var(--el-border-color);
}
</style>
