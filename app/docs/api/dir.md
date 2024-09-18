# List directory API

## Get Files and Folders from a Directory

This API endpoint allows you to retrieve a list of files and folders from a given directory.

**Method:**

```plaintext
POST /dir
```

**Parameters:**

| Attribute  | Type   | Required | Description                             |
| ---------- | ------ | -------- | --------------------------------------- |
| `path`     | string | Yes      | The full path to the directory to list. |
| `pageSize` | uint32 | Yes      | The number of items per page.           |

**Response:**

If the request is successful, the API will return a response with the following attributes:

| Attribute            | Type    | Description                              |
| -------------------- | ------- | ---------------------------------------- |
| `code`               | integer | The status code of the response.         |
| `items`              | array   | The list of files and folders.           |
| `items[].name`       | string  | The name of the file or folder.          |
| `items[].type`       | string  | The type of the item (`file` or `dir`).  |
| `items[].date_time`  | string  | The date and time of the item.           |
| `items[].size`       | integer | The size of the item in bytes.           |
| `totalItems`         | integer | The total number of items in the folder. |
| `pageSize`           | integer | The number of items per page.            |
| `totalPages`         | integer | The total number of pages.               |
| `totalFileCount`     | integer | The total number of files.               |
| `totalFolderCount`   | integer | The total number of folders.             |
| `cumulativeFileSize` | integer | The cumulative size of all files.        |


**Example Request:**

```shell
curl -X POST -d '{"path":"/tmp/" , "pageSize": 10}' http://localhost:22000/dir
```

**Example Response:**

```json
{
    "code": 200,
    "items": [
        {"name": "com.apple.launchd.XRkDTZGEv6", "type": "dir", "date_time": "2024-09-12 14:58:59", "size": 96},
        {"name": "delete_me", "type": "dir", "date_time": "2024-09-13 16:09:05", "size": 64},
        {"name": "ems_id.conf", "type": "file", "date_time": "2024-09-12 14:59:15", "size": 0},
        {"name": "f272e54842dc8e993516d168744b5dc4_501", "type": "file", "date_time": "2024-09-12 14:59:22", "size": 0}
    ],
    "totalItems": 4,
    "pageSize": 4,
    "totalPages": 1,
    "totalFileCount": 2,
    "totalFolderCount": 2,
    "cumulativeFileSize": 160
}
```

## Get Paginated Files and Folders from a Directory

This API endpoint allows you to retrieve a paginated list of files and folders from a given directory.

**Method:**

```plaintext
POST /dir/page
```

**Parameters:**

| Attribute  | Type   | Required | Description                             |
| ---------- | ------ | -------- | --------------------------------------- |
| `path`     | string | Yes      | The full path to the directory to list. |
| `page`     | uint32 | Yes      | The page number to retrieve.            |
| `pageSize` | uint32 | Yes      | The number of items per page.           |

**Response:**

If the request is successful, the API will return a response with the following attributes:

| Attribute           | Type    | Description                             |
| ------------------- | ------- | --------------------------------------- |
| `code`              | integer | The status code of the response.        |
| `items`             | array   | The list of files and folders.          |
| `items[].name`      | string  | The name of the file or folder.         |
| `items[].type`      | string  | The type of the item (`file` or `dir`). |
| `items[].date_time` | string  | The date and time of the item.          |
| `items[].size`      | integer | The size of the item in bytes.          |
| `pageSize`          | integer | The number of items per page.           |
| `totalPages`        | integer | The total number of pages.              |

**Example Request:**

```shell
curl -X POST -d '{"path":"/tmp/", "page": 1, "pageSize": 10}' http://localhost:22000/dir/page
```

**Example Response:**

```json
{
    "code": 200,
    "items": [
        {
            "name": "test",
            "type": "dir",
            "date_time": "2024-08-16 13:22:45",
            "size": 96
        },
        {
            "name": "test1",
            "type": "dir",
            "date_time": "2024-08-16 13:22:45",
            "size": 96
        },
        {
            "name": "1ed3b4a64e1741b5a5b539d2ebb1b9b8_501",
            "type": "file",
            "date_time": "2024-08-16 13:22:45",
            "size": 0
        }
    ],
    "pageSize": 3,
    "totalPages": 5
}
```

## Error Response

If the request fails, the API will return an error response with the following attributes:

| Attribute | Type    | Description                            |
| --------- | ------- | -------------------------------------- |
| `code`    | integer | The status code of the error response. |
| `message` | string  | A description of the error.            |

**Example Error Response:**

```json
{
    "code": 400,
    "message": "Invalid directory path"
}
```