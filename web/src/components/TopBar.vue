<template>
    <el-header class="top-bar" height="52px">
        <div class="top-bar-content">
            <!-- Left: Logo and Toggle Button -->
            <div class="left-section">
                <el-button class="toggle-btn" @click="toggleSidebar" text>
                    <el-icon>
                        <Expand v-if="isCollapsed" />
                        <Fold v-else />
                    </el-icon>
                </el-button>

                <div class="logo-section" @click="navigateTo('/')">
                    <img src="/assets/dataharbor-logo.svg" alt="DataHarbor" class="logo" />
                </div>
            </div>

            <!-- Right: User Section -->
            <div class="right-section">
                <!-- Login Button (when not authenticated) -->
                <div v-if="!isAuthenticated" class="login-section">
                    <el-button type="primary" @click="handleLogin" class="login-btn">
                        <el-icon class="mr-1">
                            <User />
                        </el-icon>
                        Sign In
                    </el-button>
                </div>

                <!-- User Profile (when authenticated) -->
                <div v-else class="user-profile">
                    <el-dropdown @command="handleUserAction" placement="bottom-end" :hide-on-click="false">
                        <div class="user-info">
                            <div class="user-details">
                                <div class="user-name">{{ userFullName || userLogin }}</div>
                                <div class="user-email">{{ userEmail }}</div>
                            </div>
                            <el-avatar :size="36" :src="userAvatar" :icon="User" class="user-avatar">
                                {{ userInitials }}
                            </el-avatar>
                            <el-icon class="dropdown-icon">
                                <ArrowDown />
                            </el-icon>
                        </div>
                        <template #dropdown>
                            <el-dropdown-menu>
                                <el-dropdown-item command="profile" disabled>
                                    <el-icon>
                                        <User />
                                    </el-icon>
                                    Profile
                                </el-dropdown-item>
                                <el-dropdown-item command="settings" disabled>
                                    <el-icon>
                                        <Setting />
                                    </el-icon>
                                    Settings
                                </el-dropdown-item>
                                <el-dropdown-item command="theme" disabled>
                                    <el-icon>
                                        <Moon />
                                    </el-icon>
                                    Dark Mode
                                </el-dropdown-item>
                                <el-dropdown-item divided command="logout">
                                    <el-icon>
                                        <SwitchButton />
                                    </el-icon>
                                    Sign Out
                                </el-dropdown-item>
                            </el-dropdown-menu>
                        </template>
                    </el-dropdown>
                </div>
            </div>
        </div>
    </el-header>
</template>

<script setup>
import { computed } from 'vue';
import { useRouter } from 'vue-router';
import { useAuth } from '../composables/useAuth';
import { ElMessageBox } from 'element-plus';
import {
    User,
    Setting,
    Moon,
    SwitchButton,
    Expand,
    Fold,
    ArrowDown
} from '@element-plus/icons-vue';

const props = defineProps({
    isCollapsed: {
        type: Boolean,
        required: true
    }
});

const emit = defineEmits(['toggle-sidebar']);

const router = useRouter();
const { isAuthenticated, user, logout } = useAuth();

// Computed properties for user info
const userLogin = computed(() => {
    if (!user.value) return 'User';
    return user.value.preferred_username ||
        user.value.email?.split('@')[0] ||
        user.value.sub ||
        'User';
});

const userFullName = computed(() => {
    if (!user.value) return '';
    return user.value.name ||
        (user.value.given_name && user.value.family_name
            ? `${user.value.given_name} ${user.value.family_name}`
            : user.value.given_name) ||
        '';
});

const userEmail = computed(() => {
    return user.value?.email || '';
});

const userInitials = computed(() => {
    if (!user.value) return '';

    if (user.value.name) {
        const names = user.value.name.split(' ');
        if (names.length >= 2) {
            return (names[0][0] + names[1][0]).toUpperCase();
        } else if (names.length === 1 && names[0].length > 0) {
            return names[0][0].toUpperCase();
        }
    }

    if (user.value.given_name && user.value.family_name) {
        return (user.value.given_name[0] + user.value.family_name[0]).toUpperCase();
    }

    if (user.value.preferred_username) {
        return user.value.preferred_username[0].toUpperCase();
    }

    if (user.value.email) {
        return user.value.email[0].toUpperCase();
    }

    return '';
});

const userAvatar = computed(() => {
    return user.value?.picture || '';
});

// Methods
const toggleSidebar = () => {
    emit('toggle-sidebar');
};

const navigateTo = (path) => {
    router.push(path);
};

const handleLogin = () => {
    router.push('/login');
};

const handleUserAction = async (command) => {
    switch (command) {
        case 'logout':
            try {
                await ElMessageBox.confirm(
                    'Are you sure you want to sign out?',
                    'Confirm Sign Out',
                    {
                        confirmButtonText: 'Sign Out',
                        cancelButtonText: 'Cancel',
                        type: 'warning'
                    }
                );
                await logout();
            } catch (error) {
                if (error !== 'cancel') {
                    console.error('Logout failed:', error);
                }
            }
            break;
        case 'profile':
        case 'settings':
        case 'theme':
            // These are disabled for now
            break;
    }
};
</script>

<style scoped>
.top-bar {
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-light);
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
    position: sticky;
    top: 0;
    z-index: 1000;
    padding: 0;
}

.top-bar-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 100%;
    padding: 0 16px;
    max-width: 100%;
}

.left-section {
    display: flex;
    align-items: center;
    gap: 10px;
}

.toggle-btn {
    width: 32px;
    height: 32px;
    border-radius: 6px;
    background: transparent;
    border: 1px solid var(--el-border-color-light);
    transition: all 0.2s ease;
}

.toggle-btn:hover {
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary);
}

.logo-section {
    cursor: pointer;
    transition: opacity 0.2s ease;
}

.logo-section:hover {
    opacity: 0.8;
}

.logo {
    height: 30px;
    width: auto;
    color: var(--el-color-primary);
}

.right-section {
    display: flex;
    align-items: center;
}

.login-section {
    display: flex;
    align-items: center;
}

.login-btn {
    height: 32px;
    padding: 0 14px;
    border-radius: 6px;
    font-weight: 500;
}

.user-profile {
    display: flex;
    align-items: center;
}

.user-info {
    display: flex;
    align-items: center;
    padding: 6px 10px;
    border-radius: 6px;
    cursor: pointer;
    transition: background-color 0.2s;
    gap: 10px;
}

.user-info:hover {
    background-color: var(--el-fill-color-light);
}

.user-details {
    text-align: right;
    min-width: 0;
}

.user-name {
    font-size: var(--dh-font-size-sm);
    font-weight: var(--dh-font-weight-medium);
    color: var(--el-text-color-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 150px;
}

.user-email {
    font-size: var(--dh-font-size-xs);
    color: var(--el-text-color-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 150px;
}

.user-avatar {
    background-color: var(--el-color-primary);
    color: #fff;
    flex-shrink: 0;
}

.dropdown-icon {
    color: var(--el-text-color-secondary);
    flex-shrink: 0;
    margin-left: 4px;
}

.mr-1 {
    margin-right: 4px;
}

/* Responsive adjustments */
@media (max-width: 768px) {
    .top-bar-content {
        padding: 0 12px;
    }

    .user-details {
        display: none;
    }

    .user-info {
        padding: 6px;
        gap: 6px;
    }

    .logo {
        height: 28px;
    }
}

/* Dark mode support */
.dark .top-bar {
    background: var(--el-bg-color);
    border-bottom-color: var(--el-border-color);
}
</style>
