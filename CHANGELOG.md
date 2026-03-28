# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.9] - 2026-03-28

### Added

- centralize session cookie options for security (auth) (80e3637)
- enhance token and user info caching mechanisms (auth) (be78164)

## [1.0.8] - 2026-03-28

### Added

- add XRD ping endpoint and network stats tracking (backend,frontend) (8a5c998)

## [1.0.7] - 2026-03-28

### Added

- add XRootD virtual filesystem statistics (backend,frontend) (f7f6343)

## [1.0.6] - 2026-03-27

### Changed

- reorganize TLS and security settings (xrootd-prod.cfg) (5c72e24)

## [1.0.5] - 2026-03-27

### Fixed

- adjust SameSite cookie settings for OIDC flows (auth) (a23cc7d)

### Maintenance

- Updated coverage badge. (4d67ccc)

## [1.0.4] - 2026-03-26

### Added

- update backend Dockerfile and entrypoint script (docker) (fbe2f09)

## [1.0.3] - 2026-03-26

### Fixed

- use configured frontend URL for OIDC redirect URI and improve Docker volume mounts (backend,docker) (5489813)

## [1.0.2] - 2026-03-25

### Added

- add cleanup option for unreferenced layers (build-docker) (6e60179)
- use devcontainer features rather than Go base image (devcontainer) (00d4a71)
- enhance feature descriptions and icons (HomeView) (70008d3)
- update icon sizes for better visibility (HomeView) (9abc23a)
- add workspace mount for git worktrees (devcontainer) (59902d9)
- enhance deployment configuration and health checks (docker) (bdb00ea)

### Changed

- streamline image handling functions (build-docker) (9565800)

### Documentation

- add coverage report commands to backend guide (cc4b0af)

### Maintenance

- Updated coverage badge. (6 commits)
- remove unused CMake and Go test files (a4cefb9)
- remove sandbox-related scripts and configurations (077960f)
- modernize Go idioms and update base image versions (e1a58cd)

### Tests

- add comprehensive tests for middleware and requests [GH-42] (6e7ecaa)

### Build

- update package versions (dependencies) (80e08cd)
- update dependencies (dependencies) (cc740fd)
- update package versions for stability (dependencies) (a1f8666)
- update dependencies for improved stability (deps) (49fd62b)

### CI

- update GitHub Actions to use latest action versions (90d0fa8)

## [1.0.1] - 2025-12-08

### Fixed

- resolve all CI/CD workflow issues (6ae0105)

## [1.0.0] - 2025-12-08

### Added

- add Docker Compose deployment and ZTN/TLS XRootD support (63b6089)
- add Dockerfile and configuration files (devcontainer) (c275b69)
- add Go configuration settings (vscode) (48a2eb2)
- add comprehensive configuration tests (tests) (d2cb35b)

### Fixed

- update dependencies to latest versions (go.mod) (a591164)

### Maintenance

- Updated coverage badge. (773debb, 8c3d60d, 45dfec4, 5bbbbd3)
- update VSCode settings exclusion (.gitignore) (4023018)

### Build

- add dos2unix for line ending normalization (docker) (d9a7e5c)

### Other

- feat(docker)!: add production Docker deployment (06dd378)

## [0.15.0] - 2025-10-16

### Added

- add ZTN protocol support for OAuth token authentication (xrootd) (a703d7c)
- enhance ZTN protocol configuration guide (docs) (e7dee77)
- Update RPM packaging for DataHarbor backend and frontend (packaging) (299166a)

### Fixed

- handle error when setting BEARER_TOKEN (xrd) (405aafc)

### Documentation

- update README for improved clarity and formatting (02e87b6)

### Maintenance

- Updated coverage badge. (9be86b3, ee885ea, a7145a8, 7b00abf)

### Style

- reorder XRDConfig fields for consistency (tests) (c401b1f)

### CI

- update workflow triggers for packaging changes (8fd3933)

### Other

- Add GSI Deployment Guide for DataHarbor (fe8e1c1)
- Update hep fork dependency with TLS/ZTN connection logging (05abd9f)

## [0.14.6] - 2025-10-08

### Added

- Add multi-architecture support for builds (ci) (65c0d93)

### Maintenance

- Updated coverage badge. (57ed1bd, e64acad)

### Other

- doc: improve documentation (4b292a3)

## [0.14.5] - 2025-08-11

### Fixed

- Update sync-versions script and CI workflow to handle package-lock.json files (ci) (5898100)

