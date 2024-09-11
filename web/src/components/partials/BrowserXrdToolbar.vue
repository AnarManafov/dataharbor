<template>
    <div class='toolbar'>
        <!-- First Row -->
        <el-row class='full-size-row'>
            <el-col :span='12' class='toolbar-left-content'>
                <div>
                    <el-tooltip class='box-item' effect='dark' :content='serviceStatusTooltip' placement='bottom-start'>
                        <el-icon :style='{ color: serviceStatusColor }' @click='changeDirToInitialPath' :size='18'
                            style='margin-right: 5px; margin-top: 3px'>
                            <HomeFilled />
                        </el-icon>
                    </el-tooltip>
                </div>
                <div>
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
            <el-col :span='12' class='toolbar-right-content'>
                <div style='font-size: 12px;'>
                    Data Server Host: <span style='font-weight: bold;'>{{ xrdHostName }}</span>
                </div>
                <div style='font-size: 12px;'>
                    Initial Path: <span style='font-weight: bold;'>{{ initialPath }}</span>
                </div>
            </el-col>
        </el-row>
        <!-- Second Row -->
        <el-row class='full-size-row second-row'>
            <el-col :span='24' class='toolbar-left-content'>
                <div>
                    Folders: <span style='font-weight: bold;'>{{ folderCount }}</span>
                </div>
                <div>
                    Files: <span style='font-weight: bold;'>{{ fileCount }} </span> (cumulative file size: <span
                        style='font-weight: bold;'>{{ totalFileSize
                        }}</span>)
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
    xrdHostName: String,
    currentDirectory: String,
    initialPath: String,
    folderCount: Number,
    fileCount: Number,
    totalFileSize: String
});

const emit = defineEmits(['changeDirToInitialPath', 'changeDir']);

const changeDirToInitialPath = () => {
    emit('changeDirToInitialPath');
};

const changeDir = (index: number) => {
    emit('changeDir', index);
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
    /* height: 100%;*/
}

.toolbar-right-content {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    justify-content: start;
    height: 100%;
}

.el-breadcrumb {
    font-size: 16px;
}

.second-row {
    font-size: 12px;
    /* Adjust this value to move the second row up or down */
    margin-top: 10px;
}

.second-row .toolbar-left-content>div {
    /* Adjust this value to control the spacing between folder and file counts */
    margin-bottom: 5px;
    /* Add space between folder and file counters */
    margin-right: 10px;
}
</style>