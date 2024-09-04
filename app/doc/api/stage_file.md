# Stage file API

Stage a file to server's public location to make it download ready.

## Method title

```plaintext
POST /stage_file
```

### Parameters

| Attribute | Type   | Required | Description                       |
| --------- | ------ | -------- | --------------------------------- |
| `path`    | string | Yes      | A full path to the file to stage. |

### Response

If successful, returns

- code: `200`,
- message: `success`,  
- data: is an array of the following attributes:

| Attribute | Type   | Description                           |
| --------- | ------ | ------------------------------------- |
| `path`    | string | A full file path to the staged files. |

### Example request

```shell
curl -X POST -d '{"path":"/tmp/file_to_stage"}' http://localhost:22000/stage_file
```

### Example response

```json
{
    "code": 200,
    "data": {
        "path": "/tmp/delete_me/stg_2478601451/file_to_stage"
    },
    "msg": "success"
}
```
