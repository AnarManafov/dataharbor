# Testing Guide

[← Back to Documentation](./README.md)

This document covers the existing test structure and commands for running tests in DataHarbor.

## Backend Testing (Go)

### Test Structure

```text
app/
├── common/
│   ├── logger_test.go
│   ├── sysconf_test.go
│   └── xrd_test.go
├── controller/
│   ├── auth_test.go
│   ├── fs_test.go
│   ├── health_test.go
│   ├── main_test.go
│   └── xrd_test.go
├── middleware/
│   ├── access_middleware_test.go
│   ├── auth_middleware_test.go
│   ├── cors_test.go
│   ├── main_test.go
│   ├── recovery_test.go
│   └── trace_middleware_test.go
├── response/
│   ├── error_test.go
│   └── response_test.go
├── route/
│   ├── main_test.go
│   └── routes_test.go
├── test/
│   ├── config_benchmark_test.go
│   └── config_integration_test.go
├── util/
│   └── util_test.go
└── main_test.go
```

### Test Types

- **Unit Tests**: Located throughout `app/` subdirectories (e.g., controller, middleware, common, response, route, util). These test individual functions or components in isolation.
- **Integration Tests**: In `app/test/config_integration_test.go`, covering configuration and XROOTD client logic working together.
- **Benchmark Tests**: In `app/test/config_benchmark_test.go`, measuring performance of XROOTD client creation and related operations.

### Running Integration and Benchmark Tests

To run only the integration tests:

```bash
make test-integration
```

To run only the benchmark tests (benchmarks are NOT run by default):

```bash
make test-benchmark
```

To run all tests (unit, integration, etc.) in the `app/test` directory (does NOT include benchmarks):

```bash
cd app/test
go test -v ./...
```

### Running All Backend Tests

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage
make test-coverage

# Generate HTML coverage report
make test-coverage-html

# Run specific test package
cd app && go test -v ./controller

# Run specific test function
cd app && go test -v ./controller -run TestHealthHandler

# Run tests with race detection
make test-race
```

## Frontend Testing (Vue.js)

### Current Test Structure

```text
web/
├── src/
│   ├── components/
│   │   └── __tests__/
│   │       ├── FileExplorer.test.js
│   │       └── LoginForm.test.js
│   ├── composables/
│   │   └── __tests__/
│   │       ├── useAuth.test.js
│   │       └── useFileOps.test.js
│   └── stores/
│       └── __tests__/
│           ├── auth.test.js
│           └── files.test.js
├── tests/
│   ├── unit/
│   ├── integration/
│   └── e2e/
└── vitest.config.js
```

### Running Frontend Tests

```bash
cd web

# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Run tests with coverage
npm run test:coverage

# Run specific test file
npm test -- FileExplorer.test.js

# Run e2e tests
npm run test:e2e

# Run tests in CI mode
npm run test:ci
```

## Test Commands Summary

### Development Workflow

```bash
# Quick test run (backend)
make test

# Full coverage report
make test-coverage-html
```

### Debugging Tests

```bash
# Backend: Run specific test with verbose output
cd app && go test -v ./controller -run TestSpecificFunction

# Frontend: Run single test file in watch mode
npm test -- --watch FileExplorer.test.js

# Frontend: Debug mode
npm run test:debug
```

---

## Related Documentation

- **[Development Guide](./DEVELOPMENT.md)** - Development workflow and CI/CD
- **[Backend Development](./BACKEND.md)** - Backend testing context
- **[Frontend Development](./FRONTEND.md)** - Frontend testing context
- **[Troubleshooting](./TROUBLESHOOTING.md)** - Debug test failures

---

[← Back to Documentation](./README.md) | [↑ Top](#testing-guide)
