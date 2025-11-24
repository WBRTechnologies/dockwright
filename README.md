# Dockwright Deployer

## Description

Dockwright is an opinionated deployment tool that builds a Docker image from your service, pushes it to your private registry, and deploys it using Helm. The tool ships with pre-built Helm base charts for different microservice flavours (such as stateless and stateful), and automatically selects the correct chart based on the service’s declared flavour. All base charts are packaged locally during installation, ensuring predictable, repeatable deployments across development and production environments.

---

## Why This Exists

- Establish consistent, production-grade deployment standards across all services. 
- Centralize shared Helm templates, helpers, and deployment logic in one place.
- Improve onboarding speed, maintainability, and operational consistency, with automatic base-chart selection driven by each service’s flavour.
- Eliminate duplication and drift across microservice repositories.

---

## Prerequisites

### 1. Environment Variables

Set these before running the deployer:

| Variable | Description |
|---------|-------------|
| `REGISTRY_HOST` | Registry endpoint, e.g. `registry.example.com` |
| `REGISTRY_NAMESPACE` | Registry org / namespace |
| `REGISTRY_REPOSITORY` | Repository group for service images |
| `REGISTRY_USERNAME` | Registry username |
| `REGISTRY_PASSWORD` | Registry password or token |

### 2. Required CLI Tools

- `docker`
- `helm`
- `awk`

### 3. Required Repository Structure

```
your-service/
├── Dockerfile
└── helm/
    ├── config.yaml     # contains: flavour: stateless | stateful
    └── values.yaml     # service-specific Helm overrides
```

---

## Installation

From the root of the Dockwright project:

```sh
sudo make install
```

This performs:

1. Packages every chart under `helm-base-charts/*`
2. Writes `.tgz` packages into `/usr/local/share/dockwright/charts`
3. Installs:
    - `deployer.sh` → `/usr/local/bin/dockwright`
    - `dockwright.conf` → `/etc/dockwright.conf`

After installation:

```sh
which dockwright
# /usr/local/bin/dockwright
```

---

## Usage

Inside any microservice directory containing a `Dockerfile` and `helm/` folder:

```sh
dockwright
```

This will:

1. Build the Docker image and push it to your configured registry
2. Read `flavour` from `helm/config.yaml`
3. Select the matching base chart from `/usr/local/share/dockwright/charts`
4. Run:

   ```sh
   helm upgrade --install ...
   ```

Deployments always use the same packaged charts, ensuring deterministic behavior across environments.

---

### Updating Base Charts

If you modify or add charts inside `helm-base-charts/`, re-run:

```sh
sudo make package-charts
```

This regenerates all chart `.tgz` files into `/usr/local/share/dockwright/charts`.

---

## Uninstallation

```sh
sudo make uninstall
```

This removes:

```
/usr/local/bin/dockwright
/etc/dockwright.conf
/usr/local/share/dockwright/charts
```
