<template>
    <div>
        <el-card style="max-width: 1000px">
            <template #header>
                <div class="card-header">
                    <el-breadcrumb separator="/">
                        <el-breadcrumb-item>{{ xrdHostName }}</el-breadcrumb-item>
                        <template v-for="(i, k) in currentDir.split('/')" :key="k">
                            <el-breadcrumb-item @click="changeDir(k)" v-if="i.length > 0">{{ i }}</el-breadcrumb-item>
                        </template>
                    </el-breadcrumb>
                </div>
            </template>
            <el-table @row-dblclick="selectDir" :data="tableData" :row-class-name="tableRowClassName"
                style="width: 100%;height: 800px; overflow-y: auto;"
                :default-sort="{ prop: 'name', order: 'ascending' }">
                <el-table-column prop="name" label="Name" sortable width="400" />
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
// FIXME: if this import is used, then ElMessageBox don't show up.
//import { ElMessageBox, ElMessage } from 'element-plus';

const tableRowClassName = ({
    row,
    rowIndex,
}) => {
    if (row.type === 'dir') {
        return 'warning-row'
    } else {
        return 'success-row'
    }
    return ''
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
        ElMessageBox.confirm('Do you want to download the file?', srcFileToDownload, {
            // if you want to disable its autofocus
            // autofocus: false,
            confirmButtonText: 'Download',
            cancelButtonText: 'Cancel',
        }).then(() => {
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
</style>