### Maintenance

- Update changelog header and generation logic (5e74af8)
- update dependency management instructions (docs) (a032a7e)

## [0.14.4] - 2025-08-07

### Maintenance

- clean up release notes formatting (d31fde9)

## [0.14.3] - 2025-08-07

### Maintenance

- update changelog formatting and release notes output (3d2075a)

## [0.14.2] - 2025-08-07

### Added

- enhance changelog and release notes generation (ci) (e1dfcd7)

### Maintenance

- update dependencies (eb72acc)
- Updated coverage badge. (c13e4e6)
- update release workflows (e090393)

## [0.14.1] - 2025-08-07

### Maintenance

- update dependencies (eb72acc)
- Updated coverage badge. (c13e4e6)
- update release workflows (e090393)

## [0.14.0] - 2025-08-05

### Added

- enhance user authentication display in navbar (nav) (c4b50fe)
- backend to support HTTPS for Keycloak [GH-34] (108357d)
- enhance directory listing response structure (api) (652dffb)
- native XRD client with streaming downloads [GH-10] (0efbf27)

### Changed

- migrate from viper to config package (config) (5c13038)

### Documentation

- Update README and add detailed dev doc (4c17380)

### Maintenance

- Rename from data-lake-ui to dataharbor [GH-36] (342996a)
- Updated coverage badge. (d91ade9, de78671, cf87325, 0aff3de, 0a7b63a)
- Update dependencies for improved stability (577d7de)

### Style

- Adjust component sizes and spacing (e0497ac)
- update font sizes and remove Bulma dependency (cd257d9)
- unify typography across components and styles (664d782)
- enhance layout and structure of file browser (5983dff)
- adjust sidebar width and improve table sorting (4bd70b5)

### Build

- update vue and babel dependencies to latest (deps) (c07b2e0)

### Other

- Refactor views for improved layout and styling (0573f31)
- doc: Refactor project documentation (fd6f7ae)

## [0.13.13] - 2025-05-19

### Added

- enhance logout process for improved security (auth) [GH-27] (83194d0)

### Documentation

- update changelog and release notes for v0.13.12 (8051ee9)

### Maintenance

- update dependencies in package.json (3481244)
- Updated coverage badge. (fcd7dea, 4098de7)
- update npm scripts for cross-platform (7035c45)

## [0.13.12] - 2025-05-05

### Added

- enhance RPM build process, CI workflows, and release notes generation (build) [GH-26] (562099f)
- switch to Python script for changelog generation (changelog) (845afa1)
- automate CHANGELOG and RELEASE_NOTES updates (changelog) (0b1858a)

### Documentation

- Update release notes for v0.13.9 [skip ci] (60f002a)
- Update changelog for v0.13.10 [skip ci] (4f4d0a8)

### Maintenance

- Update package versions to v0.13.8 [skip ci] (0e32a04)
- refactor version tag processing jobs (workflow) (2fbcb47)

## [0.13.11] - 2025-05-05

### Maintenance

- refactor version tag processing jobs (workflow) (ace4d8c)

## [0.13.10] - 2025-05-05

### Added

- automate CHANGELOG and RELEASE_NOTES updates (changelog) (2d841e1)

## [0.13.9] - 2025-05-05

### Added

- enhance RPM build process, CI workflows, and release notes generation (build) [GH-26] (fe53bd4)
- switch to Python script for changelog generation (changelog) (795cddf)

### Documentation

- Update release notes for v0.13.7 [skip ci] (638ddcb)

### Maintenance

- Update package versions to v0.13.7 [skip ci] (576f74d)

## [0.13.7] - 2025-04-25

### Fixed

- update job dependencies and output delimiters (workflow) (68c665a)

### Documentation

- Update release notes for v0.13.6 [skip ci] (0f5959f)

### Maintenance

- Update package versions to v0.13.6 [skip ci] (60925e6)

## [0.13.6] - 2025-04-25

### Documentation

- Update release notes for v0.13.5 [skip ci] (b9562f6)

### Maintenance

- Update package versions to v0.13.5 [skip ci] (050c79f)
- remove branch restriction for tag processing (workflow) (e35e030)

## [0.13.5] - 2025-04-25

### Maintenance

- refactor version tag processing and permissions (workflow) (4058f8a)

## [0.13.4] - 2025-04-25

### Maintenance

- update permissions and token usage for tag creation (workflow) (931ce11)

## [0.13.3] - 2025-04-25

