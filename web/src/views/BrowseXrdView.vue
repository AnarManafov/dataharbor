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
                                    <Setting />
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
                        <el-row class="full-size-row">
                            <el-col :span="12" class="toolbar-left-content">

                                <div>
                                    <el-tooltip class="box-item" effect="dark" :content="getServiceStatusTooltip()"
                                        placement="bottom-start">
                                        <el-icon :style="{ color: getServiceStatusColor() }"
                                            @click="currentDir = initialPath; listDir()" :size="18"
                                            style="margin-right: 5px; margin-top: 3px">
                                            <HomeFilled />
                                        </el-icon>
                                    </el-tooltip>
                                </div>
                                <div>
                                    <el-breadcrumb separator="/">
                                        <el-breadcrumb-item @click="currentDir = initialPath; listDir()"><a
                                                href="#">Initial
                                                Directory</a></el-breadcrumb-item>
                                        <template
                                            v-for="(item, index) in currentDir.replace(initialPath, '').split('/')"
                                            :key="index">
                                            <el-breadcrumb-item @click="changeDir(index)" v-if="item.length > 0">
                                                <a href="#">{{ item }}</a>
                                            </el-breadcrumb-item>
                                        </template>
                                    </el-breadcrumb>
                                </div>

                            </el-col>
                            <el-col :span="12" class="toolbar-right-content">
                                <div style="font-size: 12px;">
                                    Data Server Host: <span style="font-weight: bold;">{{ xrdHostName
                                        }}</span>
                                </div>
                            </el-col>
                        </el-row>
                    </div>
                </el-header>
                <el-container>
                    <el-main>
                        <el-scrollbar>
                            <el-table :data="tableData" :default-sort="{ prop: 'name', order: 'ascending' }" border>
                                <el-table-column prop="name" label="Name" sortable>
                                    <template #default="scope">
                                        <div style="display: flex; align-items: center">
                                            <el-icon :size="20" color="#409EFF" v-if="scope.row.type === 'dir'">
                                                <Folder />
                                            </el-icon>
                                            <el-icon :size="20" color="#67C23A" v-else>
                                                <Document />
                                            </el-icon>
                                            <span class="clickable" style="margin-left: 10px"
                                                :style="{ fontWeight: scope.row.type === 'dir' ? 'bold' : 'normal' }"
                                                @click="selectDir(scope.row)">{{ scope.row.name
                                                }}</span>
                                        </div>
                                    </template>
                                </el-table-column>
                                <el-table-column prop="size" label="Size" sortable width="150" />
                                <el-table-column prop="date_time" label="Date" sortable width="200" />
                                <el-table-column prop="type" label="Type" sortable width="80">
                                    <template #default="scope">
                                        <el-tag :type="scope.row.type === 'dir' ? 'primary' : 'success'"
                                            disable-transitions>{{
                                                scope.row.type }}</el-tag>
                                    </template>
                                </el-table-column>
                            </el-table>
                        </el-scrollbar>
                    </el-main>
                </el-container>
            </el-container>
        </el-container>
    </el-container>
</template>


<script lang="ts" setup>
import { getHostName, getInitialDirPath, getItemsInDir, getFileStagedForDownload, getBackendHealth } from '@/api/api';
import { onMounted, onBeforeUnmount, ref, watch } from 'vue';
import { saveAs } from 'file-saver';
import axios from 'axios';
import { Folder, Document, Menu as IconMenu, Setting, HomeFilled } from '@element-plus/icons-vue'
import { useStorage } from '@vueuse/core'


const initialPath = useStorage('initialDirectoryPath', '', sessionStorage)
const currentDir = useStorage('currentDirectory', '', sessionStorage)
const xrdHostName = ref("")
const isBackendOnline = ref(false);

let interval: number | undefined;

watch(isBackendOnline, (newValue, oldValue) => {
    if (!newValue) {
        ElMessage.error('Backend service is offline.')
        tableData.value = [];
    }
    else {
        ElMessage({
            message: 'Backend service is online.',
            type: 'success',
        })
        listDir()
    }
});

const fetchBackendServiceStatus = (): void => {
    getBackendHealth()
        .then(resp => {
            isBackendOnline.value = (resp.data.data == 'ok') ? true : false;
        })
        .catch(() => {
            isBackendOnline.value = false
        });
};

// Return the color based on the backend service status
const getServiceStatusColor = () => {
    if (isBackendOnline.value) {
        return '#67C23A' // online - green
    } else {
        return '#C23A3A' // offline - red
    }
}

// Return the tooltip text based on the backend service status
const getServiceStatusTooltip = () => {
    if (isBackendOnline.value) {
        return 'Backend service is ONLINE' // online
    } else {
        return 'Backend service is OFFLINE' // offline
    }
}

onMounted(() => {
    // Send a health check every 30 seconds
    interval = window.setInterval(fetchBackendServiceStatus, 30000);
    // Make the first call immediately
    fetchBackendServiceStatus()

    getInitialDirPath()
        .then(resp => {
            let homeDir = resp.data.data
            // Use new data only if there no value in the storage
            if (!currentDir.value) currentDir.value = homeDir
            if (!initialPath.value) initialPath.value = homeDir

            listDir()
            getXrdHostName()
        })
        .catch((error) => {
            if (error) {
                ElMessage.error(error.message)
                console.log(error);
                // Force check the backend health
                fetchBackendServiceStatus()
            }
        })
})

onBeforeUnmount(() => {
    if (interval) {
        clearInterval(interval);
    }
});

const tableData = ref([])

const changeDir = (index: number) => {
    // Add initial path, as it's subtracted when populating the data in breadcrumb.
    let initialIndex = initialPath.value.split("/").length - 1;
    index += initialIndex;

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
        })
            .then(() => {
                // @ts-ignore: TS2304: cannot find name ' require'
                // The auto import is used
                ElMessage({
                    type: 'success',
                    message: 'Preparing to download: ' + srcFileToDownload,
                })

                // Requesting to stage the file
                var destFileToDownload = "";

                getFileStagedForDownload(srcFileToDownload)
                    .then(resp => {
                        destFileToDownload = resp.data.data

                        // Force download a file 
                        axios.get(destFileToDownload, { responseType: 'blob' })
                            .then(response => {
                                saveAs(response.data, row.name);
                            })
                            .catch((response) => {
                                console.error("Could not Download the requested file from the backend.", response);
                            });
                    })
                    .catch((error) => {
                        if (error) {
                            ElMessage.error(error.message)
                            console.log(error);
                            // Force check the backend health
                            fetchBackendServiceStatus()
                        }
                    })
            })
            .catch(() => {
                console.log("Download is canceled by the user.");
            });
    }
}


const listDir = () => {
    console.log(currentDir.value);

    getItemsInDir(currentDir.value)
        .then(resp => {
            if (resp.data.data != null) {
                tableData.value = resp.data.data
            }
            else {
                ElMessage.error("Could not get a list of files for this directory.")
                console.log("Could not get a list of files for this directory.");
                ElMessage.error(resp.data.msg)
                console.log(resp.data.msg);
            }
        })
        .catch((error) => {
            if (error) {
                ElMessage.error(error.message)
                console.log(error);
                // Force check the backend health
                fetchBackendServiceStatus()
            }
        })
}

const getXrdHostName = () => {
    getHostName()
        .then(resp => { xrdHostName.value = resp.data.data })
        .catch((error) => {
            if (error) {
                ElMessage.error(error.message)
                console.log(error);
                // Force check the backend health
                fetchBackendServiceStatus()
            }
        })
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

i.el-icon-folder {
    color: blue
}
</style>