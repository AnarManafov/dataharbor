<template>
    <div class="browse-view">
        <el-container class="file-browser-container">
            <el-header class="toolbar-header">
                <Toolbar :serviceStatusTooltip="serviceStatusTooltip" :serviceStatusColor="serviceStatusColor"
                    :xrdHostName="xrdHostName" :currentDirectory="currentDirectory" :initialPath="initialPath"
                    :folderCount="folderCount" :fileCount="fileCount" :totalOnPageFileSize="totalOnPageFileSize"
                    :totalFolderCount="totalFolderCount" :totalFileCount="totalFileCount"
                    :totalFileSize="cumulativeFileSize" @changeDirToInitialPath="changeDirToInitialPath"
                    @changeDir="changeDir" />
            </el-header>
            <el-container>
                <el-header class="pagination-header">
                    <el-pagination background layout="prev, pager, next, jumper" size="small" :page-size="pageSize"
                        :total="totalItems" :current-page="currentPage" @current-change="handlePageChange" />
                </el-header>
                <el-main class="file-table-main">
                    <el-scrollbar>
                        <FileTable :tableLoading="tableLoading" :filteredData="filteredData" :filters="filters"
                            @selectDir="selectDir" />
                        <!-- Show a button to back to top -->
                        <el-backtop :right="40" :bottom="40" />
                    </el-scrollbar>
                </el-main>
                <el-footer class="pagination-footer">
                    <el-pagination background layout="prev, pager, next, jumper" size="small" :page-size="pageSize"
                        :total="totalItems" :current-page="currentPage" @current-change="handlePageChange" />
                </el-footer>
            </el-container>
        </el-container>
    </div>
</template>

<script lang="ts" setup>
import { getHostName, getInitialDirPath, getItemsInDir, getFileStagedForDownload, getBackendHealth, getPagedItemsInDir } from '@/api/api';
import { onMounted, onBeforeUnmount, ref, watch, getCurrentInstance, computed } from 'vue';
import { useRouter, onBeforeRouteUpdate } from 'vue-router';
import { saveAs } from 'file-saver';
import axios from 'axios';
import { useStorage } from '@vueuse/core'
import { displayErrorMessage, joinPaths } from '@/utils/utils';
import Toolbar from '../components/partials/BrowserXrdToolbar.vue';
import FileTable from '../components/partials/BrowserXrdFileTable.vue';

// Define props
const props = defineProps({
    path: {
        type: String,
        required: false,
        default: ''
    }
});

const router = useRouter();
const { appContext } = getCurrentInstance();
const app_colors = appContext.config.globalProperties.$app_colors;
const filters = appContext.config.globalProperties.$filters;

const initialPath = ref("");
const xrdHostName = ref("")
const isBackendOnline = ref(false);

const PAGE_SIZE = 500;

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

// Origin table data received from backend API
const tableData = ref([]);
// Define currentPage as a reactive property
const currentPage = ref(1); // default page
// Define pageSize as a reactive property
const pageSize = ref(10); // default page size
// Define totalPages as a reactive property
const totalPages = ref(0);
// Define total number of items
const totalItems = ref(0);
// Computed property of the table data.
// A filter or other modifiers can be added to change the data representation for the user
const filteredData = computed(() => {
    return tableData.value/*.filter(item => 
        item.name.toLowerCase().includes(searchQuery.value.toLowerCase())
      );*/
});

// Define the table loading state
const tableLoading = ref(false);

// Computed properties to count folders and files
const folderCount = computed(() => {
    return filteredData.value.filter(item => item.type === 'dir').length;
});
const fileCount = computed(() => {
    return filteredData.value.filter(item => item.type !== 'dir').length;
});
// Computed property to calculate the total size of the files
const totalOnPageFileSize = computed(() => {
    const totalSize = filteredData.value
        .filter(item => item.type !== 'dir')
        .reduce((acc, item) => acc + item.size, 0);
    return filters.prettyBytes(totalSize);
});

