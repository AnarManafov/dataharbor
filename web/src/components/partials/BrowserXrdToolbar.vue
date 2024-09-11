<template>
    <div class='toolbar'>
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
    initialPath: String
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
    display: flex;
    height: 100%;
}

.full-size-row {
    width: 100%;
    height: 100%;
}

.toolbar-right-content {
    display: flex;
    flex-direction: row;
    align-items: center;
    justify-content: end;
    height: 100%;
}

.toolbar-left-content {
    display: flex;
    flex-direction: row;
    align-items: center;
    justify-content: start;
    height: 100%;
}

.el-breadcrumb {
    font-size: 16px;
}

.el-row {
    margin-bottom: 20px;
}

.el-row:last-child {
    margin-bottom: 0;
}

.el-col {
    border-radius: 4px;
}

.grid-content {
    border-radius: 4px;
    min-height: 36px;
}
</style>