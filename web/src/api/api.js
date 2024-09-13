import request from "./request";

// Get the initial directory
export const getInitialDirPath = () => {
    return request.get("initial_dir");
};

// Get the items in dir
export const getItemsInDir = (path) => {
    return request.post("dir", { path });
};

// Get paged items in dir
export const getPagedItemsInDir = (path, page) => {
    return request.post("/dir/page", { path, page });
};

// Get the xrootd server's host name
export const getHostName = () => {
    return request.get("host_name");
};

// Request to stage an xrd file for download
export const getFileStagedForDownload = (path) => {
    return request.post("stage_file", { path });
};

// Check health of the backend service
export const getBackendHealth = () => {
    return request.get("health");
};