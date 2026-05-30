# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.20] - 2026-04-23

### Fixed

- fix TLS handshake failure and XRootD hostname warning (backend,docker) (c9171af)

## [1.0.19] - 2026-04-22

### Added

- update branding (8972706)
- add multi-file batch download as tar.gz archive (backend,frontend) [GH-11] (f1392d9)

### Fixed

- update help section links for issue reporting (ec93b1f)

### Maintenance

- remove GSI local environment configuration (1931b94)
- add CODEOWNERS file for repository ownership (d28785b)

### Build

- update dependencies for web package (deps) (710a2a0)
- update base images and dependencies (docker) [GH-11] (33fea7a)

### CI

- add SSH key for direct master branch deployment (ef3061a)

## [1.0.18] - 2026-03-31

### Added

- unify logging to stdout-only with JSON format and rotation (docker) [GH-45] (e214807)

### Fixed

- update healthcheck command for XRootD service (docker) (1a7861f)
- update health check for XRootD service (docker) [GH-46] (aa1b158)
- preserve directory listing on access denied navigation (frontend) [GH-44] (53e9572)

### Maintenance

- bump XRootD to 5.9.2 and add PB tier to storage formatter (2af2a8b)
- add Makefile, update docs, and clean up tooling (9ca954c)
- Updated coverage badge. (be66a9d)

## [1.0.17] - 2026-03-28

### Fixed

- mount host SSSD socket for LDAP/AD user resolution in XRootD container (docker) (958aad8)

## [1.0.16] - 2026-03-28

### Fixed

- update xrootd user/group creation instructions (docker) (f988351)

## [1.0.15] - 2026-03-28

### Fixed

- fix XRootD multiuser plugin failing to resolve mapped usernames (docker) (b1a2f55)

## [1.0.14] - 2026-03-28

### Fixed

- ensure xrootd user/group creation in Dockerfile (docker) (4e3a6a9)

## [1.0.13] - 2026-03-28

### Fixed

- update XRootD version and TLS configuration (docker) (a25673d)

## [1.0.12] - 2026-03-28

### Maintenance

- disable RPM packaging in workflows and update docs (82c456f)

## [1.0.11] - 2026-03-28

### Fixed

- update SciTokens logging level (xrootd-prod.cfg) (95e59aa)

## [1.0.10] - 2026-03-28

### Fixed

- update security library path (xrootd-prod.cfg) (02db4b5)

## [1.0.9] - 2026-03-28

### Added

- centralize session cookie options for security (auth) (f12b6c5)
- enhance token and user info caching mechanisms (auth) (40fd00c)

## [1.0.8] - 2026-03-28

### Added

- add XRD ping endpoint and network stats tracking (backend,frontend) (f808f9e)

## [1.0.7] - 2026-03-28

### Added

- add XRootD virtual filesystem statistics (backend,frontend) (478a1e0)

## [1.0.6] - 2026-03-27

### Changed

- reorganize TLS and security settings (xrootd-prod.cfg) (7460a6e)

## [1.0.5] - 2026-03-27

### Fixed

- adjust SameSite cookie settings for OIDC flows (auth) (63b34e0)

### Maintenance

- Updated coverage badge. (3cfed60)

## [1.0.4] - 2026-03-26

### Added

- update backend Dockerfile and entrypoint script (docker) (7b90751)

## [1.0.3] - 2026-03-26

### Fixed

- use configured frontend URL for OIDC redirect URI and improve Docker volume mounts (backend,docker) (e154cbc)

## [1.0.2] - 2026-03-25

### Added

- add cleanup option for unreferenced layers (build-docker) (96c7267)
- use devcontainer features rather than Go base image (devcontainer) (c421fbb)
- enhance feature descriptions and icons (HomeView) (b3119d8)
- update icon sizes for better visibility (HomeView) (159854b)
- add workspace mount for git worktrees (devcontainer) (06c5b1f)
- enhance deployment configuration and health checks (docker) (12b8811)

### Changed

