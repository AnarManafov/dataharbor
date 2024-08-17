import request from "./request";

// Get the current user's home dir
export const getHomeDirPath = () => {
  return request.get("home_dir");
};

// Get the items in dir
export const getItemsInDir = (path) => {
  return request.post("dir", { path });
};

// Get the current user's home dir
export const getHostName = () => {
    return request.get("host_name");
  };