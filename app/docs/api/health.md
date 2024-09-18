# Health API

## Health of the backend service

This API provides information about the status of the backend service.

**Method:**

```plaintext
GET /health
```

**Parameters:**

This API does not require any parameters.

**Response:**

If the request is successful, the API will return a JSON response with the following fields:

- `code`: The HTTP status code, which will be `200` for a successful request.
- `message`: A message indicating the success of the request, which will be `success`.
- `data`: A string value representing the health status of the service. The value will be `ok` if the service is alive. Any other value indicates a problem with the service.

**Example Request:**

```shell
curl --url "http://localhost:22000/health"
```

**Example Response:**

```json
{
    "code": 200,
    "data": "ok",
    "message": "success"
}
```

## Error Response

If the request fails, the API will return an error response with the following attributes:

| Attribute | Type    | Description                            |
| --------- | ------- | -------------------------------------- |
| `code`    | integer | The status code of the error response. |
| `message` | string  | A description of the error.            |

**Example Error Response:**

```json
{
    "code": 500,
    "message": "Internal Server Error"
}
```