- streamline image handling functions (build-docker) (a9d828b)

### Documentation

- add coverage report commands to backend guide (06c9088)

### Maintenance

- Updated coverage badge. (6 commits)
- remove unused CMake and Go test files (613ba7e)
- remove sandbox-related scripts and configurations (bc920a7)
- modernize Go idioms and update base image versions (a016edb)

### Tests

- add comprehensive tests for middleware and requests [GH-42] (f32b8b5)

### Build

- update package versions (dependencies) (02d24de)
- update dependencies (dependencies) (96e1930)
- update package versions for stability (dependencies) (1202c30)
- update dependencies for improved stability (deps) (5828b4f)

### CI

- update GitHub Actions to use latest action versions (98ddf07)

## [1.0.1] - 2025-12-08

### Fixed

- resolve all CI/CD workflow issues (5f1c9ab)

## [1.0.0] - 2025-12-08

### Added

- add Docker Compose deployment and ZTN/TLS XRootD support (39dbb41)
- add Dockerfile and configuration files (devcontainer) (ea18af2)
- add Go configuration settings (vscode) (901d1fb)
- add comprehensive configuration tests (tests) (48ee3f0)

### Fixed

- update dependencies to latest versions (go.mod) (399b193)

### Maintenance

- Updated coverage badge. (5da6495, d3f4d3f, 1367f03, 8adc71a)
- update VSCode settings exclusion (.gitignore) (d735405)

### Build

- add dos2unix for line ending normalization (docker) (490a814)

### Other

- feat(docker)!: add production Docker deployment (406555a)

## [0.15.0] - 2025-10-16

### Added

- add ZTN protocol support for OAuth token authentication (xrootd) (039cdb8)
- enhance ZTN protocol configuration guide (docs) (1b06f18)
- Update RPM packaging for DataHarbor backend and frontend (packaging) (80a3c3e)

### Fixed

- handle error when setting BEARER_TOKEN (xrd) (f5c4ced)

### Documentation

- update README for improved clarity and formatting (f29d4ea)

### Maintenance

- Updated coverage badge. (fa732b9, be85bdd, f5dd8c6, df796d3)

### Style

- reorder XRDConfig fields for consistency (tests) (6f82807)

### CI

- update workflow triggers for packaging changes (cda599e)

### Other

- Add GSI Deployment Guide for DataHarbor (1d9e201)
- Update hep fork dependency with TLS/ZTN connection logging (c1c2a57)

## [0.14.6] - 2025-10-08

### Added

- Add multi-architecture support for builds (ci) (1997c79)

### Maintenance

- Updated coverage badge. (0967c7f, 6808352)

### Other

- doc: improve documentation (35345a8)

## [0.14.5] - 2025-08-11

### Fixed

- Update sync-versions script and CI workflow to handle package-lock.json files (ci) (863a3e2)

### Maintenance

- Update changelog header and generation logic (5fa2386)
- update dependency management instructions (docs) (d7aed72)

## [0.14.4] - 2025-08-07

### Maintenance

- clean up release notes formatting (60ab64a)

## [0.14.3] - 2025-08-07

### Maintenance

- update changelog formatting and release notes output (d7d9746)

## [0.14.2] - 2025-08-07

### Added

- enhance changelog and release notes generation (ci) (2b753b7)

### Maintenance

- update dependencies (47781c1)
- Updated coverage badge. (859f1b0)
- update release workflows (96b3c4c)

## [0.14.1] - 2025-08-07

### Maintenance

- update dependencies (47781c1)
- Updated coverage badge. (859f1b0)
- update release workflows (96b3c4c)

## [0.14.0] - 2025-08-05

### Added

- enhance user authentication display in navbar (nav) (5ddf6ae)
- backend to support HTTPS for Keycloak [GH-34] (fe7681f)
- enhance directory listing response structure (api) (629c86b)
- native XRD client with streaming downloads [GH-10] (0724747)

### Changed

- migrate from viper to config package (config) (a2d64dc)

### Documentation

- Update README and add detailed dev doc (c527d88)

