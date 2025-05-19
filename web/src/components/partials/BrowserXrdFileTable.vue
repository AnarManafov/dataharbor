<template>
    <el-table v-loading='tableLoading' :data='sortedData' :default-sort='{ prop: "name", order: "ascending" }'
        @sort-change="handleSortChange" border>
        <el-table-column prop='name' label='Name' sortable="custom">
            <template #default='scope'>
                <div style='display: flex; align-items: center'>
                    <el-icon :size='14' color='#409EFF' v-if='scope.row.type === "dir" && scope.row.name === ".."'>
                        <ArrowUp />
                    </el-icon>
                    <el-icon :size='14' color='#409EFF' v-else-if='scope.row.type === "dir"'>
                        <Folder />
                    </el-icon>
                    <el-icon :size='14' color='#67C23A' v-else>
                        <Document />
                    </el-icon>
                    <span class='clickable' style='margin-left: 10px'
                        :style='{ fontWeight: scope.row.type === "dir" ? "bold" : "normal" }'
                        @click='() => selectDir(scope.row)'>{{ scope.row.name }}</span>
                </div>
            </template>
        </el-table-column>
        <el-table-column prop='size' label='Size' sortable="custom" width='150'>
            <template #default='scope'>
                <span v-if='scope.row.name === ".."'>-</span>
                <span v-else>{{ filters.prettyBytes(scope.row.size) }}</span>
            </template>
        </el-table-column>
        <el-table-column prop='date_time' label='Date' sortable="custom" width='200' />
        <el-table-column prop='type' label='Type' sortable="custom" width='80'>
            <template #default='scope'>
                <el-tag :type='scope.row.type === "dir" ? "primary" : "success"' disable-transitions>
                    {{ scope.row.type }}
                </el-tag>
            </template>
        </el-table-column>
        <el-table-column label='Actions' width='100' fixed="right" class-name="actions-column">
            <template #default='scope'>
                <div class="actions-wrapper">
                    <el-button v-if='scope.row.type === "file" && scope.row.name !== ".."'
                        @click='downloadFile(scope.row)' size='small' type='primary' :icon='Download'
                        :loading='scope.row.downloading' :disabled='scope.row.downloading' circle>
                    </el-button>
                </div>
            </template>
        </el-table-column>
    </el-table>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue';
import { Folder, Document, ArrowUp, Download } from '@element-plus/icons-vue';

const props = defineProps({
    filteredData: {
        type: Array,
        required: true,
        default: () => []
    },
    filters: {
        type: Object,
        required: true
    },
    tableLoading: {
        type: Boolean,
        default: false
    }
});

const emit = defineEmits(['selectDir', 'downloadFile']);

// Sorting state
const sortProp = ref('name');
const sortOrder = ref('ascending');

// Handle download action
const downloadFile = (row: any) => {
    emit('downloadFile', row);
};

// Computed property for sorted data that keeps ".." at the top
const sortedData = computed(() => {
    let data = [...props.filteredData] as any[];

    // Separate the ".." folder from other items
    const parentDirIndex = data.findIndex((item: any) => item.name === '..');
    let parentDir = null;

    if (parentDirIndex !== -1) {
        parentDir = data.splice(parentDirIndex, 1)[0];
    }

    // Sort the remaining data
    if (sortProp.value && data.length > 0) {
        data.sort((a: any, b: any) => {
            let aVal, bVal;

            switch (sortProp.value) {
                case 'name':
                    aVal = a.name.toLowerCase();
                    bVal = b.name.toLowerCase();
                    break;
                case 'size':
                    aVal = a.size || 0;
                    bVal = b.size || 0;
                    break;
                case 'date_time':
                    aVal = new Date(a.date_time || a.dateTime || 0);
                    bVal = new Date(b.date_time || b.dateTime || 0);
                    break;
                case 'type':
                    aVal = a.type;
                    bVal = b.type;
                    break;
                default:
                    return 0;
            }

            let result = 0;
            if (aVal < bVal) result = -1;
            else if (aVal > bVal) result = 1;

            return sortOrder.value === 'ascending' ? result : -result;
        });
    }

    // Add the ".." folder back at the top if it exists
    if (parentDir) {
        data.unshift(parentDir);
    }

    return data;
});

// Handle sort change
const handleSortChange = ({ prop, order }: { prop: string; order: string }) => {
    sortProp.value = prop;
    sortOrder.value = order;
};

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

/* Actions column styling */
.actions-wrapper {
    display: flex;
    justify-content: center;
    align-items: center;
    padding-right: 8px;
    /* Add right padding to prevent scrollbar overlap */
}

/* Ensure actions column has proper spacing */
:deep(.actions-column) {
    padding-right: 16px !important;
    /* Extra padding for the column itself */
}

/* Responsive adjustments for smaller screens */
@media (max-width: 768px) {
    .actions-wrapper {
        padding-right: 4px;
    }

    :deep(.actions-column) {
        padding-right: 8px !important;
    }
}
</style>