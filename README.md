# Dockwright Deployer

## Description

Dockwright is an opinionated deployment tool that builds a Docker image from your service, pushes it to your private registry, and deploys it using Helm. The tool ships with pre-built Helm base charts for different microservice flavours (such as stateless and stateful), and automatically selects the correct chart based on the service's declared flavour. All base charts are copied locally during installation, ensuring consistent deployments across development and production environments.

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
| `REGISTRY_USERNAME` | Registry username |
| `REGISTRY_PASSWORD` | Registry password or token |

**Note:** These environment variables are required when building and pushing Docker images. If you use `--docker-build=false`, they are not needed. The image repository is constructed from these values and injected into your Helm deployment.

### 2. Required CLI Tools

- `go`
- `docker` (with running daemon)
- `helm`

## Installation

From the root of the Dockwright project:

```sh
sudo make install
```

This performs:

1. Copies every chart under `base-helm-charts/*` directly to `/usr/local/share/dockwright/charts`
2. Installs:
    - `dockwright` → `/usr/local/bin/dockwright`

After installation:

```sh
which dockwright
# /usr/local/bin/dockwright
```

---

## Usage

Inside any microservice directory:

```sh
dockwright deploy
```

This will:

1. **Load Configuration**: Read from CLI flags, `.dockwright/config.yaml`, and environment variables
2. **Validate Prerequisites**: Check required tools, Kubernetes context, environment files, and configuration
3. **Docker Workflow**: Build and push the Docker image (unless `--docker-build=false`)
4. **Helm Workflow**: Deploy using the appropriate base chart with collected values files

### Repository Structure

```
your-service/
├── Dockerfile                        # Optional
└── .dockwright/
    ├── config.yaml                   # Dockwright configuration
    └── helm/
        ├── values.yaml               # Base Helm values (optional)
        ├── staging.values.yaml       # Environment-specific values
        └── production.values.yaml    # Environment-specific values
```

### Configuration File

Create a `.dockwright/config.yaml` file to set default values (can be overridden by CLI flags):

```yaml
artifactName: my-service
helm:
  flavour: stateless    # or 'stateful'
docker:
  namespace: my-org
  host: registry.example.com
  build: true
kubernetes:
  config: ~/.kube/config
  context: my-cluster
env:
  - staging
  - production
dry-run: false
auto-approve: false
```

### CLI Flags

| Flag | Description | Default |
|------|-------------|--------|
| `--artifact-name` | Name of the artifact | Current directory name |
| `--helm-flavour` | Helm chart flavour (`stateful` or `stateless`) | Required |
| `--docker-namespace` | Docker registry namespace | - |
| `--docker-host` | Docker registry host | `REGISTRY_HOST` env var |
| `--docker-build` | Whether to run Docker build | `true` |
| `--kubernetes-config` | Path to kubeconfig file | `~/.kube/config` |
| `--kubernetes-context` | Kubernetes context to use | Current context |
| `--env` | Comma-separated list of environments | - |
| `--dry-run` | Exercise pipeline without mutating resources | `false` |
| `--auto-approve` | Skip confirmation prompts | `false` |

### Dry-Run Mode

Test your deployment without making changes:

```sh
dockwright deploy --dry-run=true
```

This simulates all operations and shows what commands would be executed.

### Environment-Specific Deployments

Dockwright supports multi-environment deployments. Specify environments via CLI or config:

```sh
dockwright deploy --env=staging,production
```

For each environment, Dockwright expects a values file at:
```
.dockwright/helm/<env>.values.yaml
```

For example, with `--env=staging,production`, you need:
- `.dockwright/helm/staging.values.yaml`
- `.dockwright/helm/production.values.yaml`

### Skipping Docker Build

If you only need to deploy without rebuilding the image:

```sh
dockwright deploy --docker-build=false
```

This is useful when deploying pre-built images or when no Dockerfile exists.

### Auto-Approve for CI/CD

For automated pipelines, skip interactive prompts:

```sh
dockwright deploy --auto-approve=true
```

Deployments always use the same copied charts, ensuring deterministic behavior across environments.

### Updating Base Charts

If you modify or add charts inside `base-helm-charts/`, re-run:

```sh
sudo make package
```

This copies the updated charts directly to `/usr/local/share/dockwright/charts` without needing to re-install the CLI.

---

## Uninstallation

```sh
sudo make uninstall
```

This removes:

```
/usr/local/bin/dockwright
/usr/local/share/dockwright/charts
```
