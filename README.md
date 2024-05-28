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
go get github.com/${YOUR_USERNAME}/app
```

### Install Dependencies

```shell
go mod download
```

## Development

### Vue Tips

- [Install Vue](https://cli.vuejs.org/guide/installation.html)
- [Create new Project](https://cli.vuejs.org/guide/creating-a-project.html)
