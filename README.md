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

## Test locally

* Build and Start the frontend in a [preview mode](./web/README.md#run-locally).
* Then build and run [the backend app](./app/README.md#install-dependencies).
* Open a WEB Browser on `http://localhost:4173/`
