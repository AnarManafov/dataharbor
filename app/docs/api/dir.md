# List directory API

## Get Files and Folders from a Directory

This API endpoint allows you to retrieve a list of files and folders from a given directory.

### Method

```plaintext
POST /dir
```

### Parameters

| Attribute | Type   | Required | Description                             |
| --------- | ------ | -------- | --------------------------------------- |
| `path`    | string | Yes      | The full path to the directory to list. |

### Response

If the request is successful, the API will return a response with the following attributes:

| Attribute   | Type    | Description                             |
| ----------- | ------- | --------------------------------------- |
| `name`      | string  | The name of the file or folder.         |
| `type`      | string  | The type of the item ("dir" or "file"). |
| `date_time` | string  | The date and time of the item.          |
| `size`      | integer | The size of the file in bytes.          |

### Example Request

```shell
curl -X POST -d '{"path":"/tmp/"}' http://localhost:22000/dir
```

### Example Response

```json
{
    "code": 200,
    "data": [
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
            "date_time": "2024-08-20 18:46:06",
            "size": 0
        },
        {
            "name": "name_it",
            "type": "file",
            "date_time": "2024-08-31 15:48:13",
            "size": 0
        }
    ],
    "msg": "success"
}
```

