# Stage file API

## Stage File API

This API allows you to stage a file to the server's public location, making it ready for download.

**Method:**

```plaintext
POST /stage_file
```

**Parameters:**

| Attribute | Type   | Required | Description                         |
| --------- | ------ | -------- | ----------------------------------- |
| `path`    | string | Yes      | The full path to the file to stage. |

**Response:**

If the request is successful, the API will return a response with the following attributes:

| Attribute | Type   | Description                            |
| --------- | ------ | -------------------------------------- |
| `path`    | string | The full file path to the staged file. |

**Example Request::**

```shell
curl -X POST -d '{"path":"/tmp/file_to_stage"}' http://localhost:22000/stage_file
```

**Example Response:**

```json
{
    "code": 200,
    "data": {
        "path": "/tmp/delete_me/stg_2478601451/file_to_stage"
    },
    "msg": "success"
}

