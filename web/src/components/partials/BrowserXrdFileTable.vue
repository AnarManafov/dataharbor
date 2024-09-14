<template>
    <el-table v-loading='tableLoading' :data='filteredData' :default-sort='{ prop: "name", order: "ascending" }' border>
        <el-table-column prop='name' label='Name' sortable>
            <template #default='scope'>
                <div style='display: flex; align-items: center'>
                    <el-icon :size='20' color='#409EFF' v-if='scope.row.type === "dir"'>
                        <Folder />
                    </el-icon>
                    <el-icon :size='20' color='#67C23A' v-else>
                        <Document />
                    </el-icon>
                    <span class='clickable' style='margin-left: 10px'
                        :style='{ fontWeight: scope.row.type === "dir" ? "bold" : "normal" }'
                        @click='() => selectDir(scope.row)'>{{ scope.row.name }}</span>
                </div>
            </template>
        </el-table-column>
        <el-table-column prop='size' label='Size' sortable width='150'>
            <template #default='scope'>
                {{ filters.prettyBytes(scope.row.size) }}
            </template>
        </el-table-column>
        <el-table-column prop='date_time' label='Date' sortable width='200' />
        <el-table-column prop='type' label='Type' sortable width='80'>
            <template #default='scope'>
                <el-tag :type='scope.row.type === "dir" ? "primary" : "success"' disable-transitions>
                    {{ scope.row.type }}
                </el-tag>
            </template>
        </el-table-column>
    </el-table>
</template>

<script lang="ts" setup>
import { Folder, Document } from '@element-plus/icons-vue';

const props = defineProps({
    filteredData: Array,
    filters: Object,
    tableLoading: Boolean
});

const emit = defineEmits(['selectDir']);

const selectDir = (row: { type: string; name: string }) => {
    emit('selectDir', row);
};
</script>

<style scoped>
.clickable {
    cursor: pointer;
    text-decoration: none;
}

.clickable:hover {
    text-decoration: underline;
}
</style>