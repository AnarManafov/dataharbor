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

## Test locally

* Build and Start the frontend in a [preview mode](./web/README.md#run-locally).
* Then build and run [the backend app](./app/README.md#install-dependencies).
* Open a WEB Browser on `http://localhost:4173/`

