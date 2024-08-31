# List directory API

Get files and folder from the given directory.

## Method title

```plaintext
POST /dir
```

### Parameters

| Attribute                | Type     | Required | Description                         |
|--------------------------|----------|----------|-------------------------------------|
| `path`    `              | string   | Yes      | Full path to the directory to list. |

### Response

If successful, returns

- code: `200`,
- message: `success`,  
- data: is an array of the following attributes:

| Attribute                | Type      | Description                   |
|--------------------------|-----------|-------------------------------|
| `name`                   | string    | File name.                    |
| `type`                   | string    | Can be one of: "dir", "files" |
| `date_time`              | string    | Date time of the file.        |
| `size`                   | integer   | File size in bytes.           |

### Example request

```shell
curl -X POST -d '{"path":"/tmp/"}' http://localhost:22000/dir
```

### Example response

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
