# Health API

Status details of the backend service.

## Method title

```plaintext
POST /health
```

### Parameters

No parameters.

### Response

If successful, returns

- code: `200`,
- message: `success`,  
- data: is a `string` value, representing the health status of the service:
  - `ok` - the service is alive. Any other response indicates a problem on the service.

### Example request

```shell
curl --url "http://localhost:22000/health"
```

### Example response

```json
{"code":200,"data":"ok","msg":"success"}
```
