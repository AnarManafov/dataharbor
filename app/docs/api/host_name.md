# Server's Host API

**Host name:**

Get information about the host where the xrootd server is running.

**Method title:**

```plaintext
GET /host_name
```

**Parameters::**

This endpoint does not require any parameters.

**Response:**

If the request is successful, the API will return the following response:

- `code`: `200`
- `message`: `success`
- `data`: A `string` value representing the host name of the xrootd server.

**Example request:**

```shell
curl --url "http://localhost:22000/host_name"
```

**Example response:**

```json
{
    "code": 200,
    "data": "localhost",
    "msg": "success"
}
```

