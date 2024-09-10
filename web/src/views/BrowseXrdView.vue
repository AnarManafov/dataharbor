<template>
    <div>
        <el-divider />
    </div>
    <el-container class='layout-file-tree-container'>
        <el-container>
            <el-aside width='200px'>
                <el-scrollbar>
                    <el-menu :default-opened="['2']" default-active='2'>
                        <el-sub-menu index='1'>
                            <template #title>
                                <el-icon>
                                    <IconMenu />
                                </el-icon>Navigator One
                            </template>
                            <el-menu-item-group>
                                <template #title>Group 1</template>
                                <el-menu-item index='1-1'>Option 1</el-menu-item>
                                <el-menu-item index='1-2'>Option 2</el-menu-item>
                            </el-menu-item-group>
                            <el-menu-item-group title='Group 2'>
                                <el-menu-item index='1-3'>Option 3</el-menu-item>
                            </el-menu-item-group>
                            <el-sub-menu index='1-4'>
                                <template #title>Option4</template>
                                <el-menu-item index='1-4-1'>Option 4-1</el-menu-item>
                            </el-sub-menu>
                        </el-sub-menu>
                        <el-sub-menu index='2'>
                            <template #title>
                                <el-icon>
                                    <Setting />
                                </el-icon>Settings
                            </template>
                            <el-menu-item-group>
                                <template #title>Group 1</template>
                                <el-menu-item index='2-1'>Option 1</el-menu-item>
                                <el-menu-item index='2-2'>Option 2</el-menu-item>
                            </el-menu-item-group>
                            <el-menu-item-group title='Group 2'>
                                <el-menu-item index='2-3'>Option 3</el-menu-item>
                            </el-menu-item-group>
                            <el-sub-menu index='2-4'>
                                <template #title>Option 4</template>
                                <el-menu-item index='2-4-1'>Option 4-1</el-menu-item>
                            </el-sub-menu>
                        </el-sub-menu>
                    </el-menu>
                </el-scrollbar>
            </el-aside>
            <el-container>
                <el-header>
                    <div class='toolbar'>
                        <el-row class='full-size-row'>
                            <el-col :span='12' class='toolbar-left-content'>

                                <div>
                                    <el-tooltip class='box-item' effect='dark' :content='serviceStatusTooltip'
                                        placement='bottom-start'>
                                        <el-icon :style='{ color: serviceStatusColor }'
                                            @click='() => { changeDirToInitialPath() }' :size='18'
                                            style='margin-right: 5px; margin-top: 3px'>
                                            <HomeFilled />
                                        </el-icon>
                                    </el-tooltip>
                                </div>
                                <div>
                                    <el-breadcrumb separator='/'>
                                        <el-breadcrumb-item @click='() => { changeDirToInitialPath() }'><a>Initial
                                                Directory</a></el-breadcrumb-item>
                                        <template
                                            v-for="(item, index) in currentDirectory.replace(initialPath, '').split('/')"
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
                                    Data Server Host: <span style='font-weight: bold;'>{{ xrdHostName
                                        }}</span>
                                </div>
                            </el-col>
                        </el-row>
                    </div>
                </el-header>
                <el-container>
                    <el-main>
                        <el-scrollbar>
                            <el-table :data='filteredData' :default-sort='{ prop: "name", order: "ascending" }' border>
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
                                                @click='() => selectDir(scope.row)'>{{ scope.row.name
                                                }}</span>
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
                                        <el-tag :type='scope.row.type === "dir" ? "primary" : "success"'
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
import { onMounted, onBeforeUnmount, ref, watch, getCurrentInstance, computed } from 'vue';
import { useRouter, useRoute, onBeforeRouteUpdate } from 'vue-router';
import { saveAs } from 'file-saver';
import axios from 'axios';
import { Folder, Document, Menu as IconMenu, Setting, HomeFilled } from '@element-plus/icons-vue'
import { useStorage } from '@vueuse/core'
import { displayErrorMessage, joinPaths } from '@/utils/utils';

// Define props
const props = defineProps({
    path: {
        type: String,
        required: false,
        default: ''
    }
});

const router = useRouter();
const route = useRoute();
const { appContext } = getCurrentInstance();
const app_colors = appContext.config.globalProperties.$app_colors;
const filters = appContext.config.globalProperties.$filters;