### Maintenance

- enhance permissions for tag creation (workflow) (c259a1c)

## [0.13.2] - 2025-04-25

### Maintenance

- explicitly checkout and push to master branch (workflow) (9b76137)

## [0.13.1] - 2025-04-25

### Maintenance

- Update version to v0.13.0 in package.json files (7c97110)
- Update release notes for v0.13.0 (e299a2e)
- update CI workflows for versioning (workflow) (40a03ab)
- Updated coverage badge. (aef0155)
- refine CI workflows for consistency (workflow) (22cb104)
- standardize quotes in version tag processor (workflow) (58d6828)

## [0.13.0] - 2025-04-24

### Maintenance

- improve git pull process in auto-version (workflow) (952cf21)

## [0.12.0] - 2025-04-24

### Maintenance

- Update version to v0.11.0 in package.json files (763bce3)
- improve versioning and release notes process (workflow) (e2e7f20)

## [0.11.0] - 2025-04-24

### Maintenance

- Update version to v0.10.0 and release notes (2af751d)
- update auto-version and frontend workflows (workflow) (3feaa56)

## [0.10.0] - 2025-04-24

### Added

- update changelog generation process (build) (b9506dc)

### Maintenance

- Update version to v0.9.0 and release notes (bab99fb)

## [0.9.0] - 2025-04-24

### Added

- enhance RPM packaging process and changelog generation (build) (c0d997f)

### Maintenance

- Update version to v0.8.0 and release notes (eb1c686)

## [0.8.0] - 2025-04-24

### Added

- streamline auto versioning and release process (ci) (19cc1af)

### Maintenance

- Update version to v0.7.0 and release notes (82fcc95)

## [0.7.0] - 2025-04-24

### Added

- enhance publish release workflow (ci) (158ed93)

### Maintenance

- Update version to v0.6.0 and release notes (48195f8)

## [0.6.0] - 2025-04-24

### Added

- enhance auth UI and logout handling (Nav) [GH-19] (cb0490d)
- Add release publishing workflow & version management (9d9e7dc)

### Maintenance

- Updated coverage badge. (18 commits)

### Other

- RPM packaging for backend [GH-20] (b9b7d58)
- Add debug log to RPM build (bd75f25)
- Fix backend CI (f672ef3)
- CI fix for RPM artifacts (b9482fb)
- RPM packaging for frontend [GH-21] (8f5dcaf)
- RPM: generate change log at build time (729faa8)
- RPM changelog fix (3a20710)
- RPM packaging improvment (503a892)
- Fix backend packaging (8314038)
- Update backend dependencies (756df39)
- Add ngingx for RPM spec of frontend (b6b156e)
- Fix RPM build with nginx (6bc6515, 6dcd9c7)
- Fix nginx package conflict. (82f4d4f)
- Update documentation (671d7ba)
- backend RPM to use host arch as target (efacc0d)
- Update dependencies (638f99e, 0dcba13, c6ce6a7, 08f4555)
- Update frontend package (98d94e8)
- Fix frontend packaging (be1c9a4)
- Relocate asset files to work on prod and dev env (b5a1c3f)
- update dependencies (cd7f739)
- Update doc (5d6615c)
- Improve UI components and error handling (b3cbce7)
- Implement Keycloak OIDC authentication [GH-19] (8ef3d07)

## [0.5.0] - 2024-10-02

### Maintenance

- Updated coverage badge. (18 commits)

### Other

