# List directory API

## Get Files and Folders from a Directory

This API endpoint allows you to retrieve a list of files and folders from a given directory.

**Method:**

```plaintext
POST /dir
```

**Parameters:**

| Attribute | Type   | Required | Description                             |
| --------- | ------ | -------- | --------------------------------------- |
| `path`    | string | Yes      | The full path to the directory to list. |

**Response:**

If the request is successful, the API will return a response with the following attributes:

| Attribute    | Type    | Description                              |
| ------------ | ------- | ---------------------------------------- |
| `name`       | string  | The name of the file or folder.          |
| `type`       | string  | The type of the item ("dir" or "file").  |
| `date_time`  | string  | The date and time of the item.           |
| `size`       | integer | The size of the file in bytes.           |
| `totalItems` | integer | The total number of items in the folder. |
| `pageSize`   | integer | The number of items per page.            |
| `totalPages` | integer | The total number of pages.               |

**Example Request:**

```shell
curl -X POST -d '{"path":"/tmp/"}' http://localhost:22000/dir
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
    "pageSize": 4,
    "totalPages": 2
}

## Get Paginated Files and Folders from a Directory

This API endpoint allows you to retrieve a paginated list of files and folders from a given directory.

**Method:**

```plaintext
POST /dir/page
```

**Response:**

If the request is successful, the API will return a response with the following attributes:

| Attribute    | Type    | Description                             |
| ------------ | ------- | --------------------------------------- |
| `name`       | string  | The name of the file or folder.         |
| `type`       | string  | The type of the item ("dir" or "file"). |
| `date_time`  | string  | The date and time of the item.          |
| `size`       | integer | The size of the file in bytes.          |
| `pageSize`   | integer | The number of items per page.           |
| `totalPages` | integer | The total number of pages.              |

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
            "name": "1ed3b4a64e1741b5a5b539d2ebb1b9b8_501"
        }
    ],
    "totalPages": 5
}
```