### Maintenance

- Rename from data-lake-ui to dataharbor [GH-36] (7130c50)
- Updated coverage badge. (16941f3, 86756d6, 9f860ff, 9624b92, af03d43)
- Update dependencies for improved stability (f659980)

### Style

- Adjust component sizes and spacing (4a424a7)
- update font sizes and remove Bulma dependency (e9a865f)
- unify typography across components and styles (3ab3273)
- enhance layout and structure of file browser (8014eca)
- adjust sidebar width and improve table sorting (4a233cc)

### Build

- update vue and babel dependencies to latest (deps) (477cf89)

### Other

- Refactor views for improved layout and styling (9a12373)
- doc: Refactor project documentation (645c9c7)

## [0.13.13] - 2025-05-19

### Added

- enhance logout process for improved security (auth) [GH-27] (ffbfd02)

### Documentation

- update changelog and release notes for v0.13.12 (594e821)

### Maintenance

- update dependencies in package.json (87d57c7)
- Updated coverage badge. (834f420, 1b866ed)
- update npm scripts for cross-platform (a54f1ca)

## [0.13.12] - 2025-05-05

### Added

- enhance RPM build process, CI workflows, and release notes generation (build) [GH-26] (38b3074)
- switch to Python script for changelog generation (changelog) (762d352)
- automate CHANGELOG and RELEASE_NOTES updates (changelog) (836b9da)

### Documentation

- Update release notes for v0.13.9 [skip ci] (36560d7)
- Update changelog for v0.13.10 [skip ci] (e85f2ec)

### Maintenance

- Update package versions to v0.13.8 [skip ci] (158e81d)
- refactor version tag processing jobs (workflow) (843e1d5)

## [0.13.11] - 2025-05-05

### Maintenance

- refactor version tag processing jobs (workflow) (2ecbf86)

## [0.13.10] - 2025-05-05

### Added

- automate CHANGELOG and RELEASE_NOTES updates (changelog) (51ce2f1)

## [0.13.9] - 2025-05-05

### Added

- enhance RPM build process, CI workflows, and release notes generation (build) [GH-26] (d999931)
- switch to Python script for changelog generation (changelog) (7fdf81a)

### Documentation

- Update release notes for v0.13.7 [skip ci] (be83f32)

### Maintenance

- Update package versions to v0.13.7 [skip ci] (63e7b58)

## [0.13.7] - 2025-04-25

### Fixed

- update job dependencies and output delimiters (workflow) (8022ce2)

### Documentation

- Update release notes for v0.13.6 [skip ci] (dd8deac)

### Maintenance

- Update package versions to v0.13.6 [skip ci] (bb4692c)

## [0.13.6] - 2025-04-25

### Documentation

- Update release notes for v0.13.5 [skip ci] (554bb54)

### Maintenance

- Update package versions to v0.13.5 [skip ci] (33eca89)
- remove branch restriction for tag processing (workflow) (28a0846)

## [0.13.5] - 2025-04-25

### Maintenance

- refactor version tag processing and permissions (workflow) (ea06434)

## [0.13.4] - 2025-04-25

### Maintenance

- update permissions and token usage for tag creation (workflow) (38e9bb4)

## [0.13.3] - 2025-04-25

### Maintenance

- enhance permissions for tag creation (workflow) (d3b7e96)

## [0.13.2] - 2025-04-25

### Maintenance

- explicitly checkout and push to master branch (workflow) (cba1d9a)

## [0.13.1] - 2025-04-25

### Maintenance

- Update version to v0.13.0 in package.json files (441893f)
- Update release notes for v0.13.0 (466d569)
- update CI workflows for versioning (workflow) (b5e77f8)
- Updated coverage badge. (b5db503)
- refine CI workflows for consistency (workflow) (2051166)
- standardize quotes in version tag processor (workflow) (ac63deb)

## [0.13.0] - 2025-04-24

### Maintenance

- improve git pull process in auto-version (workflow) (dbbe468)

## [0.12.0] - 2025-04-24

