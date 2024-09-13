# Frontend Release Notes

## [0.5.0] - NOT YET RELEASED

- Modified: Split `BrowseXrdView.vue` into multiple components.
- Modified: Refactored the toolbar component of the File Browser.
- Added: Enhanced route handling to prevent redundant navigation and ensure proper directory loading.
- Added: Users can now navigate in the File Browser via the web browser URL bar and navigation buttons.
- Added: Display of folder and file counts, as well as cumulative file size in the toolbar.
- Added: Support pagination for directory listings.

## [0.4.0] - 2024-09-09

- Modified: Refactored the Browse Files view.
- Added: Users can now download the selected file.
- Added: Show corresponding icons for files and folders in the File Browser.
- Added: Use session storage to save the state of the File Browser.
- Added: Containerization of the frontend with an Nginx server.
- Added: Periodic backend service health checks and corresponding UI updates based on service status. (GH-7)
- Added: Clean the table and highlight the home icon in red if the backend appears offline. (GH-7)
- Added: Error handling of protocol errors. (GH-6)
- Added: A filter to convert file size to a human-readable format with kBytes, MB, GB, etc.
- Added: Runtime config file support.
- Fixed: If the user can't enter the directory, it should not be added to the current path. (GH-14)
