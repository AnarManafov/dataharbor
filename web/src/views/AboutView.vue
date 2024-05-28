<template>
  <div>
    <el-card style="max-width: 480px">
      <template #header>
        <div class="card-header">
          <el-breadcrumb separator="/">
            <el-breadcrumb-item @click="changeDir(k)" v-for="(i, k) in currentDir.split('/')" :key="k">{{ i
              }}</el-breadcrumb-item>
          </el-breadcrumb>
        </div>
      </template>
      <el-table @row-dblclick="selectDir" :data="tableData" :row-class-name="tableRowClassName"
        style="width: 100%;height: 600px; overflow-y: auto;">
        <el-table-column prop="name" label="Name" width="180" />
        <el-table-column prop="type" label="Type" width="180" />
      </el-table>
      <template #footer></template>
    </el-card>


  </div>
</template>

<script lang="ts" setup>
import { getHomeDirPath, getItemsInDir } from '@/api/api';
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
onMounted(() => {
  getHomeDirPath().then(resp => {
    let homeDir = resp.data.data
    currentDir.value = homeDir

    listDir()
  })
})

const tableData = ref([])

const changeDir = (index) => {
  console.log(index);

  currentDir.value = currentDir.value.split("/").filter((k, i) => {
    console.log(i);

    return i <= index
  }).join('/')


  listDir()
}

const selectDir = (row) => {
  console.log(row);
  if (row.type == 'dir') {
    currentDir.value = currentDir.value + "/" + row.name
    listDir()
  } else {
    ElMessageBox.alert('This is a mock a open a file', 'FileName: ' + row.name, {
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
</script>

<style>
.el-table .warning-row {
  --el-table-tr-bg-color: var(--el-color-warning-light-9);
}

.el-table .success-row {
  --el-table-tr-bg-color: var(--el-color-success-light-9);
}
</style>