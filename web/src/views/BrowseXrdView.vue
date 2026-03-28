<template>
    <div class="browse-view">
        <div class="file-browser-container">
            <!-- Sticky header section that contains toolbar and pagination -->
            <div class="sticky-header-section">
                <div class="toolbar-header">
                    <Toolbar :serviceStatusTooltip="serviceStatusTooltip" :serviceStatusColor="serviceStatusColor"
                        :currentDirectory="currentDirectory" :initialPath="initialPath" :folderCount="folderCount"
                        :fileCount="fileCount" :totalOnPageFileSize="totalOnPageFileSize"
                        :totalFolderCount="totalFolderCount" :totalFileCount="totalFileCount"
                        :totalFileSize="cumulativeFileSize" :vfsStat="vfsStat"
                        @changeDirToInitialPath="changeDirToInitialPath" @changeDir="changeDir" />
                </div>
                <div class="pagination-header">
                    <el-pagination background layout="prev, pager, next, jumper" size="small" :page-size="pageSize"
                        :total="totalItems" :current-page="currentPage" @current-change="handlePageChange" />
                </div>
            </div>

            <!-- Scrollable content area -->
            <div class="file-table-container">
                <!-- Always show file table (it handles empty states with parent directory) -->
                <FileTable :tableLoading="tableLoading" :filteredData="filteredData" :filters="filters"
                    @selectDir="selectDir" @downloadFile="handleDownloadFile" />

                <!-- Show empty state overlay when no actual files exist, but still allow table to show parent dir -->
                <div v-if="!tableLoading && tableData.length === 0 && filteredData.length <= 1"
                    class="empty-state-overlay">
                    <el-empty description="This directory is empty" />
                </div>

                <!-- Show a button to back to top -->
                <el-backtop :right="40" :bottom="40" />
            </div>
        </div>
    </div>
</template>

<script lang="ts" setup>
import { getInitialDirPath, getItemsInDir, getStreamingDownloadUrl, getBackendHealth, getPagedItemsInDir, getVirtualFSStat, pingXrd } from '@/api/api';
import { onMounted, onBeforeUnmount, ref, watch, getCurrentInstance, computed } from 'vue';
import { useRouter, onBeforeRouteUpdate } from 'vue-router';
import { ElMessage, ElMessageBox } from 'element-plus';
import { Folder } from '@element-plus/icons-vue';
import axios from 'axios';
import { useStorage } from '@vueuse/core'
import { displayErrorMessage, joinPaths } from '@/utils/utils';
import Toolbar from '../components/partials/BrowserXrdToolbar.vue';
import FileTable from '../components/partials/BrowserXrdFileTable.vue';
import { DownloadService } from '@/services/downloadService';
import { useNetworkStats } from '@/composables/useNetworkStats';

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
const isBackendOnline = ref(false);
const vfsStat = ref(null);

const { recordDownloadSpeed, updateLatency, updateQueryTime, _isPinging } = useNetworkStats();

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
    let data = [...tableData.value];
    let parentDirItem = null;

    // Check if we should show the ".." folder (parent directory)
    const shouldShowParentDir = currentDirectory.value && initialPath.value &&
        currentDirectory.value !== initialPath.value &&
        currentDirectory.value.startsWith(initialPath.value);

    if (shouldShowParentDir) {
        parentDirItem = {
            name: '..',
            type: 'dir',
            size: 0,
            dateTime: '',
            date_time: '',
            isParentDir: true
        };
    }

    // Apply any filtering logic here (search, type filters, etc.)
    // ... existing filter logic can be added here ...

    // Apply sorting to the data (excluding the parent directory)
    // Note: The sorting logic would go here when it's implemented
    // For now, we'll just maintain the original order

    // Add the parent directory at the top if it should be shown
    if (parentDirItem) {
        data.unshift(parentDirItem);
    }

    return data;
});

// Define the table loading state
const tableLoading = ref(false);

