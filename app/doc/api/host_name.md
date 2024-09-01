# Server's Host API

Get information about the host, where xrootd is running.

## Method title

```plaintext
GET /host_name
```

### Parameters

No parameters.

### Response

If successful, returns

- code: `200`,
- message: `success`,  
- data: is a `string` value, representing a host name of the xrootd server.

### Example request

```shell
curl --url "http://localhost:22000/host_name"
```

### Example response

```json
{"code":200,"data":"localhost","msg":"success"}
```
