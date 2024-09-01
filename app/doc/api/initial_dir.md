# Initial directory API

Get the initial directory of the current xrootd server.

## Method title

```plaintext
GET /initial_dir
```

### Parameters

No parameters are required.

### Response

If successful, returns

- code: `200`,
- message `success`,  
- data: is a `string` value, representing an initial directory of the xrootd server.

### Example request

```shell
curl --url "http://localhost:22000/initial_dir"
```

### Example response

```json
{"code":200,"data":"/tmp/","msg":"success"}
```
