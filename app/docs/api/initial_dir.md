# Initial directory API

## Initial Directory

This endpoint retrieves the initial directory of the current xrootd server.

**Method:**

```plaintext
GET /initial_dir
```

**Parameters:**

No parameters are required for this endpoint.

**Response:**

If the request is successful, the API will return a response with the following structure:

```json
{
    "code": 200,
    "message": "success",
    "data": "/tmp/"
}
```

The response includes the following fields:

- `code`: The HTTP status code of the response (200 for success).
- `message`: A message indicating the status of the request.
- `data`: The initial directory of the xrootd server, represented as a string value.

**Example Request:**

```shell
curl --url "http://localhost:22000/initial_dir"
```

**Example Response:**

```json
{
    "code": 200,
    "data": "/tmp/",
    "message": "success"
}
```