// Computed properties to count folders and files
const folderCount = computed(() => {
    return filteredData.value.filter(item => item.type === 'dir' && item.name !== '..').length;
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
        // Only load directory if currentDirectory has been set
        // (avoid loading with empty path before onMounted sets the initial directory)
        if (currentDirectory.value) {
            loadDirectory(currentDirectory.value);
        }
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
 * Ping the XRD server to measure latency.
 * Runs periodically alongside health checks.
 */
const fetchXrdLatency = (): void => {
    if (_isPinging.value) return;
    _isPinging.value = true;
    pingXrd()
        .then(resp => {
            if (resp.data?.data?.latencyMs != null) {
                updateLatency(resp.data.data.latencyMs, resp.data.data.connectMs);
            }
        })
        .catch(() => {
            // Silently fail — latency will show as N/A
        })
        .finally(() => {
            _isPinging.value = false;
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
    // Listen for authorization errors from API interceptor
    const handleAuthDenied = (event) => {
        ElMessage({
            message: event.detail.message,
            type: 'error',
            duration: 10000,
            showClose: true,
            dangerouslyUseHTMLString: true
        });
    };

    window.addEventListener('auth:access-denied', handleAuthDenied);

    // Cleanup listener when component unmounts
    onBeforeUnmount(() => {
        window.removeEventListener('auth:access-denied', handleAuthDenied);
    });

    // Get the initial directory path.
    // This needs to be done before any other browsing operation.
    getInitialDirPath()
        .then(resp => {
            let homeDir = resp.data.data

            // Check if we have a path from props (route parameter)
            let targetPath = homeDir;
            if (props.path) {
                try {
                    targetPath = decodeURIComponent(props.path);
                } catch (e) {
                    console.warn('Failed to decode path prop:', props.path, e);
                    targetPath = props.path;
                }
            }

            // Use new data only if there no value in the storage, or use the path from props
            if (!currentDirectory.value || props.path) {
                currentDirectory.value = targetPath;
            }
            initialPath.value = homeDir
            fetchVFSStat(homeDir)

            // Load the initial directory after setting the current directory
            // This ensures the file list is populated immediately after login
            loadDirectory(targetPath)
        })
        .catch((error) => {
            if (error) {
                displayErrorMessage(error)
                // Force check the backend health
                fetchBackendServiceStatus()
            }
        })

    // Send a health check every 30 seconds
    interval = window.setInterval(() => {
        fetchBackendServiceStatus();
        fetchXrdLatency();
    }, 30000);
    // Make the first call immediately
    fetchBackendServiceStatus()
    fetchXrdLatency()
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
        if (row.name === '..') {
            // Navigate up one level
            const pathParts = currentDirectory.value.split('/').filter(part => part !== '');
            if (pathParts.length > 0) {
                pathParts.pop(); // Remove the last directory
                const parentPath = pathParts.length > 0 ? '/' + pathParts.join('/') : '/';

                // Make sure we don't go above the initial path
                if (parentPath === initialPath.value ||
                    (initialPath.value.startsWith(parentPath) && parentPath !== '/') ||
                    (parentPath.startsWith(initialPath.value) && parentPath.length >= initialPath.value.length)) {
                    currentDirectory.value = parentPath;
                    console.log('selectDir go up to: %s', currentDirectory.value);

                    // Push the new path to the router
                    routerPushNewPath(currentDirectory.value);
                }
            }
        } else {
            // Regular directory navigation
            // IMPORTANT: Calculate the target path but DON'T update currentDirectory yet
            // We need to verify user has permission to access this directory first
            const targetPath = joinPaths(currentDirectory.value, row.name);
            console.log('Attempting to navigate to: %s', targetPath);

            try {
                // Show loading state
                tableLoading.value = true;
                tableData.value = [];

                // Try to list the target directory BEFORE navigating
                // This will trigger authorization check and fail fast if user doesn't have access
                const resp = await getItemsInDir(targetPath);

                // Success! User has permission, now we can safely navigate
                currentDirectory.value = targetPath;
                console.log('selectDir: %s', currentDirectory.value);

                // Update the table with the new directory contents
                if (resp.data.items != null) {
                    tableData.value = resp.data.items;
                    pageSize.value = resp.data.pageSize;
                    totalPages.value = resp.data.totalPages;
                    totalItems.value = resp.data.totalItems;
                    totalFileCount.value = resp.data.totalFileCount;
                    totalFolderCount.value = resp.data.totalFolderCount;
                    cumulativeFileSize.value = filters.prettyBytes(resp.data.cumulativeFileSize);
                    if (resp.data.queryTimeMs != null) updateQueryTime(resp.data.queryTimeMs);
                } else {
                    tableData.value = [];
                }

                // Push the new path to the router ONLY after successful navigation
                routerPushNewPath(currentDirectory.value);
            } catch (error) {
                // Authorization or other error occurred
                // currentDirectory.value is unchanged, so no path corruption
                console.error('Failed to navigate to directory:', targetPath, error);

                // The error will be displayed by the global error handler (auth:access-denied event)
                // or by the API error normalization in api.js
                ElMessage({
                    message: error.message || 'You do not have permission to access this directory',
                    type: 'error',
                    duration: 5000,
                    showClose: true
                });
            } finally {
                tableLoading.value = false;
            }
        }
    } else {
        // File selected - redirect to download handler
        handleDownloadFile(row);
    }
}

/**
 * Handle file download using StreamSaver.js
 * @param {Object} row - The file row object
 */
const handleDownloadFile = async (row: { name: string; size?: number; downloading?: boolean }) => {
    if (row.downloading) {
        console.log('Download already in progress for:', row.name);
        return;
    }

    const filePath = joinPaths(currentDirectory.value, row.name);

    try {
        // Show confirmation dialog
        await ElMessageBox.confirm(
            'Do you want to download this file?',
            filePath,
            {
                confirmButtonText: 'Download',
                cancelButtonText: 'Cancel',
                type: 'info'
            }
        );

        // Set downloading state
        row.downloading = true;

        ElMessage({
            type: 'info',
            message: `Starting download: ${row.name}`,
            duration: 2000
        });

        // Use StreamSaver for download
        const result = await DownloadService.downloadFile(
            filePath,
            row.name,
            row.size || 0
        );

        if (result.success) {
            let message = `Download completed: ${row.name}`;
            if (result.speed) {
                const { mbps, duration, bytesPerSec } = result.speed;
                const sizeMB = ((row.size || 0) / (1024 * 1024)).toFixed(1);
                message += ` (${sizeMB} MB in ${duration}s at ${mbps} MB/s)`;
                // Record speed for estimation
                recordDownloadSpeed(bytesPerSec);
            }

            ElMessage({
                type: 'success',
                message: message,
                duration: 5000 // Longer duration for speed info
            });
        } else {
            ElMessage({
                type: 'error',
                message: `Download failed: ${result.error}`,
                duration: 5000
            });
        }
    } catch (error) {
        if (error === 'cancel') {
            console.log('Download cancelled by user');
        } else {
            console.error('Download error:', error);
            ElMessage({
                type: 'error',
                message: `Download error: ${error instanceof Error ? error.message : 'Unknown error'}`,
                duration: 5000
            });
        }
    } finally {
        // Clear downloading state
        row.downloading = false;
    }
}

/**
 * Function to get the virtual filesystem statistics.
 *
 * @param {string} path - The path to get stats for.
 * @returns {void}
 */
const fetchVFSStat = (path: string = '/') => {
    getVirtualFSStat(path)
        .then(resp => { vfsStat.value = resp.data.data })
        .catch((error) => {
            if (error) {
                console.warn('Failed to fetch VFS stats:', error.message);
                vfsStat.value = null;
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

    // Decode the path parameter if it was encoded
    if (path && path !== initialPath.value) {
        try {
            path = decodeURIComponent(path);
        } catch (e) {
            console.warn('Failed to decode path parameter:', path, e);
        }
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
        pathTmp = pathTmp.join('/');
    }

    // Encode the path for URL safety
    const encodedPath = encodeURIComponent(pathTmp);
    router.push({ name: 'browse', params: { path: encodedPath } });
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
        if (resp.data.items != null) {
            tableData.value = resp.data.items;
            pageSize.value = resp.data.pageSize; // Update pageSize
            totalPages.value = resp.data.totalPages; // Update totalPages
            totalItems.value = resp.data.totalItems; // Update totalItems
            totalFileCount.value = resp.data.totalFileCount; // Update totalFileCount
            totalFolderCount.value = resp.data.totalFolderCount; // Update totalFolderCount
            cumulativeFileSize.value = filters.prettyBytes(resp.data.cumulativeFileSize); // Update cumulativeFileSize
            if (resp.data.queryTimeMs != null) updateQueryTime(resp.data.queryTimeMs);
            console.log("pageSize: %d, totalPages: %d, totalItems: %d, totalFileCount: %d, totalFolderCount: %d, cumulativeFileSize: %d", pageSize.value, totalPages.value, totalItems.value, totalFileCount.value, totalFolderCount.value, cumulativeFileSize.value);
        } else {
            // Handle empty directory gracefully - this is normal and not an error
            tableData.value = [];
            // Set default values for empty directory
            pageSize.value = PAGE_SIZE;
            totalPages.value = 0;
            totalItems.value = 0;
            totalFileCount.value = 0;
            totalFolderCount.value = 0;
            cumulativeFileSize.value = "0 B";

            // Only throw an error if there's an actual error code and message
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
            // Handle empty directory gracefully - this is normal and not an error
            tableData.value = [];
            // Only throw an error if there's an actual error code and message
            if (resp.data.code != 200 && resp.data.error != "") {
                throw new Error(resp.data.error);
            }
        }
    } catch (error) {
        // Check backend health and show error
        await fetchBackendServiceStatus();
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
/* Main container styles */
.browse-view {
    height: 100%;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    /* Hide overflow on the main container */
}

.file-browser-container {
    flex: 1;
    display: flex;
    flex-direction: column;
    height: 100%;
    overflow: hidden;
    /* Hide overflow on the file browser container */
    min-height: 0;
    /* Important: allows flex child to shrink below content size */
}

/* Sticky header section that contains toolbar and pagination */
.sticky-header-section {
    position: sticky;
    top: 0;
    z-index: 1000;
    /* Higher z-index to ensure it stays above content */
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-light);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
    overflow: visible;
    /* Allow content to be visible but prevent scrollbars */
    flex-shrink: 0;
    /* Prevent shrinking */
}

.toolbar-header {
    background: var(--el-bg-color);
    color: var(--el-text-color-primary);
    padding: 16px 24px;
    border-bottom: 1px solid var(--el-border-color-lighter);
    overflow: hidden;
    /* Prevent scrollbars on toolbar */
}

.pagination-header {
    display: flex;
    justify-content: center;
    background: var(--el-bg-color);
    padding: 12px 24px;
    border-top: 1px solid var(--el-border-color-lighter);
    overflow: hidden;
    /* Prevent scrollbars on pagination */
}

/* Scrollable file table container */
.file-table-container {
    flex: 1;
    overflow-y: auto;
    overflow-x: hidden;
    /* Hide horizontal scrollbar */
    padding: 16px 24px;
    background: var(--el-bg-color-page);
    min-height: 0;
    /* Important: allows flex child to shrink below content size */
    display: flex;
    flex-direction: column;
    position: relative;
    /* For absolute positioning of empty state overlay */
}

/* Ensure the table has proper spacing from the scrollbar */
:deep(.el-table) {
    margin-right: 8px;
    /* Add margin to prevent overlap with scrollbar */
}

/* Fixed right column styling to prevent scrollbar overlap */
:deep(.el-table .el-table__fixed-right) {
    right: 8px;
    /* Offset the fixed right column to account for scrollbar */
}

/* GitHub-style pagination sizing */
:deep(.el-pagination) {
    font-size: 12px;
}

:deep(.el-pagination .el-pager li) {
    font-size: 12px;
}

/* Responsive adjustments */
@media (max-width: 768px) {

    .toolbar-header,
    .file-table-container,
    .pagination-header {
        padding: 12px 16px;
    }

    /* Adjust table spacing on mobile */
    :deep(.el-table) {
        margin-right: 4px;
    }

    :deep(.el-table .el-table__fixed-right) {
        right: 4px;
    }
}
</style>
