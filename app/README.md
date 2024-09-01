# Backend

The backend is written in Go and provides an interface to XROOTD-related operations.  
At first the plan was to use XRD API directly. Unfortunately there is only XRD-client C++ API, which is difficult to support in Go. Sometime is it also unreliable even via a [SWIG wrapper](https://www.swig.org/Doc3.0/Go.html).  

For now it was decided to use xrd-client command line calls to cover all XRD-related operations. All command line calls are covered with an async. timeout to prevent blocking of the entire app.  

The backend app is running a lightweight, local WEB server and responds on requests send by the frontend (a Vue SPA).  
The server port is configurable, see [the app's configuration](./config/application.template.yaml).

## Features

- List files using the selected XROOTD server.
- Can stage a requested file for download. It copies the file to a WEB server's public location, into a staged temporary directory, for further download.
- Implements a sanitation job to periodically check and clean staged temporary files.

## API

### Files and Directories

- [Initial directory](./doc/api/initial_dir.md) : `GET /initial_dir`
- [List a directory](./doc/api/dir.md): `POST /dir`
- [Stage a file. Prepare for download.](./doc/api/stage_file.md): `POST /stage_file`

### General Details

- [Host Info](./doc/api/host_name.md): `GET /host_name`
- [Service's health](./doc/api/health.md): `GET /health`

## Requirements

- [Go](https://go.dev/)
  - OSX

    ```shell
    brew install go
    ```

### Build the project

- Install Dependencies

  ```shell
  cd app
  go mod download
  ```

- Build

  ```shell
  cd app
  go build -o app .
  ```

- Run

  ```shell
   go run .
  ```

  Run with the app's config file

  ```shell
  go run . --config=<the_config_file_name>
  ```

  or just start an executable.

## Containerization

The backend is not that big - just one executable and one configuration file, for now.  
It might be an overkill to containerize it. But just in case it is needed, a [Podman Container file](./Containerfile) is in the app directory and below are some instructions.

### Build a Podman container

```shell
podman build -t data_lake_ui_frontend:0.0.4 .
```

,where 0.0.4 is the version of the app. TODO: Need to automate that, by taking the version from the `git describe`, etc.

### Run a Podman container

```shell
podman run --network=host data_lake_ui_backend:0.0.4
```

### Known issues

#### An xrootd client bin directory

An xrootd client bin directory needs to be exposed to the container.

This link helps with some Podman on OSX issues: <https://github.com/ansible/vscode-ansible/wiki/macos>

## Dev Tips

### Initialize Go

The following command will generate a `go.mod` file.

```shell
cd app
go mod init github.com/${YOUR_USERNAME}/app
```

### Register missing dependencies

```shell
cd app
go get github.com/${YOUR_USERNAME}/app
```

### Add new module requirements and sums

```shell
cd app
go mod tidy
```
