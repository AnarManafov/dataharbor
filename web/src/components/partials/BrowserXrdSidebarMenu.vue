<template>
    <el-menu :default-opened="['1']" default-active='1'>
        <el-sub-menu index='1'>
            <template #title>
                <el-icon>
                    <IconMenu />
                </el-icon>Navigation
            </template>
            <el-menu-item index='1-1' @click="$emit('navigateToHome')">Home Directory</el-menu-item>
            <el-menu-item index='1-2' @click="$emit('navigateToFavorites')">Favorites</el-menu-item>
            <el-menu-item index='1-3' @click="$emit('navigateToRecent')">Recently Accessed</el-menu-item>
        </el-sub-menu>
        <el-sub-menu index='2'>
            <template #title>
                <el-icon>
                    <Setting />
                </el-icon>Settings
            </template>
            <el-menu-item index='2-1' @click="$emit('openPreferences')">Preferences</el-menu-item>
            <el-menu-item index='2-2' @click="$emit('toggleDarkMode')">Dark Mode</el-menu-item>
        </el-sub-menu>
    </el-menu>
</template>

<script lang="ts" setup>
import { Menu as IconMenu, Setting } from '@element-plus/icons-vue';

// Define emits for navigation events
defineEmits([
    'navigateToHome',
    'navigateToFavorites',
    'navigateToRecent',
    'openPreferences',
    'toggleDarkMode'
]);
</script>

<style scoped>
.el-menu {
    border-right: none;
}
</style>