- Update main doc (cc7fa0e)
- Update .gitignore (b3a168c)
- Refactor project structure (6a5e303)
- Route handling and navigation in File Browser. (26aafd6)
- Update README.md (b86133e, 4d09ff4, 3f38861, e9d0a5d)
- Update dependencies (3c706cc)
- Split BrowserXrd vue on components (fc11578)
- Display Initial Path on file browser toolbar (b5cf5cf)
- Refactor BrowseXrdView.vue (1149ba1)
- Add npm workspace (14d56d7)
- Removed unused files (03b39e3, 23a80a7)
- Add script to generate test files (4d49417)
- Process directory listing in pages [GH-15] (5f8206c)
- Add Loading feedback [GH-17] (2b06ab5)
- Remove unused file (2dd3ff3)
- Fix release notes date. (d45bb15)
- Add unit-tests (7 commits)
- Improve code base and test coverege (2d64e06)
- Revise backend CI Actions (15ae21d, 8c1860f, 1859f67)
- Add unit tests (b712ff3, 5a185e5, 4b5c279)
- CI: update upload artifact action version (4ea270d)
- CI: Update action versions (1cad5cd)
- Improve naming in code (4fec6cf)
- Add unit-tests + bug fixes (4f8e5f7)
- Minor fix (af75d17)
- Update API doc (c6222fb)
- Fix error handling in tests (de02d3e)
- CI: Merge backend and lint actions (2bf3417)
- CI: Fix backend action name (087cb35)
- CI: Add Go Vet (281d00f)
- CI: minor fixes (4e086ff)
- Frontend: Use color consts (d69ba17)
- Minor docs update (1091bcf)
- Backend code cleanning (5b5959f)
- Fix lint error (c0989c4)
- Removed unused argument (2ce25b5)
- Add a back to top button. (c259828)
- Add a rout for Auth token (f092d80)
- Add Auth with JWT (897427b)
- Fix navbar burger menu (c554798)
- Prevent unauth users access the browse route (4db5388)
- Update frontend dependencies (655e02d)
- Set user name on login (90b379e)
- Update release notes (0270cc3)

## [0.4.0] - 2024-09-09

### Other

- Vue project skeleton (a52994a)
- Add the backend skeleton (e686bac)
- Update app skeleton (90ddd2a)
- WiP (c3752da, 2f9e654, 323c4b5)
- Update README.md (a97c97c, ee44fd9)
- Updated build instructions (4a81e87)
- Add ReleaseNotes (eed7d4f)
- Update Release notes format (ef7c7f0)
- Add xrd references (17ce3e6)
- Add backend with xrd client (b0c8b5d)
- Update release notes (2b048ab, 57988da, e8c85f7, 0c5d42d)
- Revamped frontend UI (8d49503)
- Most xrd settings to config (26c9d14)
- Add file download (7dabaca)
- Update docs (25d280d)
- Add sanitation job (a6457bc)
- Move sanitation code into a separate module (b67a4a9)
- Add missing implementation (de821f1)
- Simplify range expression (5035d12)
- Create go.yml (7864ae7)
- Update go.yml (e854a81)
- Add github actions (a8dc258)
- Update github actions (176f221, b1e76a6)
- Default wrk dir in Actions (4dad977)
- Actions: setup go cache issue (0d8d07b)
- Fix Navbar burger menu (d43c855)
- Minor Navbar changes (2640e22, 1b8afc6)
- Update Navbar (614ebc2)
- Add golangci-lint static checker (bac0be3)
- Adjust golangci-lint action (6970de2, 38d5700)
- Add first backend unit-test (3bb7c43)
- Call SanitationJob before the first tick (4179f04)
- Update .gitignore (c80a789)
- Update dependencies (92c9542)
- Minor fixes (a453c51)
- File names are clickable (ca3681b)
- Make breadcrumb path navi clickable (e450e12)
- Major refactor of the Browser view (d65a0c3)
- Refactored the Browse Files view. (b8e6c51)
- Show icons for files/folder. (ee49240)
- Dirs use bold font in Browser (1af7f30)
- Multiple cosmetic changes (31a9363)
- Fix style of Browser toolbar (26d60a8)
- Improve path element on Browser (a9b6035)
- Show server host on GUI (ef1326c)
- Fix file names with spaces (daf3290)
- Use session storage for File Browser states (89dabfc)
- Frontend containerization. (8796b4a)
- Containerization (adab47d)
- REST API doc (5e71608)
- Minor changes (34ce49b, 25a6d15)
- Update REST API doc (f03d8e9)
- Detailed error response. [GH-5] (281c599)
- pings the health status of backend [GH-7] (6d7b3e7)
- Add proper error logging. [GH-6] (4da972a)
- Update frontend dependencies (092548b)
- Update backend dependencies (43db7a2)
- Start using js plug-ins (f22f15e)
- Pretty format for file sizes. (5fa448d)
- Code and doc improvments (fe3c41d)
- Fix single quotes usage in Vue templates (ce74c9c, 99046a3)
- Use arrow functions for event handlers. (223c6ee)
- Use computed property for table data (fe3125e)
- File Browser view code minor fixes (c2fe133)
- Prevent adding failed dir to current path [GH-14] (2ee90c8)
- Add runtime config file. (07ccca1)
- Fix double dir load at startup (8bb7bee)
- Introduce project wide dev mode (b4d7810)
- Refactor the project source tree (9f8cf30)

