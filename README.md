# Data Lake UI

The Data Lake UI provides a user interface to access GSI Lustre cluster data.  
It consists of the following main parts:

* [Frontend](./web/README.md)
* [Backend](./app/README.md)
  * [xrootd](./doc/xrootd.md)

## Features

* Users can remotely browse directories and files.
* Directories and files are listed with their properties (size, modification date).
* Users can select and download individual files.

## Versioning

The project follows Semantic Versioning for versioning.  
Each release is tagged according to the following rules:

* **Backend Versions**: Use standard version tags (e.g., `v1.0.0`) to ensure compatibility with Go tools.
* **Frontend Versions**: Use a prefix (e.g., `frontend-v1.0.0`) to distinguish frontend versions.
* **Global Versions**: Use a prefix (e.g., `global-v1.0.0`) to distinguish global versions.

## How to run locally

This setup ensures that both the Vue 3 frontend and Go backend are running simultaneously in development mode, making development more efficient.

### Start Development Servers

To start both the frontend and backend servers in development mode (with a hot-reload support), run:

```shell
npm run dev
```

This will start the Vue frontend server and the Go backend server concurrently.  
The frontend will be running on [Local](http://localhost:5173/) (or on the port specified by Vite).

### Configuration

If you need to provide an optional configuration file path to the Go backend, set the `CONFIG_FILE_PATH` environment variable before running the npm run dev command:

```shell
CONFIG_FILE_PATH=/path/to/config.yaml npm run dev
```

### Build the Project

To build both the frontend and backend, run:

```shell
npm run dev
```

This will build the Vue frontend using Vite and compile the Go backend.

## Containerization

The project can provide Two Containers ([One for Frontend](./web/README.md#containerization), [One for Backend](./app/README.md#containerization)). The backend can be treated as a microservice.  The backend is relatively compact, therefore container is not really needed, as it can be deployed as an app. But splitting frontend and backend deployment is still advisable, see below.

* **Isolation**:
Each service runs in its own container, which means they are isolated from each other. This can make debugging easier and improve security.
* **Scalability**:  
You can scale the frontend and backend independently based on their respective loads. For example, if your backend needs more resources, you can scale it up without affecting the frontend.
* **Flexibility**:  
You can update or redeploy one service without affecting the other. This is particularly useful for continuous deployment and integration.
* **Best Practices**:  
This approach aligns with the microservices architecture, which is a widely adopted best practice in modern application development.
