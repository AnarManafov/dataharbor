<template>
    <el-menu :default-opened="['2']" default-active='2'>
        <el-sub-menu index='1'>
            <template #title>
                <el-icon>
                    <IconMenu />
                </el-icon>Navigator One
            </template>
            <el-menu-item-group>
                <template #title>Group 1</template>
                <el-menu-item index='1-1'>Option 1</el-menu-item>
                <el-menu-item index='1-2'>Option 2</el-menu-item>
            </el-menu-item-group>
            <el-menu-item-group title='Group 2'>
                <el-menu-item index='1-3'>Option 3</el-menu-item>
            </el-menu-item-group>
            <el-sub-menu index='1-4'>
                <template #title>Option4</template>
                <el-menu-item index='1-4-1'>Option 4-1</el-menu-item>
            </el-sub-menu>
        </el-sub-menu>
        <el-sub-menu index='2'>
            <template #title>
                <el-icon>
                    <Setting />
                </el-icon>Settings
            </template>
            <el-menu-item-group>
                <template #title>Group 1</template>
                <el-menu-item index='2-1'>Option 1</el-menu-item>
                <el-menu-item index='2-2'>Option 2</el-menu-item>
            </el-menu-item-group>
            <el-menu-item-group title='Group 2'>
                <el-menu-item index='2-3'>Option 3</el-menu-item>
            </el-menu-item-group>
            <el-sub-menu index='2-4'>
                <template #title>Option 4</template>
                <el-menu-item index='2-4-1'>Option 4-1</el-menu-item>
            </el-sub-menu>
        </el-sub-menu>
    </el-menu>
</template>

<script lang="ts" setup>
import { Menu as IconMenu, Setting } from '@element-plus/icons-vue';
</script>

<style scoped>
.el-menu {
    border-right: none;
}
</style>