const initialPath = ref("");
const xrdHostName = ref("")
const isBackendOnline = ref(false);

// The current directory path. A ref property
const currentDirectory = ref(useStorage('currentDirectory', '', sessionStorage));

// computed ref
// The tooltip text based on the backend service status
const serviceStatusTooltip = computed(() => {
    return isBackendOnline.value ? 'Backend service is ONLINE' : 'Backend service is OFFLINE'
})
//// the color of the service status
const serviceStatusColor = computed(() => {
    return isBackendOnline.value ? app_colors.online : app_colors.offline
})


let interval: number | undefined;
/**
 * Watcher to check the backend service status.
 * 
 * @param {boolean} newValue - The new value of the backend service status.
 * @returns {void}
 */
watch(isBackendOnline, async (newValue) => {
    if (!newValue) {
        displayErrorMessage(new Error('Backend service is offline.'))
        // Clear the table data
        tableData.value = [];
    }
    else {
        ElMessage({
            message: 'Backend service is online.',
            type: 'success',
        })
        // Push the new path to the router
        loadDirectory(currentDirectory.value);
    }
});

/**
 * Function to fetch the backend service status.
 * 
 * @returns {void}
 */
const fetchBackendServiceStatus = (): void => {
    getBackendHealth()
        .then(resp => {
            isBackendOnline.value = (resp.data.data == 'ok') ? true : false;
        })
        .catch(() => {
            isBackendOnline.value = false
        });
};

/**
 * This function is called when the component is mounted.
 * It performs the following tasks:
 * 1. Sends a health check to the backend service every 30 seconds.
 * 2. Makes an immediate call to fetch the backend service status.
 * 3. Retrieves the initial directory path and sets it as the current directory if there is no value in the storage.
 * 4. Sets the initial path value to the home directory.
 * 5. Retrieves the Xrd host name.
 * 6. Handles any errors that occur during the process, displaying an error message and forcing a backend service health check.
 */
onMounted(() => {
    // Get the initial directory path.
    // This needs to be done before any other browsing operation.
    getInitialDirPath()
        .then(resp => {
            let homeDir = resp.data.data
            // Use new data only if there no value in the storage
            if (!currentDirectory.value) currentDirectory.value = homeDir
            initialPath.value = homeDir

            getXrdHostName()
        })
        .catch((error) => {
            if (error) {
                displayErrorMessage(error)
                // Force check the backend health
                fetchBackendServiceStatus()
            }
        })

    // Send a health check every 30 seconds
    interval = window.setInterval(fetchBackendServiceStatus, 30000);
    // Make the first call immediately
    fetchBackendServiceStatus()
})

// Origin table data received from backend API
const tableData = ref([])
// Computed property of the table data.
// A filter or other modifiers can be added to change the data representation for the user
const filteredData = computed(() => {
    return tableData.value/*.filter(item => 
        item.name.toLowerCase().includes(searchQuery.value.toLowerCase())
      );*/
});

/**
 * Function to change the current directory to the initial path.
 * 
 * @returns {void}
 */
const changeDirToInitialPath = async () => {
    console.log('User forced to change the directory to the initial path: %s', initialPath.value);
    currentDirectory.value = initialPath.value;
    // Push the new path to the router
    routerPushNewPath(initialPath.value);
}

/**
 * Function to change the directory based on the breadcrumb.
 * 
 * @param {number} index - The index of the breadcrumb item.
 * @returns {void}
 */
const changeDir = async (index: number) => {
    // Add initial path, as it's subtracted when populating the data in breadcrumb.
    let initialIndex = initialPath.value.split("/").length - 1;
    index += initialIndex;

    console.log('changeDir index: %d', index);
    // Cache the current directory value before changing it
    let oldCurrentDirectory = currentDirectory.value;
    // Change the current directory value
    currentDirectory.value = currentDirectory.value.split("/").filter((_, i) => {
        console.log(i);
        return i <= index
    }).join('/')

    // Push the new path to the router
    routerPushNewPath(currentDirectory.value);
}

/**
 * Function to select a directory or download a file.
 * 
 * @param {Object} row - The row object that contains the file or directory information.
 * @returns {void}
 */
