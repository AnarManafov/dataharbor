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

## Requirements

- Go  
  **OSX**

    ```shell
    brew install go
    ```

## Install Dependencies

```shell
cd app
go mod download
```

## Build

```shell
cd app
go build -o app .
```

## Run

```shell
 go run .
```

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

## Links
