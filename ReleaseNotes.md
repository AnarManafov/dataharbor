# Data Lake UI Release Notes

## v0.4 NOT YET RELEASED

### Frontend

- Modified: Refactored the Browse Files view.
- Added: Users are now able to download the selected file.
- Added: Show corresponding icons for files and folder in the File Browser.
- Added: Use a session storage to save the state of the File Browser.
- Added: Containerization of the frontend with an nginx server.
- Added: Periodically check the status of the Backend. (GH-7)
- Added: Clean the table and highlight the home icon in red if backed appears offline. (GH-7)
- Added: Error handling of protocol errors. (GH-6)
- Added: A filter tp convert file size to a human readable format with kBytes, MB, GB, etc.
- Added: A runtime config file support.
- Fixed: If the user can't enter the directory, it should not be added to the current path. (GH-14)

### Backend

- Fixed: pull file names with with white spaces.
- Added: sanitation job to periodically check and clean staged temporary files.
- Added: XRD settings moved to the backend configuration.
- Added: detailed error response. (GH-5)
- Added: REST API documentation.
- Added: use js plugins in order to improve the code.

## v0.3 (2024-08-18)

- Revamped frontend UI (add new home view, nav bar, etc.)
- Removed unused code.
- Added file size and file date to the xrd view (updated front-/backend).

## v0.2 (2024-08-17)

- The backend is to use XRD client.

## v0.1 (2024-07-14)

- Initial version.
- Working skeletons of a Frontend and a Backend.
- It can browse files in the local home folder of the server user.
