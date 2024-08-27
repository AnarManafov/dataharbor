<template>
    <div>
        <el-card style="max-width: 1000px">
            <template #header>
                <div class="card-header">
                    <el-breadcrumb separator="/">
                        <el-breadcrumb-item>{{ xrdHostName }}</el-breadcrumb-item>
                        <template v-for="(item, index) in currentDir.split('/')" :key="index">
                            <el-breadcrumb-item @click="changeDir(index)" v-if="item.length > 0">{{ item
                                }}</el-breadcrumb-item>
                        </template>
                    </el-breadcrumb>
                </div>
            </template>
            <el-table :data="tableData" :row-class-name="tableRowClassName"
                style="width: 100%;height: 800px; overflow-y: auto;"
                :default-sort="{ prop: 'name', order: 'ascending' }">
                <el-table-column prop="name" label="Name" sortable width="400">
                    <template #default="scope">
                        <span class="clickable" @click="selectDir(scope.row)">{{ scope.row.name }}</span>
                    </template>
                </el-table-column>
                <el-table-column prop="size" label="Size" sortable width="150" />
                <el-table-column prop="date_time" label="Date" sortable width="200" />
                <el-table-column prop="type" label="Type" sortable width="80" />
            </el-table>
            <template #footer></template>
        </el-card>
    </div>
</template>


<script lang="ts" setup>
import { getHostName, getHomeDirPath, getItemsInDir, getFileStagedForDownload } from '@/api/api';
import { onMounted, ref } from 'vue';
import { saveAs } from 'file-saver';
import axios from 'axios';

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

.table-wrapper {
    width: 0;
    flex: 1 1 auto;
}

.clickable {
    cursor: pointer;
    text-decoration: none;
}

.clickable:hover {
    text-decoration: underline;
}
</style>