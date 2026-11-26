// Command api-probe calls the endpoints of a running DataHarbor backend and
// validates every live response against the OpenAPI specification.
//
//	API_PROBE_BASE_URL=https://localhost:22000 API_PROBE_COOKIE=... API_PROBE_INSECURE=1 \
//	    go run ./cmd/api-probe -spec ../api/openapi.yaml
//
// See package internal/apiprobe for the checks, flags and exit codes.
package main

import (
	"os"

	"github.com/AnarManafov/dataharbor/app/internal/apiprobe"
)

func main() {
	os.Exit(apiprobe.Main(os.Args[1:], os.Getenv, os.Stdout, os.Stderr))
}
