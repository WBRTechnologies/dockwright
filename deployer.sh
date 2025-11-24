#!/usr/bin/env bash
set -euo pipefail

########################################
# ❗ Assumptions / Requirements
#
# - Script is vendor-agnostic: registry host, namespace, repo,
#   username, and password must be supplied via environment variables.
# - Registry must support standard Docker `docker login` using
#   username/password (no vendor-specific helpers like `aws ecr get-login-password`).
# - `flavour` field in ./helm/config.yaml must be either:
#       stateless  OR  stateful
# - Base charts are packaged locally as:
#       $CHARTS_DIR/stateless.tgz
#       $CHARTS_DIR/stateful.tgz
#   where CHARTS_DIR is defined in /etc/dockwright.conf.
# - Current directory name is treated as service name unless overridden.
# - ./helm/config.yaml and ./helm/values.yaml must exist.
########################################

########################################
# 🎯 Deployer Script
#
# - Builds and pushes a Docker image from CWD
# - Uses directory name as image name
# - Pushes to a generic OCI-compatible registry
# - Reads flavour from ./helm/config.yaml
# - Uses stateless/stateful base Helm chart based on flavour
# - Uses ./helm/values.yaml for values
# - Uses locally packaged Helm charts (.tgz) for deployment
########################################

########################################
# 🌈 Pretty Logging Setup
########################################

if [[ -t 1 ]]; then
  BOLD="$(tput bold)" || BOLD=""
  RESET="$(tput sgr0)" || RESET=""
  RED="$(tput setaf 1)" || RED=""
  GREEN="$(tput setaf 2)" || GREEN=""
  YELLOW="$(tput setaf 3)" || YELLOW=""
  BLUE="$(tput setaf 4)" || BLUE=""
  CYAN="$(tput setaf 6)" || CYAN=""
else
  BOLD=""; RESET=""; RED=""; GREEN=""; YELLOW=""; BLUE=""; CYAN="";
fi

ICON_INFO="[ℹ]"
ICON_OK="[✔]"
ICON_WARN="[!]"
ICON_ERR="[✖]"
ICON_STEP="[→]"
ICON_SECTION="[§]"

log_section() {
  echo ""
  echo "${BOLD}${BLUE}${ICON_SECTION} $1${RESET}"
  echo "${BLUE}────────────────────────────────────────────${RESET}"
}

log_info() {
  echo "${CYAN}${ICON_INFO} $1${RESET}"
}

log_step() {
  echo "${BLUE}${ICON_STEP} $1${RESET}"
}

log_success() {
  echo "${GREEN}${ICON_OK} $1${RESET}"
}

log_warn() {
  echo "${YELLOW}${ICON_WARN} $1${RESET}" >&2
}

log_error() {
  echo "${RED}${ICON_ERR} $1${RESET}" >&2
}

die() {
  log_error "$1"
  exit 1
}

########################################
# 📁 Paths & Static Config
########################################

readonly CONFIG_FILE="./helm/config.yaml"
readonly VALUES_FILE="./helm/values.yaml"

# Global dockwright config file path
readonly DOCKWRIGHT_CONF="/etc/dockwright.conf"

########################################
# ⚙️ Load Global Config
########################################

load_config() {
  log_section "⚙️ Loading global config"

  if [[ ! -f "$DOCKWRIGHT_CONF" ]]; then
    die "Config file not found: $DOCKWRIGHT_CONF
Expected to contain CHARTS_DIR=/path/to/charts.
Install dockwright (make install) to create it."
  fi

  # shellcheck source=/etc/dockwright.conf
  # (path is fixed and intentional)
  # We expect this to define CHARTS_DIR.
  source "$DOCKWRIGHT_CONF"

  if [[ -z "${CHARTS_DIR:-}" ]]; then
    die "CHARTS_DIR not set in $DOCKWRIGHT_CONF – refusing to continue."
  fi

  log_success "Loaded config from $DOCKWRIGHT_CONF"
  log_step "CHARTS_DIR: $CHARTS_DIR"
}

########################################
# 🔧 CLI prerequisites
########################################

require_cmd() {
  local cmd=$1
  command -v "$cmd" >/dev/null 2>&1 || die "Required command '$cmd' not found in PATH."
}