### Maintenance

- Update version to v0.11.0 in package.json files (f144fa6)
- improve versioning and release notes process (workflow) (665e7e6)

## [0.11.0] - 2025-04-24

### Maintenance

- Update version to v0.10.0 and release notes (89c6771)
- update auto-version and frontend workflows (workflow) (8ea5320)

## [0.10.0] - 2025-04-24

### Added

- update changelog generation process (build) (2c8480e)

### Maintenance

- Update version to v0.9.0 and release notes (059803c)

## [0.9.0] - 2025-04-24

### Added

- enhance RPM packaging process and changelog generation (build) (2045756)

### Maintenance

- Update version to v0.8.0 and release notes (0d80f89)

## [0.8.0] - 2025-04-24

### Added

- streamline auto versioning and release process (ci) (1403f64)

### Maintenance

- Update version to v0.7.0 and release notes (957b72b)

## [0.7.0] - 2025-04-24

### Added

- enhance publish release workflow (ci) (8db8576)

### Maintenance

- Update version to v0.6.0 and release notes (eae32b0)

## [0.6.0] - 2025-04-24

### Added

- enhance auth UI and logout handling (Nav) [GH-19] (475779c)
- Add release publishing workflow & version management (22f68c6)

### Maintenance

- Updated coverage badge. (18 commits)

### Other

- RPM packaging for backend [GH-20] (50d5554)
- Add debug log to RPM build (c953cb1)
- Fix backend CI (17086a6)
- CI fix for RPM artifacts (0d00654)
- RPM packaging for frontend [GH-21] (74e4081)
- RPM: generate change log at build time (701c33f)
- RPM changelog fix (f7558b5)
- RPM packaging improvment (714914a)
- Fix backend packaging (67c0721)
- Update backend dependencies (ca92af3)
- Add ngingx for RPM spec of frontend (eae6c1e)
- Fix RPM build with nginx (f90bb00, fb1b866)
- Fix nginx package conflict. (c4ac430)
- Update documentation (5e55c76)
- backend RPM to use host arch as target (cfb7fb5)
- Update dependencies (89e995b, 50b4a9d, 9b3df4f, 01f411a)
- Update frontend package (026a2cc)
- Fix frontend packaging (06fe279)
- Relocate asset files to work on prod and dev env (ab8badd)
- update dependencies (0eebaaf)
- Update doc (5f566a9)
- Improve UI components and error handling (22b74c3)
- Implement Keycloak OIDC authentication [GH-19] (07aaa19)

## [0.5.0] - 2024-10-02

### Maintenance

- Updated coverage badge. (18 commits)

### Other

- Update main doc (fc39ccd)
- Update .gitignore (3544ffa)
- Refactor project structure (8eba04f)
- Route handling and navigation in File Browser. (4d51fe6)
- Update README.md (bb50bbb, 2b38cb3, 03040b0, d550f62)
- Update dependencies (b2ad280)
- Split BrowserXrd vue on components (6a852bf)
- Display Initial Path on file browser toolbar (02f654b)
- Refactor BrowseXrdView.vue (dd8fca6)
- Add npm workspace (53351d0)
- Removed unused files (e68ffcd, ae19273)
- Add script to generate test files (1511aa1)
- Process directory listing in pages [GH-15] (a52182a)
- Add Loading feedback [GH-17] (d249c03)
- Remove unused file (712b888)
- Fix release notes date. (40bf012)
- Add unit-tests (7 commits)
- Improve code base and test coverege (fe41655)
- Revise backend CI Actions (7f05dbd, 35565cc, bd64519)
- Add unit tests (f63c46a, 9c343be, 2e5bdb8)
- CI: update upload artifact action version (7afa17f)
- CI: Update action versions (2c83839)
- Improve naming in code (75e126c)
- Add unit-tests + bug fixes (3adac4c)
- Minor fix (d50b1ac)
- Update API doc (144be35)
- Fix error handling in tests (f424cda)
- CI: Merge backend and lint actions (ab8396b)
- CI: Fix backend action name (9814117)
- CI: Add Go Vet (494b4d8)
- CI: minor fixes (6001c87)
- Frontend: Use color consts (5c1731a)
- Minor docs update (bcd2adc)
- Backend code cleanning (807ec37)
- Fix lint error (a513a8e)
- Removed unused argument (503d3f6)
- Add a back to top button. (6ee2576)
- Add a rout for Auth token (48e1f1c)
- Add Auth with JWT (a736ab1)
- Fix navbar burger menu (bee23f5)
- Prevent unauth users access the browse route (22a9994)
- Update frontend dependencies (5fd6ed9)
- Set user name on login (15f4225)
- Update release notes (63e64b9)

