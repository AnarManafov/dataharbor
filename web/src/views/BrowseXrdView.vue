<template>
    <div>
        <el-divider />
    </div>
    <el-container class="layout-file-tree-container">
        <el-container>
            <el-aside width="200px">
                <el-scrollbar>
                    <el-menu :default-openeds="['2']" default-active="2">
                        <el-sub-menu index="1">
                            <template #title>
                                <el-icon>
                                    <IconMenu />
                                </el-icon>Navigator One
                            </template>
                            <el-menu-item-group>
                                <template #title>Group 1</template>
                                <el-menu-item index="1-1">Option 1</el-menu-item>
                                <el-menu-item index="1-2">Option 2</el-menu-item>
                            </el-menu-item-group>
                            <el-menu-item-group title="Group 2">
                                <el-menu-item index="1-3">Option 3</el-menu-item>
                            </el-menu-item-group>
                            <el-sub-menu index="1-4">
                                <template #title>Option4</template>
                                <el-menu-item index="1-4-1">Option 4-1</el-menu-item>
                            </el-sub-menu>
                        </el-sub-menu>
                        <el-sub-menu index="2">
                            <template #title>
                                <el-icon>
                                    <setting />
                                </el-icon>Settings
                            </template>
                            <el-menu-item-group>
                                <template #title>Group 1</template>
                                <el-menu-item index="2-1">Option 1</el-menu-item>
                                <el-menu-item index="2-2">Option 2</el-menu-item>
                            </el-menu-item-group>
                            <el-menu-item-group title="Group 2">
                                <el-menu-item index="2-3">Option 3</el-menu-item>
                            </el-menu-item-group>
                            <el-sub-menu index="2-4">
                                <template #title>Option 4</template>
                                <el-menu-item index="2-4-1">Option 4-1</el-menu-item>
                            </el-sub-menu>
                        </el-sub-menu>
                    </el-menu>
                </el-scrollbar>
            </el-aside>
            <el-container>
                <el-header>
                    <div class="toolbar">
                        <el-row class="full-size-row" justify="space-between">
                            <el-col :span="12" class="toolbar-left-content">
                                <div>
                                    <el-breadcrumb separator="/">
                                        <el-breadcrumb-item>{{ xrdHostName }}:</el-breadcrumb-item>
                                        <template v-for="(item, index) in currentDir.split('/')" :key="index">
                                            <el-breadcrumb-item @click="changeDir(index)" v-if="item.length > 0">
                                                <a href="#">{{ item }}</a>
                                            </el-breadcrumb-item>
                                        </template>
                                    </el-breadcrumb>
                                </div>
                            </el-col>
                            <el-col :span="12" class="toolbar-right-content">
                                <div>
                                    <span>Second Column Items Placeholder</span>
                                </div>
                            </el-col>
                        </el-row>
                    </div>
                </el-header>
                <el-container>
                    <el-main>
                        <el-scrollbar>
                            <el-table :data="tableData" :row-class-name="tableRowClassName"
                                :default-sort="{ prop: 'name', order: 'ascending' }" border>
                                <el-table-column prop="name" label="Name" sortable>
                                    <template #default="scope">
                                        <span class="clickable" @click="selectDir(scope.row)">{{ scope.row.name
                                            }}</span>
                                    </template>
                                </el-table-column>
                                <el-table-column prop="size" label="Size" sortable width="150" />
                                <el-table-column prop="date_time" label="Date" sortable width="200" />
                                <el-table-column prop="type" label="Type" sortable width="80" />
                            </el-table>
                        </el-scrollbar>
                    </el-main>
                </el-container>
            </el-container>
        </el-container>
    </el-container>
</template>


<script lang="ts" setup>
import { getHostName, getHomeDirPath, getItemsInDir, getFileStagedForDownload } from '@/api/api';
import { onMounted, ref } from 'vue';
import { saveAs } from 'file-saver';
import axios from 'axios';
import { Menu as IconMenu, Setting } from '@element-plus/icons-vue'

const tableRowClassName = ({
    row,
    rowIndex,
}) => {
    if (row.type === 'dir') {
        return 'warning-row'
    } else {
        return 'success-row'
    }
}

const currentDir = ref("")
const xrdHostName = ref("")
onMounted(() => {
    getHomeDirPath().then(resp => {
        let homeDir = resp.data.data
        currentDir.value = homeDir

        listDir()
        getXrdHostName()
    })
})

const tableData = ref([])

const changeDir = (index: number) => {
    console.log("changeDir index: " + index);

    currentDir.value = currentDir.value.split("/").filter((k, i) => {
        console.log(i);

        return i <= index
    }).join('/')

    listDir()
}

const selectDir = (row: { type: string; name: string; }) => {
    if (row.type == 'dir') {
        currentDir.value = currentDir.value + "/" + row.name;
        listDir();
    } else {
        const srcFileToDownload = currentDir.value + "/" + row.name
        // @ts-ignore: TS2304: cannot find name ' require'
        // The auto import is used
        ElMessageBox.confirm('Do you want to download the file?', srcFileToDownload, {
            // if you want to disable its autofocus
            // autofocus: false,
            confirmButtonText: 'Download',
            cancelButtonText: 'Cancel',
        }).then(() => {
            // @ts-ignore: TS2304: cannot find name ' require'
            // The auto import is used
            ElMessage({
                type: 'success',
                message: 'Preparing to download: ' + srcFileToDownload,
            })

            // Requesting to stage the file
            var destFileToDownload = "";

            getFileStagedForDownload(srcFileToDownload).then(resp => {
                destFileToDownload = resp.data.data

                // Force download a file 
                axios.get(destFileToDownload, { responseType: 'blob' })
                    .then(response => {
                        saveAs(response.data, row.name);
                    }).catch((response) => {
                        console.error("Could not Download the requested file from the backend.", response);
                    });
            })
        }).catch(() => {
            console.log("Download is canceled by the user.");
        });
    }
}

const listDir = () => {
    console.log(currentDir.value);

    getItemsInDir(currentDir.value).then(resp => {
        tableData.value = resp.data.data
    })
}

const getXrdHostName = () => {
    getHostName().then(resp => { xrdHostName.value = resp.data.data })
}
</script>


<style>
.el-table .warning-row {
    --el-table-tr-bg-color: var(--el-color-warning-light-9);
}

.el-table .success-row {
    --el-table-tr-bg-color: var(--el-color-success-light-9);
}


.clickable {
    cursor: pointer;
    text-decoration: none;
}

.clickable:hover {
    text-decoration: underline;
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

.layout-file-tree-container .el-header {
    position: sticky;
    /* background-color: var(--el-color-primary-light-7);*/
    color: var(--el-text-color-primary);
    text-align: center;
}

.layout-file-tree-container .el-aside {
    color: var(--el-text-color-primary);
    /*background: var(--el-color-primary-light-8);*/
}

.layout-file-tree-container .el-menu {
    border-right: none;
}

.layout-file-tree-container .el-main {
    padding-right: 20px;
    padding-bottom: 20px;
}

.layout-file-tree-container .toolbar {
    display: flex;
    height: 100%;
    /* align-items: center;
    justify-content: center;
     
    right: 20px;*/
}

.full-size-row {
    width: 100%;
    height: 100%;
}

.toolbar-right-content {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    justify-content: center;
    height: 100%;
}

.toolbar-left-content {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    justify-content: center;
    height: 100%;
}

.el-breadcrumb {
    font-size: 18px;
}
</style>