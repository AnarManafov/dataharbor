<template>
    <div class='toolbar'>
        <!-- First Row -->
        <el-row class='full-size-row'>
            <el-col :span='19' class='toolbar-left-content'>
                <div>
                    <el-tooltip class='box-item' effect='dark' :content='serviceStatusTooltip' placement='bottom-start'>
                        <el-icon :style='{ color: serviceStatusColor }' @click='changeDirToInitialPath' :size='16'
                            style='margin-right: 5px; margin-top: 3px'>
                            <HomeFilled />
                        </el-icon>
                    </el-tooltip>
                </div>
                <div class='breadcrumb-container'>
                    <el-breadcrumb separator='/'>
                        <el-breadcrumb-item @click='changeDirToInitialPath'><a>Initial
                                Directory</a></el-breadcrumb-item>
                        <template v-for="(item, index) in currentDirectory.replace(initialPath, '').split('/')"
                            :key='index'>
                            <el-breadcrumb-item @click='() => changeDir(index)' v-if='item.length > 0'>
                                <a>{{ item }}</a>
                            </el-breadcrumb-item>
                        </template>
                    </el-breadcrumb>
                </div>
            </el-col>
            <el-col :span='5' class='toolbar-right-content'>
                <div class='storage-stats' v-if='vfsStat'>
                    <el-tooltip effect='dark' placement='bottom'
                        :content='`${vfsStat.nodesRW} R/W node(s), ${vfsStat.nodesStaging} staging node(s)`'>
                        <span class='stat-item'>
                            <span class='stat-label'>Free:</span>
                            <span class='stat-value'>{{ formatFreeSpace(vfsStat.freeSpaceMB) }}</span>
                        </span>
                    </el-tooltip>
                    <span class='stat-separator'>|</span>
                    <span class='stat-item'>
                        <span class='stat-label'>Used:</span>
                        <span class='stat-value' :class='utilizationClass(vfsStat.utilizationPercent)'>{{
                            vfsStat.utilizationPercent
                            }}%</span>
                    </span>
                </div>
                <div style='font-size: 10px; color: var(--el-text-color-secondary); margin-top: 2px;'
                    :title='currentDirectory'>
                    {{ currentDirectory }}
                </div>
            </el-col>
        </el-row>
        <!-- Second Row -->
        <el-row class='full-size-row second-row'>
            <el-col :span='24' class='toolbar-left-content column-layout'>
                <div class="current-page-stats">
                    Current page: <span style='font-weight: bold'>{{ folderCount + fileCount }}</span> (<span
                        style='font-weight: bold; color: var(--el-color-primary)'>{{ folderCount }} folders</span>,
                    <span style='font-weight: bold; color: var(--el-color-success)'>{{ fileCount }} files</span>),
                    cumulative
                    file size:
                    <span style='font-weight: bold;'>{{ totalOnPageFileSize }}</span>
                </div>
                <div class="total-stats">
                    Total: <span style='font-weight: bold'>{{ totalFileCount + totalFolderCount }}</span> (<span
                        style='font-weight: bold; color: var(--el-color-primary)'>{{ totalFolderCount }} folders</span>,
                    <span style='font-weight: bold; color: var(--el-color-success)'>{{ totalFileCount }} files</span>),
                    cumulative file
                    size: <span style='font-weight: bold;'>{{ totalFileSize }}</span>
                </div>
            </el-col>
        </el-row>
    </div>
</template>

<script lang="ts" setup>
import { HomeFilled } from '@element-plus/icons-vue';

const props = defineProps({
    serviceStatusTooltip: String,
    serviceStatusColor: String,
    currentDirectory: String,
    initialPath: String,
    folderCount: Number,
    fileCount: Number,
    totalOnPageFileSize: String,
    totalFolderCount: Number,
    totalFileCount: Number,
    totalFileSize: String,
    vfsStat: Object
});

const emit = defineEmits(['changeDirToInitialPath', 'changeDir']);

const changeDirToInitialPath = () => {
    emit('changeDirToInitialPath');
};

const changeDir = (index: number) => {
    emit('changeDir', index);
};

const formatFreeSpace = (mb: number): string => {
    if (mb >= 1024 * 1024) {
        return `${(mb / (1024 * 1024)).toFixed(1)} TB`;
    } else if (mb >= 1024) {
        return `${(mb / 1024).toFixed(1)} GB`;
    }
    return `${mb} MB`;
};

const utilizationClass = (percent: number): string => {
    if (percent >= 90) return 'utilization-critical';
    if (percent >= 70) return 'utilization-warning';
    return 'utilization-ok';
};
</script>

<style scoped>
.toolbar {
    padding: 10px;
}

.full-size-row {
    width: 100%;
}

.toolbar-left-content {
    display: flex;
    flex-direction: row;
    align-items: center;
    justify-content: start;
    min-width: 0;
}

.toolbar-right-content {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    justify-content: start;
    height: 100%;
}

.breadcrumb-container {
    flex: 1 1 0%;
    min-width: 0;
    /* Ensures the breadcrumb can shrink/grow as needed */
    display: flex;
    align-items: center;
}

.el-breadcrumb {
    font-size: 16px;
    width: 100%;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.second-row {
    font-size: 12px;
    /* Adjust this value to move the second row up or down */
    margin-top: 10px;
}

.second-row .toolbar-left-content>div {
    font-size: 12px;
    /* Adjust this value to control the spacing between folder and file counts */
    margin-bottom: 5px;
    /* Add space between folder and file counters */
    margin-right: 10px;
}

.column-layout {
    flex-direction: column;
    align-items: flex-start;
}

/* Storage stats styling */
.storage-stats {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 11px;
    white-space: nowrap;
}

.stat-item {
    display: inline-flex;
    align-items: center;
    gap: 3px;
}

.stat-label {
    color: var(--el-text-color-secondary);
}

.stat-value {
    font-weight: bold;
    color: var(--el-text-color-primary);
}

.stat-separator {
    color: var(--el-border-color);
}

.utilization-ok {
    color: var(--el-color-success);
}

.utilization-warning {
    color: var(--el-color-warning);
}

.utilization-critical {
    color: var(--el-color-danger);
}
</style>
