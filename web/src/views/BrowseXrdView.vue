<template>
    <div>
        <el-card style="max-width: 1000px">
            <template #header>
                <div class="card-header">
                    <el-breadcrumb separator="/">
                        <el-breadcrumb-item>{{ xrdHostName }}</el-breadcrumb-item>
                        <template v-for="(i, k) in currentDir.split('/')" :key="k">
                            <el-breadcrumb-item @click="changeDir(k)" v-if="i.length > 1">{{
                                i
                            }}</el-breadcrumb-item>
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
import { getHostName, getHomeDirPath, getItemsInDir } from '@/api/api';
import { onMounted, ref } from 'vue';

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

const changeDir = (index) => {
    console.log("changeDir index: " + index);

    currentDir.value = currentDir.value.split("/").filter((k, i) => {
        console.log(i);

        return i <= index
    }).join('/')

    listDir()
}

const selectDir = (row) => {
    if (row.type == 'dir') {
        currentDir.value = currentDir.value + "/" + row.name
        listDir()
    } else {
        ElMessageBox.alert('This is a mockup to download a file', 'FileName: ' + row.name, {
            // if you want to disable its autofocus
            // autofocus: false,
            confirmButtonText: 'OK',
            callback: (action: Action) => {
                ElMessage({
                    type: 'info',
                    message: `action: ${action}`,
                })
            },
        })
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