check_prereqs() {
  log_section "🔧 Checking CLI prerequisites"
  require_cmd docker
  require_cmd helm
  require_cmd awk
  log_success "All required commands are available."
}

########################################
# 🔐 Required Environment Variables
########################################

require_env() {
  local name=$1
  [[ -n "${!name-}" ]] || die "${name} env var is required"
}

check_env() {
  log_section "🔐 Validating required environment variables"
  require_env REGISTRY_NAMESPACE
  require_env REGISTRY_REPOSITORY
  require_env REGISTRY_USERNAME
  require_env REGISTRY_PASSWORD
  require_env REGISTRY_HOST
  log_success "All required environment variables are present."
}

########################################
# ✅ Pre-flight File Checks
########################################

check_files() {
  log_section "📁 Checking required files"

  [[ -f "$CONFIG_FILE" ]] || die "Config file not found: $CONFIG_FILE"
  log_success "Found config file: $CONFIG_FILE"

  [[ -f "$VALUES_FILE" ]] || die "Values file not found: $VALUES_FILE"
  log_success "Found values file: $VALUES_FILE"
}

########################################
# 🧾 Flavour Detection & Validation
########################################

read_and_validate_flavour() {
  log_section "🧾 Reading and validating flavour"

  # Extract `flavour:` from helm/config.yaml (simple, flat YAML assumption)
  local raw_flavour
  raw_flavour=$(awk -F': *' '/^flavour:/ {print $2}' "$CONFIG_FILE" | tr -d '"[:space:]') || true

  if [[ -z "${raw_flavour:-}" ]]; then
    die "'flavour' not set in $CONFIG_FILE – refusing to build."
  fi

  FLAVOUR="$raw_flavour"
  log_success "Flavour '${FLAVOUR}' read from config."
}

########################################
# 🐳 Docker Build & Push
########################################

build_and_push_image() {
  log_section "🐳 Building and pushing Docker image"

  SERVICE_NAME="${SERVICE_NAME:-$(basename "$PWD")}"
  IMAGE_TAG="${IMAGE_TAG:-latest}"

  IMAGE_REPO="${REGISTRY_HOST}/${REGISTRY_NAMESPACE}/${REGISTRY_REPOSITORY}/${SERVICE_NAME}"
  IMAGE_REF="${IMAGE_REPO}:${IMAGE_TAG}"

  log_step "Service name: ${SERVICE_NAME}"
  log_step "Image reference: ${IMAGE_REF}"

  log_info "Logging into Docker registry: ${REGISTRY_HOST}"
  echo "${REGISTRY_PASSWORD}" | docker login "${REGISTRY_HOST}" \
    -u "${REGISTRY_USERNAME}" --password-stdin
  log_success "Docker login successful."

  log_info "Building Docker image..."
  DOCKER_DEFAULT_PLATFORM=linux/amd64 \
    docker build -t "${IMAGE_REF}" .
  log_success "Docker build completed."

  log_info "Pushing Docker image..."
  docker push "${IMAGE_REF}"
  log_success "Docker image pushed: ${IMAGE_REF}"
}

########################################
#  🌀 Helm Deployment
########################################

helm_deploy() {
  log_section "🌀 Helm deployment"

  local base_chart_name
  if [[ "$FLAVOUR" == "stateless" ]]; then
    base_chart_name="stateless"
  elif [[ "$FLAVOUR" == "stateful" ]]; then
    base_chart_name="stateful"
  else
    die "Invalid flavour '$FLAVOUR'. Accepted: stateless, stateful – refusing to deploy."
  fi

  local base_chart_tgz_path="${CHARTS_DIR}/${base_chart_name}.tgz"

  log_step "Using base chart package: ${base_chart_tgz_path}"

  log_info "Running helm upgrade --install..."
  helm upgrade --install "${SERVICE_NAME}" \
    "${base_chart_tgz_path}" \
    -f "${VALUES_FILE}" \
    --set image.repository="${IMAGE_REPO}" \
    --set image.tag="${IMAGE_TAG}"

  log_success "Helm release '${SERVICE_NAME}' deployed."
}

########################################
# 🚀 Main
########################################

main() {
  log_section "🚀 Starting deployer"

  load_config
  check_prereqs
  check_env
  check_files
  read_and_validate_flavour
  build_and_push_image
  helm_deploy

  echo ""
  log_success "Deployment flow completed successfully."
}

main "$@"
