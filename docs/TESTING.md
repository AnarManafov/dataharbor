# Testing Guide

This document covers the existing test structure and commands for running tests in DataHarbor.

## Test Coverage Requirements

- **Backend Minimum Coverage**: 80% for all packages
- **Controller Coverage**: 90% (critical path)
- **Middleware Coverage**: 85% (security critical)
- **Frontend Coverage**: 75% minimum

## Backend Testing (Go)

### Current Test Structure

```text
app/
├── controller/
│   ├── auth_test.go
│   ├── fs_test.go
│   └── health_test.go
├── middleware/
│   ├── auth_middleware_test.go
│   └── cors_test.go
├── common/
│   ├── logger_test.go
│   └── xrd_test.go
└── main_test.go
```

### Running Backend Tests

```bash
cd app

# Run all tests
go test -v ./...

# Run tests with coverage
go test -cover ./...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Run specific test package
go test -v ./controller

# Run specific test function
go test -v ./controller -run TestHealthHandler

# Run tests with race detection
go test -race ./...
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
# Quick test run (backend + frontend)
cd app && go test ./... && cd ../web && npm test

# Full coverage report
cd app && go test -cover ./... && cd ../web && npm run test:coverage

# Watch mode for active development
cd web && npm run test:watch
```

### Debugging Tests

```bash
# Backend: Run specific test with verbose output
go test -v ./controller -run TestSpecificFunction

# Frontend: Run single test file in watch mode
npm test -- --watch FileExplorer.test.js

# Frontend: Debug mode
npm run test:debug
```

## Test Types

### Unit Tests
- Individual function and method testing
- Isolated component testing  
- Mock external dependencies (XROOTD, OIDC)

### Integration Tests
- API endpoint testing
- Authentication flow testing
- File system operations

### End-to-End Tests
- Complete user workflows
- Cross-browser compatibility
- Full authentication flow

## Best Practices

1. **Test Organization**
   - Keep tests close to source code
   - Use descriptive test names
   - Group related tests together

2. **Mock Usage**
   - Mock external dependencies (XROOTD, OIDC providers)
   - Use dependency injection for testability
   - Reset mocks between tests

3. **Performance**
   - Keep unit tests fast (< 100ms each)
   - Use parallel execution where possible
   - Optimize test data setup/teardown