const selectDir = async (row: { type: string; name: string; }) => {
    console.log('selectDir row element: %s', row.name);
    if (row.type == 'dir') {

        // Change the current directory value
        currentDirectory.value = joinPaths(currentDirectory.value, row.name);
        console.log('selectDir: %s', currentDirectory.value);

        // Push the new path to the router
        routerPushNewPath(currentDirectory.value);
    } else {
        const srcFileToDownload = joinPaths(currentDirectory.value, row.name);
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
                                displayErrorMessage(new Error(`Could not Download the requested file: ${srcFileToDownload}<br>${response}`))
                            });
                    })
                    .catch((error) => {
                        if (error) {
                            displayErrorMessage(error)
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

/**
 * Function to get the XRootD hostname.
 * 
 * @async
 * @function getXrdHostName
 * @returns {Promise<void>} - A promise that resolves when the XRootD hostname is fetched successfully.
 */
const getXrdHostName = () => {
    getHostName()
        .then(resp => { xrdHostName.value = resp.data.data })
        .catch((error) => {
            if (error) {
                displayErrorMessage(new Error(`Error: ${error.message}<br>Please check the backend service status.`))
                // Force check the backend health
                fetchBackendServiceStatus()
            }
        })
}

/**
 * Executes before the route is updated.
 * Loads the directory based on the route parameters.
 * @param {Object} to - The new route that we are navigating to.
 * @param {Object} from - The previous route that we are navigating from.
 * @param {Function} next - The function to call to continue the navigation.
 */
onBeforeRouteUpdate((to, from, next) => {
    console.debug('onBeforeRouteUpdate to: %o, from: %o', to, from);
    let path = to.params.path || initialPath.value;
    if (Array.isArray(path)) {
        path = path.join('/');
    }

    loadDirectory(path);
    next();
});

/**
 * Loads a directory and updates the current directory value.
 * 
 * @param {string} path - The path of the directory to load. If not provided, the initial path value will be used.
 * @returns {Promise<void>} - A promise that resolves when the directory is loaded successfully.
 */
const loadDirectory = async (path) => {
    let pathTmp = path || initialPath.value;
    if (Array.isArray(pathTmp)) {
        path = pathTmp.join('/');
    }
    currentDirectory.value = path;
    try {
        await listDir();
    } catch (error) {
        displayErrorMessage(error);
    }
};

/**
 * Function to navigate to a new path using the router.
 * 
 * @param {string | string[]} _path - The new path to navigate to.
 * @returns {void}
 */
const routerPushNewPath = (_path) => {
    console.log('routerPushNewPath: %s', _path);
    let pathTmp = _path || initialPath.value;
    if (Array.isArray(pathTmp)) {
        _path = pathTmp.join('/');
    }

    router.push({ name: 'browse', params: { path: _path } }); // No need to encode the path
};

/**
 * Function to list the directory and update the table data.
 * It modifies the tableData ref property and also CurrentDirectory ref property.
 * 
 * @async
 * @function listDir
 * @throws {Error} Error received from the backend while listing: {currentDirectory value}<br>{error message}
 */
const listDir = async () => {
    console.log("Listing dir: " + currentDirectory.value);
    // Cache the current directory value before changing it
    let oldCurrentDirectory = currentDirectory.value;

    if (!isBackendOnline.value) {
        await fetchBackendServiceStatus();
        // No need to do anything if the backend is still offline
        if (!isBackendOnline.value) return;
    }

    try {
        const resp = await getItemsInDir(currentDirectory.value);
        if (resp.data.data != null) {
            tableData.value = resp.data.data;
        } else {
            // Revert the current directory value if the directory change fails
            currentDirectory.value = oldCurrentDirectory;
            // Empty the table data if the response is null and no errors
            tableData.value = [];
            if (resp.data.code != 200 && resp.data.msg != "") {
                throw new Error(resp.data.msg);
            }
        }
    } catch (error) {
        // Revert the current directory value if the directory change fails
        currentDirectory.value = oldCurrentDirectory;
        // Check the backend health
        await fetchBackendServiceStatus();
        // Return an error
        throw new Error(`Error received from the backend while listing: ${currentDirectory.value}<br>${error.message}`);
    }
};

/**
 * Called before the component is unmounted.
 * Clears the interval if it exists.
 */
onBeforeUnmount(() => {
    if (interval) {
        clearInterval(interval);
    }
});

</script>


<style scoped>
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