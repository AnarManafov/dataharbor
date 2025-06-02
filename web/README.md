# Frontend

The frontend is built using [Vite](https://vitejs.dev/) and [Vue 3](https://vuejs.org/).

## Requirements

- npm  

    ```shell
    # OS X
    brew install npm

    # Windows
    winget install OpenJS.NodeJS.LTS
   ```

## Build/Run

- install dependencies

  ```shell
  npm install
  ```

- Compiles and hot-reloads for development

  ```shell
  npm run dev
  ```

- Compiles and minifies for production

  ```shell
  npm run build
  ```

- Upgrade dependencies (when needed)

  ```shell
  npx npm-check-updates -u
  npm install
  ```

## SSL Certificate Configuration

The development server uses HTTPS with self-signed certificates for testing. The certificate configuration automatically searches for certificates in multiple locations with the following priority:

1. **Environment variables** (highest priority):
   ```shell
   export VITE_SSL_KEY="/path/to/server.key"
   export VITE_SSL_CERT="/path/to/server.crt"
   ```

2. **PKM workspace** (for documentation/testing):
   - Relative path: `../../pkm/docs/gsi/dataharbor/test/cert/`
   - User home: `~/Documents/workspace/pkm/docs/gsi/dataharbor/test/cert/`

3. **Local fallback**: `../app/config/` (original location)

### Certificate Management

- **Check certificate status**:

  ```shell
  npm run cert:check
  ```

- **Use environment variables** (most portable):

  ```shell
  export VITE_SSL_KEY="/path/to/your/server.key"
  export VITE_SSL_CERT="/path/to/your/server.crt"
  npm run dev                  # Uses environment variables automatically
  npm run sandbox              # Uses environment variables automatically
  ```

- **Run with PKM certificates** (uses `$HOME` for cross-platform compatibility):

  ```shell
  npm run dev:pkm-certs        # Development with PKM certs
  npm run sandbox:pkm-certs    # Sandbox mode with PKM certs
  ```

- **Setup certificates**: Use `npm run cert:setup` or see `scripts/setup-certs-example.sh` for guidance
- If no certificates are found, the server will run in HTTP mode with a warning

The configuration is handled by `cert-config.js` which provides cross-platform compatibility for different PKM repository locations.

## Containerization

The frontend is containerized, described by the [Podman Container file](./Containerfile).  
The container includes nginx, therefore once started can be used out of the box.

### Build a Podman container

```shell
podman build -t dataharbor_frontend:0.4.0 .
```

, where 0.0.4 is the version of the app. TODO: Need to automate that, by taking the version from the `git describe`, etc.

### Run a Podman container

```shell
podman run -p 8080:8080 dataharbor_frontend:0.4.0
```

The nginx of the container will be serving on port 8080, which can be changed if needed, of course.

## Packaging

### RPM

- SPEC File: [dataharbor-frontend.spec](../packaging/dataharbor-frontend.spec)
- To build the package:

  ```shell
  # only for OS X 
  brew install rpm

  # Build command
  python3 packaging/build_rpm.py -b
  ```

- Package name: `dataharbor-frontend-<VERSION>-<RELEASE>.noarch.rpm`

The package will install:

- all frontend required files into `/usr/share/dataharbor-frontend`.
- [ngingx.conf](./nginx.conf) into `/etc/dataharbor-frontend/ngingx/ngingx.conf`

```shell
## Check locations of installed files
rpm -ql package_name
```

#### Configuration

Update the configuration file to include the path to the custom configuration file:

```shell
sudo vim /etc/nginx/nginx.conf
```

```Nginx
http {
    include /etc/nginx/sites-enabled/*;
    include /etc/nginx/sites-available/*;
}
```

Create a symbolic link to your custom configuration file in the `sites-available` and `sites-enabled`:

```shell
sudo ln -s /etc/dataharbor-frontend/nginx/nginx.conf /etc/nginx/sites-available/nginx.conf

sudo ln -s /etc/nginx/sites-available/nginx.conf /etc/nginx/sites-enabled/nginx.conf

# Restart NGINX to apply the changes:
sudo systemctl restart nginx
```

## Customize configuration

See [Configuration Reference](https://cli.vuejs.org/config/).

## Links

- [Vue SFC Playground](https://play.vuejs.org/)
- [Vite](https://vitejs.dev)
- [Install Vue](https://cli.vuejs.org/guide/installation.html)
- [Create new Project](https://cli.vuejs.org/guide/creating-a-project.html)
- [Serving Single-Page Application in a single binary file with Go](https://dev.to/aryaprakasa/serving-single-page-application-in-a-single-binary-file-with-go-12ij)
- [Go+Vue example](https://github.com/Simon-L/vue-go)