// Declare totalFileCount variable
const totalFileCount = ref(0);
// Declare totalFolderCount variable
const totalFolderCount = ref(0);
// Declare cumulativeFileSize variable
const cumulativeFileSize = ref("");

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
    // Show the loading spinner
    tableLoading.value = true;
    // Clear the table while loading to avoid showing the old data and a big empty table
    tableData.value = [];
    try {
        await listDir();
    } catch (error) {
        displayErrorMessage(error);
    } finally {
        // Hide the loading spinner
        tableLoading.value = false;
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
        const resp = await getItemsInDir(currentDirectory.value, PAGE_SIZE);
        if (resp.data.items != null) {
            tableData.value = resp.data.items;
            pageSize.value = resp.data.pageSize; // Update pageSize
            totalPages.value = resp.data.totalPages; // Update totalPages
            totalItems.value = resp.data.totalItems; // Update totalItems
            totalFileCount.value = resp.data.totalFileCount; // Update totalFileCount
            totalFolderCount.value = resp.data.totalFolderCount; // Update totalFolderCount
            cumulativeFileSize.value = filters.prettyBytes(resp.data.cumulativeFileSize); // Update cumulativeFileSize
            console.log("pageSize: %d, totalPages: %d, totalItems: %d, totalFileCount: %d, totalFolderCount: %d, cumulativeFileSize: %d", pageSize.value, totalPages.value, totalItems.value, totalFileCount.value, totalFolderCount.value, cumulativeFileSize.value);
        } else {
            // Revert the current directory value if the directory change fails
            currentDirectory.value = oldCurrentDirectory;
            // Empty the table data if the response is null and no errors
            tableData.value = [];
            if (resp.data.code != 200 && resp.data.error != "") {
                throw new Error(resp.data.error);
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
 * Function to handle page change event.
 * 
 * @param {number} page - The new page number.
 * @returns {void}
 */
const handlePageChange = async (page: number) => {
    console.log('handlePageChange: %d', page);
    try {
        const resp = await getPagedItemsInDir(currentDirectory.value, page, PAGE_SIZE);
        if (resp.data.items != null) {
            tableData.value = resp.data.items;
        } else {
            // Empty the table data if the response is null and no errors
            tableData.value = [];
            if (resp.data.code != 200 && resp.data.error != "") {
                throw new Error(resp.data.error);
            }
        }
    } catch (error) {
        // Check the backend health
        await fetchBackendServiceStatus();
        // Return an error
        throw new Error(`Error received from the backend while listing: ${currentDirectory.value}<br>${error.message}`);
    }
    currentPage.value = page;

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
.browse-view {
    height: 100vh;
    display: flex;
    flex-direction: column;
}

.file-browser-container {
    height: 100%;
    display: flex;
    flex-direction: column;
}

.file-browser-container .el-header {
    position: sticky;
    top: 0;
    z-index: 10;
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-light);
    color: var(--el-text-color-primary);
    text-align: center;
    height: auto;
    padding: 16px 24px;
}

.file-table-main {
    flex: 1;
    padding: 16px 24px;
    background: var(--el-bg-color-page);
    overflow: hidden;
}

.toolbar-header {
    /* Ensure the toolbar is above other elements */
    z-index: 12;
}

.pagination-header,
.pagination-footer {
    z-index: 11;
    display: flex;
    justify-content: center;
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-light);
    padding: 12px 24px;
    height: auto;
}

.pagination-header {
    border-top: 1px solid var(--el-border-color-light);
}

.pagination-footer {
    border-top: 1px solid var(--el-border-color-light);
    border-bottom: none;
}

/* Responsive adjustments */
@media (max-width: 768px) {

    .file-browser-container .el-header,
    .file-table-main,
    .pagination-header,
    .pagination-footer {
        padding: 12px 16px;
    }
}
</style>
