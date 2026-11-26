# DataHarbor API specification

`openapi.yaml` is the OpenAPI 3.1 description of the DataHarbor backend API.

It is written by hand and describes the current behaviour of the backend as it
is. Where the code and `docs/API.md` disagree, the code wins. `docs/API.md` will
be regenerated from this specification in a later ticket (#116).

## How the specification is kept honest

- Route test: `app/route/openapi_test.go` compares the routes registered in
  `route/routes.go` with the operations in the specification. It runs as part of
  `go test ./...`.
- In-process contract test: `app/route/contract_test.go` validates real handler
  responses against the specification.
- `make api-lint` lints the specification with Redocly (config in
  `api/redocly.yaml`). CI runs it in the "OpenAPI specification lint" job.
- `make api-probe` validates a running backend against the specification.

## Running the probe

The probe calls the endpoints of a running backend and validates every response
against `openapi.yaml`. It is configured through environment variables:

- `API_PROBE_BASE_URL`: backend base URL, for example `https://localhost:22000`
  (required).
- `API_PROBE_COOKIE`: value of the `dataharbor-session` cookie (required). Log in
  with the browser, then copy the cookie value from the browser's developer
  tools. A leading `dataharbor-session=` is accepted and removed.
- `API_PROBE_INSECURE`: set to `1` to skip TLS certificate verification, needed
  for the self-signed certificates of the dev stack. Off by default.

The cookie is read from the environment only. It is never accepted as an
argument, never printed, and all output is passed through a redactor that masks
it. To keep it out of your shell history, read it without echo:

```bash
read -rs API_PROBE_COOKIE && export API_PROBE_COOKIE
API_PROBE_BASE_URL=https://localhost:22000 API_PROBE_INSECURE=1 make api-probe
```

The probe does not follow redirects and does not store cookies.

### Output

One line per check, then a summary:

```text
PASS|FAIL|SKIP  METHOD path  status  [reason]
SUMMARY  pass=N fail=N skip=N
```

Every operation in the specification appears in the output, either as a check or
as a SKIP line with the reason.

Behind the nginx gateway of the docker deployments, `GET /health` is answered
by nginx and not by the backend. The probe reports that as a SKIP; `GET
/api/health` reaches the same handler and is checked.

### Exit codes

`make api-probe` fails when the probe exits non-zero. The probe binary itself
(`bin/api-probe`, built by the make target) uses:

- `0`: every check passed or was skipped.
- `1`: at least one check failed (a response did not match the specification).
- `2`: usage or configuration error, for example a missing environment variable.

### Endpoints that are never called

- `POST /api/auth/logout`: it would end the session.
- `GET /api/auth/callback`: it needs an authorization code from the identity
  provider.

### Upload checks (opt-in)

The upload endpoints write to the server, so they run only on request. Pass a
remote scratch directory the probe may write to:

```bash
API_PROBE_FLAGS="-upload -scratch-dir /remote/dir" make api-probe
```

`-upload` and `-scratch-dir` must be given together; the directory must be an
absolute path without `..`. The probe uploads one small file (named
`api-probe-<timestamp>-<suffix>.txt`) and aborts a second session. The uploaded
file stays in the scratch directory, because there is no delete endpoint.

## Changing the API

Change the handler, the specification and the tests in the same commit.
Adding a route without describing it in `openapi.yaml` makes the route test fail.
