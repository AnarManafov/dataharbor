# data_lake_ui

## Frontend

### Production Build

```shell
cd web
npm run build
```

### Local run

```shell
cd web
npm run serve
```

### Backend

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

### Install Dependencies

```shell
cd app
go mod download
```

### Build

```shell
cd app
go build -o app .
```

### Run

```shell
 go run .
```

## Development

### Tips and Links

- [Install Vue](https://cli.vuejs.org/guide/installation.html)
- [Create new Project](https://cli.vuejs.org/guide/creating-a-project.html)
- [Serving Single-Page Application in a single binary file with Go](https://dev.to/aryaprakasa/serving-single-page-application-in-a-single-binary-file-with-go-12ij)
- [Go+Vue eample](https://github.com/Simon-L/vue-go)
- [Project Template example1](https://github.com/tdewolff/go-vue-template)
- [Project Template example2](https://github.com/dkfbasel/scratch)
