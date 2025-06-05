# DataHarbor

![CI](https://github.com/AnarManafov/dataharbor/actions/workflows/backend.yml/badge.svg)
![Coverage](https://img.shields.io/badge/Coverage-32.7%25-yellow)
![CI](https://github.com/AnarManafov/dataharbor/actions/workflows/frontend.yml/badge.svg)

DataHarbor provides a user interface to access GSI Lustre cluster data.  
It consists of the following main parts:

- [Frontend](./web/README.md)
- [Backend](./app/README.md)

## Features

- Secure, standards-based authentication using OpenID Connect (OIDC) and a Backend-For-Frontend (BFF) pattern
- User login and logout with strong session security
- Remotely browse and navigate directories and files on the GSI Lustre cluster
- View detailed file and directory properties (size, modification date, permissions, etc.)
- Select and securely download individual files

## Documentation

For detailed information about architecture, authentication, versioning, CI/CD workflows, and local development setup, see [DEVELOPMENT.md](./docs/DEVELOPMENT.md).

## Running DataHarbor Manually (Full Stack)

To run the complete DataHarbor application (backend and frontend) manually on a server, follow these steps:

### 1. Prerequisites

- **Go** (for backend)
- **npm** (for frontend)
- **xrootd client** (for backend, see [backend docs](./app/README.md))

### 2. Clone the Repository

```shell
git clone https://github.com/AnarManafov/dataharbor.git
cd dataharbor
```

### 3. Install Dependencies

- **Backend:**

  ```shell
  cd app
  go mod download
  cd ..
  ```

- **Frontend:**

  ```shell
  cd web
  npm install
  cd ..
  ```

### 4. Build the Backend

```shell
cd app
go build -o dataharbor-backend .
cd ..
```

### 5. (Optional) Configure the Backend

- Edit or provide a config file if needed (see [`app/config/application.template.yaml`](./app/config/application.template.yaml)).
- To use a custom config file, run the backend with `--config=<path-to-config.yaml>`.

### 6. Start the Backend

```shell
cd app
./dataharbor-backend --config=<path-to-config.yaml>  # or omit --config if not needed
cd ..
```

By default, the backend runs on `localhost:8081` (check your config for the actual port).

### 7. Start the Frontend

```shell
cd web
npm run build           # For production
npm run dev             # For development (hot reload)
cd ..
```

- For production, serve the built frontend with a web server (e.g., nginx, or use the container).
- For development, the frontend runs on `localhost:5173` by default.

### 8. Access the Application

- Open your browser and go to `http://localhost:5173` (or the port shown in the frontend output).
- The frontend will communicate with the backend as configured (see proxy settings in `web/vite.config.js` if needed).

### Notes

- Ensure both backend and frontend are running and accessible to each other (adjust firewall, ports, and proxy settings as needed).
- For HTTPS/local SSL, see [frontend README](./web/README.md) for certificate setup.
- For more advanced deployment (RPM, containers), see the respective [backend](./app/README.md) and [frontend](./web/README.md) READMEs.
