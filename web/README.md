# web (Frontend)

The frontend is built using [Vite](https://vitejs.dev/) and [Vue 3](https://vuejs.org/).

## Requirements

* npm  
  **OSX**

    ```shell
    brew install npm
    ```

* [Vite](https://vitejs.dev/guide/)

## Build/Run

* install dependencies

  ```shell
  npm install
  ```

* Compiles and hot-reloads for development

  ```shell
  npm run dev
  ```

* Compiles and minifies for production

  ```shell
  npm run build
  ```

* Run locally

  ```shell
  npm run preview
  ```

* Upgrade dependencies (when needed)

  ```shell
  npx npm-check-updates -u
  npm install
  ```

## Containerization

The frontend is containerized, described by the [Podman Container file](./Containerfile).  
The container includes nginx, therefore once started can be used out of the box.

### Build a Podman container

```shell
podman build -t data_lake_ui_backend:0.4.0 .
```

, where 0.0.4 is the version of the app. TODO: Need to automate that, by taking the version from the `git describe`, etc.

### Run a Podman container

```shell
podman run -p 8080:8080 data_lake_ui_frontend:0.4.0
```

The nginx of the container will be serving on port 8080, which can be changed if needed, of course.

## Customize configuration

See [Configuration Reference](https://cli.vuejs.org/config/).

## Links

* [Vite](https://vitejs.dev)
* [Install Vue](https://cli.vuejs.org/guide/installation.html)
* [Create new Project](https://cli.vuejs.org/guide/creating-a-project.html)
* [Serving Single-Page Application in a single binary file with Go](https://dev.to/aryaprakasa/serving-single-page-application-in-a-single-binary-file-with-go-12ij)
* [Go+Vue example](https://github.com/Simon-L/vue-go)
