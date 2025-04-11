# Backend Release Notes

## [0.6.0] - NOT YET RELEASED

- Added: RPM package build. (GH-20)
- Added: Backend-For-Frontend (BFF) pattern for secure authentication. (GH-19)
- Added: HTTP-only cookie session management for improved security.  (GH-19)
- Added: Server-side token refresh handling.  (GH-19)
- Fixed: CORS implementation to properly support credentials.  (GH-19)

## [0.5.0] - 2024-10-02

- Added: Save directory listings in a runtime cache.
- Added: Retrieve directory listings from the cache, if available.
- Added: Unit-tests.

## [0.4.0] - 2024-09-09

- Fixed: Pull file names with white spaces.
- Added: Sanitation job to periodically check and clean staged temporary files.
- Added: XRD settings moved to the backend configuration.
- Added: Detailed error response. (GH-5)
- Added: REST API documentation.
- Added: Use JS plugins to improve the code.
