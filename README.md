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

## Development

### Vue Tips

- [Install Vue](https://cli.vuejs.org/guide/installation.html)
- [Create new Project](https://cli.vuejs.org/guide/creating-a-project.html)
