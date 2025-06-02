<template>
    <div class="layout-container">
        <!-- Top Bar -->
        <TopBar :is-collapsed="isCollapsed" @toggle-sidebar="toggleSidebar" />

        <!-- Main Layout -->
        <div class="main-layout">
            <!-- Sidebar -->
            <el-aside :width="isCollapsed ? '64px' : '280px'" class="sidebar" :class="{ 'collapsed': isCollapsed }">
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
            </el-aside>

            <!-- Main Content Area -->
            <el-main class="main-content">
                <slot />
            </el-main>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import TopBar from './TopBar.vue';
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
}

.sidebar-content {
    display: flex;
    flex-direction: column;
    height: 100%;
    padding: 0;
}

.navigation-section {
    flex: 1;
    overflow-y: auto;
    padding: 16px 0;
}

.sidebar-menu {
    border: none;
    background: transparent;
}

.sidebar-menu .el-menu-item {
    margin: 2px 8px;
    border-radius: 6px;
    height: 40px;
    line-height: 40px;
}

.sidebar-menu .el-menu-item:hover {
    background-color: var(--el-color-primary-light-9);
    color: var(--el-color-primary);
}

.sidebar-menu .el-menu-item.is-active {
    background-color: var(--el-color-primary-light-8);
    color: var(--el-color-primary);
    font-weight: 500;
}

.sidebar-menu .el-sub-menu .el-sub-menu__title {
    margin: 2px 8px;
    border-radius: 6px;
    height: 40px;
    line-height: 40px;
}

.sidebar-menu .el-menu-item[disabled] {
    opacity: 0.5;
    cursor: not-allowed;
}

.main-content {
    flex: 1;
    padding: 0;
    overflow: auto;
    background: var(--el-bg-color-page);
}

/* Responsive adjustments */
@media (max-width: 768px) {
    .sidebar {
        position: fixed;
        height: calc(100vh - 60px);
        top: 60px;
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
