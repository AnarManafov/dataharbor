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
            <el-col :span='24' class='page-stats-bar'>
                <span class='net-stat-item'>
                    <span class='net-stat-label'>Showing:</span>
                    <span class='net-stat-value' style='color: var(--el-color-primary)'>{{ folderCount
                    }}&nbsp;folders</span>
                    <span class='net-stat-dot'>&middot;</span>
                    <span class='net-stat-value' style='color: var(--el-color-success)'>{{ fileCount
                    }}&nbsp;files</span>
                    <span class='net-stat-dot'>&middot;</span>
                    <span class='net-stat-value'>{{ totalOnPageFileSize }}</span>
                </span>
                <span class='net-stat-separator'>|</span>
                <span class='net-stat-item'>
                    <span class='net-stat-label'>Total:</span>
                    <span class='net-stat-value' style='color: var(--el-color-primary)'>{{ totalFolderCount
                    }}&nbsp;folders</span>
                    <span class='net-stat-dot'>&middot;</span>
                    <span class='net-stat-value' style='color: var(--el-color-success)'>{{ totalFileCount
                    }}&nbsp;files</span>
                    <span class='net-stat-dot'>&middot;</span>
                    <span class='net-stat-value'>{{ totalFileSize }}</span>
                </span>
            </el-col>
        </el-row>
        <!-- Third Row: Network Performance Stats -->
        <el-row class='full-size-row third-row' v-if='hasNetworkData'>
            <el-col :span='24' class='network-stats-bar'>
                <el-tooltip effect='dark' placement='bottom' :content='latencyTooltip'>
                    <span class='net-stat-item'>
                        <span class='net-stat-icon' :style='{ color: latencyColor }'>&#9679;</span>
                        <span class='net-stat-label'>Latency:</span>
                        <span class='net-stat-value'>{{ latencyDisplay }}</span>
                    </span>
                </el-tooltip>
                <span class='net-stat-separator'>|</span>
                <el-tooltip effect='dark' placement='bottom' content='Average download speed based on recent transfers'>
                    <span class='net-stat-item'>
                        <span class='net-stat-label'>Avg Speed:</span>
                        <span class='net-stat-value'>{{ avgSpeedDisplay }}</span>
                    </span>
                </el-tooltip>
                <span class='net-stat-separator'>|</span>
                <el-tooltip effect='dark' placement='bottom'
                    content='Time the XRD server took to respond to the last directory listing'>
                    <span class='net-stat-item'>
                        <span class='net-stat-label'>Query:</span>
                        <span class='net-stat-value'>{{ queryTimeDisplay }}</span>
                    </span>
                </el-tooltip>
                <span class='net-stat-separator' v-if='downloadCount > 0'>|</span>
                <el-tooltip effect='dark' placement='bottom'
                    content='Number of completed downloads used for speed estimation' v-if='downloadCount > 0'>
                    <span class='net-stat-item'>
                        <span class='net-stat-label'>Samples:</span>
                        <span class='net-stat-value'>{{ downloadCount }}</span>
                    </span>
                </el-tooltip>
            </el-col>
        </el-row>
    </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue';
import { HomeFilled } from '@element-plus/icons-vue';
import { useNetworkStats } from '@/composables/useNetworkStats';

const { latencyMs, connectMs, latencyQuality, avgSpeedFormatted, queryTimeMs, downloadSpeeds } = useNetworkStats();

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
    if (mb >= 1024 * 1024 * 1024) {
        return `${(mb / (1024 * 1024 * 1024)).toFixed(1)} PB`;
    } else if (mb >= 1024 * 1024) {
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

// Network stats computed properties
const hasNetworkData = computed(() => latencyMs.value !== null || queryTimeMs.value !== null || downloadSpeeds.value.length > 0);

const latencyDisplay = computed(() => {
    if (latencyMs.value === null) return 'Measuring...';
    return `${latencyMs.value} ms`;
});

const latencyColor = computed(() => latencyQuality.value.color);

const latencyTooltip = computed(() => {
    if (latencyMs.value === null) return 'Measuring XRD server latency...';
    const conn = connectMs.value !== null ? ` | Connect: ${connectMs.value} ms` : '';
    return `XRD server round-trip: ${latencyMs.value} ms (${latencyQuality.value.label})${conn}`;
});

const avgSpeedDisplay = computed(() => avgSpeedFormatted.value || 'No data');

const queryTimeDisplay = computed(() => {
    if (queryTimeMs.value === null) return 'N/A';
    return `${queryTimeMs.value} ms`;
});

const downloadCount = computed(() => downloadSpeeds.value.length);
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
    margin-top: 10px;
}

.page-stats-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 11px;
    white-space: nowrap;
    color: var(--el-text-color-secondary);
}

.net-stat-dot {
    color: var(--el-border-color);
    margin: 0 1px;
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

/* Network stats row styling */
.third-row {
    margin-top: 6px;
    padding-top: 6px;
    border-top: 1px dashed var(--el-border-color-lighter);
}

.network-stats-bar {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 11px;
    white-space: nowrap;
    color: var(--el-text-color-secondary);
}

.net-stat-item {
    display: inline-flex;
    align-items: baseline;
    gap: 4px;
    cursor: default;
    white-space: nowrap;
}

.net-stat-icon {
    font-size: 8px;
    margin-right: 2px;
}

.net-stat-label {
    color: var(--el-text-color-secondary);
    margin-right: 1px;
}

.net-stat-value {
    font-weight: bold;
    color: var(--el-text-color-primary);
    font-family: var(--dh-font-family-mono, monospace);
}

.net-stat-separator {
    color: var(--el-border-color);
}
</style>
