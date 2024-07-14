# Backend

## Requirements

* Go
  * OSX

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