## [0.4.0] - 2024-09-09

### Other

- Vue project skeleton (ddcfa09)
- Add the backend skeleton (c3863ce)
- Update app skeleton (c5fd75a)
- WiP (64c1fc5, e8e1217, 970df49)
- Update README.md (350e75c, 26a04c7)
- Updated build instructions (9def7a1)
- Add ReleaseNotes (3724679)
- Update Release notes format (bef59a0)
- Add xrd references (4ce453c)
- Add backend with xrd client (fbb1396)
- Update release notes (d54e3c4, 2ed63c6, f037b19, 014c5b5)
- Revamped frontend UI (20aff9b)
- Most xrd settings to config (d743941)
- Add file download (5ca69bd)
- Update docs (40a6446)
- Add sanitation job (a0e1efb)
- Move sanitation code into a separate module (1e48970)
- Add missing implementation (8667248)
- Simplify range expression (90ccac9)
- Create go.yml (e360ed9)
- Update go.yml (954278e)
- Add github actions (584f24f)
- Update github actions (74b4149, bf8bc61)
- Default wrk dir in Actions (7aaf575)
- Actions: setup go cache issue (92d3ab4)
- Fix Navbar burger menu (d673c62)
- Minor Navbar changes (988e095, 08e3311)
- Update Navbar (21e3abc)
- Add golangci-lint static checker (d1d1989)
- Adjust golangci-lint action (92e673a, c751610)
- Add first backend unit-test (35e4371)
- Call SanitationJob before the first tick (a41ac5d)
- Update .gitignore (4b90153)
- Update dependencies (5956b44)
- Minor fixes (529f840)
- File names are clickable (df87df1)
- Make breadcrumb path navi clickable (475eaa5)
- Major refactor of the Browser view (0457bc3)
- Refactored the Browse Files view. (2e3b91c)
- Show icons for files/folder. (eb33ea4)
- Dirs use bold font in Browser (b5c630b)
- Multiple cosmetic changes (b40eb8d)
- Fix style of Browser toolbar (f87e545)
- Improve path element on Browser (02ea02a)
- Show server host on GUI (e3c1e0f)
- Fix file names with spaces (a0dfa5b)
- Use session storage for File Browser states (4829e9b)
- Frontend containerization. (534cec7)
- Containerization (afd2f63)
- REST API doc (0f30113)
- Minor changes (cb66fb0, a2755d3)
- Update REST API doc (76ebbe7)
- Detailed error response. [GH-5] (c24eb53)
- pings the health status of backend [GH-7] (b6f3485)
- Add proper error logging. [GH-6] (a13e8f9)
- Update frontend dependencies (32a94d2)
- Update backend dependencies (fe8f871)
- Start using js plug-ins (3b97be7)
- Pretty format for file sizes. (e356585)
- Code and doc improvments (919992d)
- Fix single quotes usage in Vue templates (89d9482, 7f4201b)
- Use arrow functions for event handlers. (608fa4d)
- Use computed property for table data (f60da9f)
- File Browser view code minor fixes (386e326)
- Prevent adding failed dir to current path [GH-14] (8acf7f8)
- Add runtime config file. (3048990)
- Fix double dir load at startup (833134e)
- Introduce project wide dev mode (ea89178)
- Refactor the project source tree (ae85e9b)

