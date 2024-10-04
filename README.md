![CI](https://github.com/AnarManafov/data_lake_ui/actions/workflows/backend.yml/badge.svg)
![Coverage](https://img.shields.io/badge/Coverage-83.9%25-brightgreen)
![CI](https://github.com/AnarManafov/data_lake_ui/actions/workflows/frontend.yml/badge.svg)

<!-- coverage-badge:begin -->
<!-- coverage-badge:end -->

# Data Lake UI


The Data Lake UI provides a user interface to access GSI Lustre cluster data.  
It consists of the following main parts:

* [Frontend](./web/README.md)
* [Backend](./app/README.md)

## Features

* Users can remotely browse directories and files.
* Directories and files are listed with their properties (size, modification date).
* Users can select and download individual files.

## Versioning

The project follows Semantic Versioning for versioning.  
Each release is tagged according to the following rules:

* **Backend Versions**: Use standard version tags `app/vX.Y.Z` (e.g., `app/v1.0.0`) to ensure compatibility with Go tools.
* **Frontend Versions**: Use a prefix `web/vX.Y.Z` (e.g., `web/v1.0.0`) to distinguish Frontend versions.
* **Global Versions**: Use a prefix `vX.Y.Z` (e.g., `v1.0.0`) to distinguish global versions.

### Release process

* Frontend:
  * Update the top level version in the [package.json](./web/package.json) to reflect the frontend version.
  * Update [RELEASE_NOTES](./web/RELEASE_NOTES.md)
* Backend:
  * * Update [RELEASE_NOTES](./app/RELEASE_NOTES.md)
* Global:
  * Update the top level version in the [package.json](./package.json) to reflect the global versions number.
  * Update [RELEASE_NOTES](./RELEASE_NOTES.md)
* Apply git tags according to [Versioning rules](#versioning)

## How to run locally

This setup ensures that both the Vue 3 Frontend and Go Backend are running simultaneously in development mode, making development more efficient.

### Install dependencies

```shell
npm install
```

Since the project defines npm workspaces, this command can be run from the root folder of the project or from the Frontend subdirectory `./web`.

### Start Development Servers

To start both the Frontend and Backend servers in development mode (with a hot-reload support), run:

```shell
npm run dev
```

This will start the Vue Frontend server and the Go Backend server concurrently.  
The Frontend will be running on [localhost:5173](http://localhost:5173/) (or on the port specified by Vite).

### Configuration

If you need to provide an optional configuration file path to the Go Backend, set the `CONFIG_FILE_PATH` environment variable before running the npm run dev command:

```shell
CONFIG_FILE_PATH=/path/to/config.yaml npm run dev
```

### Build the Project

To build both the Frontend and Backend, run:

```shell
npm run build
```

This will build the Vue Frontend using Vite and compile the Go Backend.

## Containerization

The project can provide Two Containers ([One for Frontend](./web/README.md#containerization), [One for Backend](./app/README.md#containerization)). The Backend can be treated as a microservice.  The Backend is relatively compact, therefore container is not really needed, as it can be deployed as an app. But splitting Frontend and Backend deployment is still advisable, see below.

* **Isolation**:
Each service runs in its own container, which means they are isolated from each other. This can make debugging easier and improve security.
* **Scalability**:  
You can scale the Frontend and Backend independently based on their respective loads. For example, if your Backend needs more resources, you can scale it up without affecting the Frontend.
* **Flexibility**:  
You can update or redeploy one service without affecting the other. This is particularly useful for continuous deployment and integration.
* **Best Practices**:  
This approach aligns with the microservices architecture, which is a widely adopted best practice in modern application development.
