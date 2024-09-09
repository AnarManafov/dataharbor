# Frontend Release Notes

## [0.4.0] - NOT YET RELEASED

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