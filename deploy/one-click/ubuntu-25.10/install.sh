#!/usr/bin/env bash
set -euo pipefail

SCRIPT_VERSION="0.5.0"
SUPPORTED_OS_ID="ubuntu"
SUPPORTED_VERSION_ID="25.10"

INSTALL_DIR="/opt/snowpanel"
REPO_URL="https://github.com/snowfallx-bot/SnowPanel.git"
BRANCH="main"
APP_ENV="production"
BACKEND_PORT="8080"
FRONTEND_PORT="5173"
ADMIN_USERNAME="admin"
ADMIN_EMAIL="admin@snowpanel.local"
BOOTSTRAP_ADMIN="true"
JWT_SECRET=""
ADMIN_PASSWORD=""
FORCE_UNSUPPORTED="false"
DOCKER_REGISTRY_MIRROR=""
DOCKER_PULL_RETRIES="3"
POSTGRES_IMAGE="postgres:16-alpine"
REDIS_IMAGE="redis:7-alpine"
POSTGRES_IMAGE_FALLBACK="postgres:16"
REDIS_IMAGE_FALLBACK="redis:7"

GENERATED_JWT_SECRET="false"
GENERATED_ADMIN_PASSWORD="false"

log() {
  printf '[SnowPanel][INFO] %s\n' "$*"
}

warn() {
  printf '[SnowPanel][WARN] %s\n' "$*" >&2
}

die() {
  printf '[SnowPanel][ERROR] %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
SnowPanel one-click installer (Ubuntu 25.10 host-agent mode)

Usage:
  sudo bash install.sh [options]

Options:
  --install-dir <path>       Install path (default: /opt/snowpanel)
  --repo-url <url>           Repository URL
  --branch <name>            Git branch/tag (default: main)
  --app-env <env>            App env (default: production)
  --backend-port <port>      Backend published port (default: 8080)
  --frontend-port <port>     Frontend published port (default: 5173)
  --admin-username <name>    Bootstrap admin username (default: admin)
  --admin-email <email>      Bootstrap admin email
  --admin-password <secret>  Bootstrap admin password (auto-generate if omitted)
  --jwt-secret <secret>      JWT secret (auto-generate if omitted)
  --docker-registry-mirror <url>  Docker registry mirror URL
  --docker-pull-retries <n>  Retry count for docker pull/compose up (default: 3)
  --postgres-image <image>   Primary PostgreSQL image (default: postgres:16-alpine)
  --redis-image <image>      Primary Redis image (default: redis:7-alpine)
  --postgres-image-fallback <image>  Fallback PostgreSQL image (default: postgres:16)
  --redis-image-fallback <image>     Fallback Redis image (default: redis:7)
  --force-unsupported        Allow non-Ubuntu-25.10 systems
  -h, --help                 Show this help

Install result:
  - host core-agent systemd service
  - backend/frontend/postgres/redis by docker compose (host-agent override)
EOF
}

require_option_value() {
  local option="$1"
  local value="${2:-}"
  if [[ -z "${value}" || "${value}" == --* ]]; then
    die "option ${option} requires a value"
  fi
}

validate_port() {
  local name="$1"
  local value="$2"
  if ! [[ "${value}" =~ ^[0-9]+$ ]]; then
    die "${name} must be a number: ${value}"
  fi
  if (( value < 1 || value > 65535 )); then
    die "${name} must be between 1 and 65535: ${value}"
  fi
}

validate_positive_int() {
  local name="$1"
  local value="$2"
  if ! [[ "${value}" =~ ^[0-9]+$ ]]; then
    die "${name} must be a positive integer: ${value}"
  fi
  if (( value < 1 )); then
    die "${name} must be >= 1: ${value}"
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --install-dir)
      require_option_value "$1" "${2:-}"
      INSTALL_DIR="${2:-}"
      shift 2
      ;;
    --repo-url)
      require_option_value "$1" "${2:-}"
      REPO_URL="${2:-}"
      shift 2
      ;;
    --branch)
      require_option_value "$1" "${2:-}"
      BRANCH="${2:-}"
      shift 2
      ;;
    --app-env)
      require_option_value "$1" "${2:-}"
      APP_ENV="${2:-}"
      shift 2
      ;;
    --backend-port)
      require_option_value "$1" "${2:-}"
      BACKEND_PORT="${2:-}"
      shift 2
      ;;
    --frontend-port)
      require_option_value "$1" "${2:-}"
      FRONTEND_PORT="${2:-}"
      shift 2
      ;;
    --admin-username)
      require_option_value "$1" "${2:-}"
      ADMIN_USERNAME="${2:-}"
      shift 2
      ;;
    --admin-email)
      require_option_value "$1" "${2:-}"
      ADMIN_EMAIL="${2:-}"
      shift 2
      ;;
    --admin-password)
      require_option_value "$1" "${2:-}"
      ADMIN_PASSWORD="${2:-}"
      shift 2
      ;;
    --jwt-secret)
      require_option_value "$1" "${2:-}"
      JWT_SECRET="${2:-}"
      shift 2
      ;;
    --docker-registry-mirror)
      require_option_value "$1" "${2:-}"
      DOCKER_REGISTRY_MIRROR="${2:-}"
      shift 2
      ;;
    --docker-pull-retries)
      require_option_value "$1" "${2:-}"
      DOCKER_PULL_RETRIES="${2:-}"
      shift 2
      ;;
    --postgres-image)
      require_option_value "$1" "${2:-}"
      POSTGRES_IMAGE="${2:-}"
      shift 2
      ;;
    --redis-image)
      require_option_value "$1" "${2:-}"
      REDIS_IMAGE="${2:-}"
      shift 2
      ;;
    --postgres-image-fallback)
      require_option_value "$1" "${2:-}"
      POSTGRES_IMAGE_FALLBACK="${2:-}"
      shift 2
      ;;
    --redis-image-fallback)
      require_option_value "$1" "${2:-}"
      REDIS_IMAGE_FALLBACK="${2:-}"
      shift 2
      ;;
    --force-unsupported)
      FORCE_UNSUPPORTED="true"
      shift 1
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      die "unknown argument: $1"
      ;;
  esac
done

if [[ -z "${INSTALL_DIR}" || "${INSTALL_DIR}" == "/" ]]; then
  die "install directory must not be empty or /"
fi
if [[ "${INSTALL_DIR}" != /* ]]; then
  die "install directory must be an absolute path: ${INSTALL_DIR}"
fi

validate_port "BACKEND_PORT" "${BACKEND_PORT}"
validate_port "FRONTEND_PORT" "${FRONTEND_PORT}"
validate_positive_int "DOCKER_PULL_RETRIES" "${DOCKER_PULL_RETRIES}"
if [[ -n "${DOCKER_REGISTRY_MIRROR}" ]] && ! [[ "${DOCKER_REGISTRY_MIRROR}" =~ ^https?:// ]]; then
  die "DOCKER_REGISTRY_MIRROR must start with http:// or https://"
fi
if [[ -z "${POSTGRES_IMAGE}" || -z "${REDIS_IMAGE}" ]]; then
  die "POSTGRES_IMAGE and REDIS_IMAGE must not be empty"
fi

if [[ "${EUID}" -ne 0 ]]; then
  die "please run as root (example: sudo bash install.sh)"
fi

if [[ -f /etc/os-release ]]; then
  # shellcheck disable=SC1091
  source /etc/os-release
else
  die "/etc/os-release not found; cannot detect operating system"
fi

OS_ID="${ID:-unknown}"
OS_VERSION_ID="${VERSION_ID:-unknown}"
OS_CODENAME="${VERSION_CODENAME:-${UBUNTU_CODENAME:-}}"

if [[ "${OS_ID}" != "${SUPPORTED_OS_ID}" || "${OS_VERSION_ID}" != "${SUPPORTED_VERSION_ID}" ]]; then
  if [[ "${FORCE_UNSUPPORTED}" != "true" ]]; then
    die "unsupported OS: ${OS_ID} ${OS_VERSION_ID}. expected ${SUPPORTED_OS_ID} ${SUPPORTED_VERSION_ID}. use --force-unsupported to continue."
  fi
  warn "forcing install on unsupported OS: ${OS_ID} ${OS_VERSION_ID}"
fi

if [[ -z "${OS_CODENAME}" ]]; then
  die "cannot resolve ubuntu codename from /etc/os-release"
fi

apt_install() {
  DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "$@"
}

configure_docker_registry_mirror() {
  local mirror="$1"
  if [[ -z "${mirror}" ]]; then
    return 0
  fi

  log "configuring docker registry mirror: ${mirror}"
  install -d -m 755 /etc/docker
  local daemon_file="/etc/docker/daemon.json"
  local tmp_file
  tmp_file="$(mktemp)"

  if [[ -f "${daemon_file}" ]]; then
    if jq empty "${daemon_file}" >/dev/null 2>&1; then
      jq --arg mirror "${mirror}" \
        '.["registry-mirrors"] = ((.["registry-mirrors"] // []) + [$mirror] | unique)' \
        "${daemon_file}" >"${tmp_file}"
    else
      warn "existing ${daemon_file} is invalid json; backup and replace with mirror config"
      cp "${daemon_file}" "${daemon_file}.bak.$(date +%s)"
      jq -n --arg mirror "${mirror}" '{"registry-mirrors": [$mirror]}' >"${tmp_file}"
    fi
  else
    jq -n --arg mirror "${mirror}" '{"registry-mirrors": [$mirror]}' >"${tmp_file}"
  fi

  install -m 644 "${tmp_file}" "${daemon_file}"
  rm -f "${tmp_file}"
  systemctl restart docker
}

recover_docker_daemon() {
  systemctl reset-failed docker >/dev/null 2>&1 || true
  if systemctl restart docker >/dev/null 2>&1; then
    return 0
  fi
  systemctl start docker >/dev/null 2>&1 || true
  systemctl is-active --quiet docker
}

run_with_retry() {
  local retries="$1"
  local description="$2"
  shift 2

  local attempt=1
  while (( attempt <= retries )); do
    if "$@"; then
      return 0
    fi
    warn "${description} failed (attempt ${attempt}/${retries})"
    if (( attempt < retries )); then
      warn "recovering docker daemon before retry"
      recover_docker_daemon || warn "docker daemon is still not active after recovery attempt"
      sleep $((attempt * 3))
    fi
    attempt=$((attempt + 1))
  done

  return 1
}

docker_pull_image() {
  local image="$1"
  docker pull "${image}" >&2
}

clear_registry_mirrors_if_present() {
  local daemon_file="/etc/docker/daemon.json"
  if [[ ! -f "${daemon_file}" ]]; then
    return 1
  fi

  if ! jq -e '.["registry-mirrors"] and (.["registry-mirrors"] | length > 0)' "${daemon_file}" >/dev/null 2>&1; then
    return 1
  fi

  local backup_file="${daemon_file}.bak.$(date +%s)"
  cp "${daemon_file}" "${backup_file}"
  jq 'del(.["registry-mirrors"])' "${daemon_file}" >"${daemon_file}.tmp"
  install -m 644 "${daemon_file}.tmp" "${daemon_file}"
  rm -f "${daemon_file}.tmp"
  warn "removed registry-mirrors from ${daemon_file}; backup: ${backup_file}"
  recover_docker_daemon || true
  return 0
}

pull_image_with_fallback() {
  local role="$1"
  local primary="$2"
  local fallback="$3"

  if run_with_retry "${DOCKER_PULL_RETRIES}" "docker pull ${primary}" docker_pull_image "${primary}"; then
    printf '%s' "${primary}"
    return 0
  fi

  warn "${role} primary image pull failed: ${primary}"
  warn "removing possibly broken local reference: ${primary}"
  docker image rm -f "${primary}" >/dev/null 2>&1 || true

  if [[ -z "${fallback}" ]]; then
    if clear_registry_mirrors_if_present; then
      warn "retrying ${role} primary image without registry mirrors"
      if run_with_retry "${DOCKER_PULL_RETRIES}" "docker pull ${primary}" docker_pull_image "${primary}"; then
        printf '%s' "${primary}"
        return 0
      fi
    fi
    return 1
  fi

  warn "trying ${role} fallback image: ${fallback}"
  if run_with_retry "${DOCKER_PULL_RETRIES}" "docker pull ${fallback}" docker_pull_image "${fallback}"; then
    printf '%s' "${fallback}"
    return 0
  fi

  warn "${role} fallback image pull failed: ${fallback}"
  docker image rm -f "${fallback}" >/dev/null 2>&1 || true

  if clear_registry_mirrors_if_present; then
    warn "retrying ${role} images without registry mirrors"
    if run_with_retry "${DOCKER_PULL_RETRIES}" "docker pull ${primary}" docker_pull_image "${primary}"; then
      printf '%s' "${primary}"
      return 0
    fi
    if run_with_retry "${DOCKER_PULL_RETRIES}" "docker pull ${fallback}" docker_pull_image "${fallback}"; then
      printf '%s' "${fallback}"
      return 0
    fi
  fi

  return 1
}

set_env_file_value() {
  local file="$1"
  local key="$2"
  local value="$3"
  local escaped
  escaped="$(printf '%s' "${value}" | sed -e 's/[\/&]/\\&/g')"
  if grep -qE "^${key}=" "${file}"; then
    sed -i "s/^${key}=.*/${key}=${escaped}/" "${file}"
  else
    printf '%s=%s\n' "${key}" "${value}" >> "${file}"
  fi
}

generate_token() {
  local length="$1"
  set +o pipefail
  local output
  output="$(tr -dc 'A-Za-z0-9' < /dev/urandom | head -c "${length}")"
  set -o pipefail
  printf '%s' "${output}"
}

generate_password() {
  local upper lower digit symbol rest raw
  set +o pipefail
  upper="$(tr -dc 'A-Z' < /dev/urandom | head -c 1)"
  lower="$(tr -dc 'a-z' < /dev/urandom | head -c 1)"
  digit="$(tr -dc '0-9' < /dev/urandom | head -c 1)"
  symbol="$(tr -dc '!@#$%^*+=_' < /dev/urandom | head -c 1)"
  rest="$(tr -dc 'A-Za-z0-9!@#$%^*+=_' < /dev/urandom | head -c 20)"
  set -o pipefail
  raw="${upper}${lower}${digit}${symbol}${rest}"
  printf '%s' "${raw}"
}

log "starting installer v${SCRIPT_VERSION}"
log "target OS: ${OS_ID} ${OS_VERSION_ID} (${OS_CODENAME})"

log "installing base packages"
apt-get update -y
apt_install ca-certificates curl gnupg git lsb-release jq build-essential pkg-config libssl-dev

if ! command -v docker >/dev/null 2>&1 || ! docker compose version >/dev/null 2>&1; then
  log "installing docker engine and compose plugin"
  DOCKER_REPO_CODENAME="${OS_CODENAME}"
  if ! curl -fsI "https://download.docker.com/linux/ubuntu/dists/${DOCKER_REPO_CODENAME}/Release" >/dev/null 2>&1; then
    warn "docker apt repo does not contain '${DOCKER_REPO_CODENAME}', falling back to 'noble'"
    DOCKER_REPO_CODENAME="noble"
  fi
  install -m 0755 -d /etc/apt/keyrings
  curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
  chmod a+r /etc/apt/keyrings/docker.asc

  cat >/etc/apt/sources.list.d/docker.list <<EOF
deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu ${DOCKER_REPO_CODENAME} stable
EOF
  apt-get update -y
  apt_install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
fi

systemctl enable --now docker
configure_docker_registry_mirror "${DOCKER_REGISTRY_MIRROR}"
recover_docker_daemon || die "docker service is not active after setup"

if ! command -v cargo >/dev/null 2>&1; then
  log "installing rust toolchain (minimal)"
  curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y --profile minimal
fi

if [[ -f /root/.cargo/env ]]; then
  # shellcheck disable=SC1091
  source /root/.cargo/env
fi

if ! command -v cargo >/dev/null 2>&1; then
  die "cargo is not available after rustup install"
fi

if [[ -d "${INSTALL_DIR}/.git" ]]; then
  log "updating existing repository at ${INSTALL_DIR}"
  git -C "${INSTALL_DIR}" fetch --tags origin
  git -C "${INSTALL_DIR}" checkout "${BRANCH}"
  git -C "${INSTALL_DIR}" pull --ff-only origin "${BRANCH}"
else
  log "cloning repository to ${INSTALL_DIR}"
  rm -rf "${INSTALL_DIR}"
  git clone --depth 1 --branch "${BRANCH}" "${REPO_URL}" "${INSTALL_DIR}"
fi

ENV_FILE="${INSTALL_DIR}/.env"
if [[ ! -f "${ENV_FILE}" ]]; then
  cp "${INSTALL_DIR}/.env.example" "${ENV_FILE}"
fi

if [[ -z "${JWT_SECRET}" ]]; then
  JWT_SECRET="$(generate_token 56)"
  GENERATED_JWT_SECRET="true"
fi

if [[ "${BOOTSTRAP_ADMIN}" == "true" && -z "${ADMIN_PASSWORD}" ]]; then
  ADMIN_PASSWORD="$(generate_password)"
  GENERATED_ADMIN_PASSWORD="true"
fi

log "writing .env settings"
set_env_file_value "${ENV_FILE}" "APP_ENV" "${APP_ENV}"
set_env_file_value "${ENV_FILE}" "BACKEND_PORT" "${BACKEND_PORT}"
set_env_file_value "${ENV_FILE}" "FRONTEND_PORT" "${FRONTEND_PORT}"
set_env_file_value "${ENV_FILE}" "AGENT_TARGET" "host.docker.internal:50051"
set_env_file_value "${ENV_FILE}" "JWT_SECRET" "${JWT_SECRET}"
set_env_file_value "${ENV_FILE}" "BOOTSTRAP_ADMIN" "${BOOTSTRAP_ADMIN}"
set_env_file_value "${ENV_FILE}" "DEFAULT_ADMIN_USERNAME" "${ADMIN_USERNAME}"
set_env_file_value "${ENV_FILE}" "DEFAULT_ADMIN_EMAIL" "${ADMIN_EMAIL}"
set_env_file_value "${ENV_FILE}" "DEFAULT_ADMIN_PASSWORD" "${ADMIN_PASSWORD}"
set_env_file_value "${ENV_FILE}" "LOGIN_ATTEMPT_STORE" "redis"
set_env_file_value "${ENV_FILE}" "VITE_API_BASE_URL" "http://127.0.0.1:${BACKEND_PORT}"

log "building and installing host core-agent service"
pushd "${INSTALL_DIR}/core-agent" >/dev/null
cargo build --release
popd >/dev/null

install -Dm755 "${INSTALL_DIR}/core-agent/target/release/core-agent" /usr/local/bin/core-agent
install -d -m 750 /etc/snowpanel

if [[ ! -f /etc/snowpanel/core-agent.env ]]; then
  install -Dm640 "${INSTALL_DIR}/deploy/core-agent/systemd/core-agent.env.example" /etc/snowpanel/core-agent.env
  log "created /etc/snowpanel/core-agent.env from template"
else
  log "keeping existing /etc/snowpanel/core-agent.env"
fi

install -Dm644 "${INSTALL_DIR}/deploy/core-agent/systemd/core-agent.service" /etc/systemd/system/core-agent.service
systemctl daemon-reload
systemctl enable --now core-agent
systemctl restart core-agent
systemctl is-active --quiet core-agent || die "core-agent service is not active"

log "starting docker compose stack (host-agent mode)"
pushd "${INSTALL_DIR}" >/dev/null
log "pre-pulling runtime images (with retries and fallback)"
POSTGRES_IMAGE="$(pull_image_with_fallback "postgres" "${POSTGRES_IMAGE}" "${POSTGRES_IMAGE_FALLBACK}")" \
  || die "failed to pull postgres image (primary: ${POSTGRES_IMAGE}, fallback: ${POSTGRES_IMAGE_FALLBACK})"
REDIS_IMAGE="$(pull_image_with_fallback "redis" "${REDIS_IMAGE}" "${REDIS_IMAGE_FALLBACK}")" \
  || die "failed to pull redis image (primary: ${REDIS_IMAGE}, fallback: ${REDIS_IMAGE_FALLBACK})"

set_env_file_value "${ENV_FILE}" "POSTGRES_IMAGE" "${POSTGRES_IMAGE}"
set_env_file_value "${ENV_FILE}" "REDIS_IMAGE" "${REDIS_IMAGE}"
log "selected runtime images: POSTGRES_IMAGE=${POSTGRES_IMAGE}, REDIS_IMAGE=${REDIS_IMAGE}"

run_with_retry "${DOCKER_PULL_RETRIES}" "docker compose up" \
  docker compose -f docker-compose.yml -f docker-compose.host-agent.yml up -d --build \
  || die "docker compose up failed after ${DOCKER_PULL_RETRIES} attempts"
popd >/dev/null

wait_http() {
  local url="$1"
  local max_attempts="${2:-60}"
  local interval_seconds="${3:-2}"
  local attempt=1
  while (( attempt <= max_attempts )); do
    if curl -fsS "${url}" >/dev/null 2>&1; then
      return 0
    fi
    sleep "${interval_seconds}"
    attempt=$((attempt + 1))
  done
  return 1
}

log "verifying backend health"
wait_http "http://127.0.0.1:${BACKEND_PORT}/health" 60 2 || die "backend /health check failed"
wait_http "http://127.0.0.1:${BACKEND_PORT}/ready" 60 2 || die "backend /ready check failed"

CREDENTIAL_DIR="/root/.snowpanel"
CREDENTIAL_FILE="${CREDENTIAL_DIR}/installer-output.env"
install -d -m 700 "${CREDENTIAL_DIR}"
cat >"${CREDENTIAL_FILE}" <<EOF
INSTALL_TIME_UTC=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
INSTALL_DIR=${INSTALL_DIR}
APP_ENV=${APP_ENV}
BACKEND_PORT=${BACKEND_PORT}
FRONTEND_PORT=${FRONTEND_PORT}
ADMIN_USERNAME=${ADMIN_USERNAME}
ADMIN_EMAIL=${ADMIN_EMAIL}
DEFAULT_ADMIN_PASSWORD=${ADMIN_PASSWORD}
JWT_SECRET=${JWT_SECRET}
EOF
chmod 600 "${CREDENTIAL_FILE}"

log "installation completed"
printf '\n'
printf 'SnowPanel is ready:\n'
printf '  Frontend: http://127.0.0.1:%s\n' "${FRONTEND_PORT}"
printf '  Backend health: http://127.0.0.1:%s/health\n' "${BACKEND_PORT}"
printf '  Credentials file: %s\n' "${CREDENTIAL_FILE}"
printf '\n'
printf 'Bootstrap admin:\n'
printf '  username: %s\n' "${ADMIN_USERNAME}"
printf '  email: %s\n' "${ADMIN_EMAIL}"
if [[ "${GENERATED_ADMIN_PASSWORD}" == "true" ]]; then
  printf '  password: %s (auto-generated)\n' "${ADMIN_PASSWORD}"
else
  printf '  password: (from --admin-password or existing env)\n'
fi
if [[ "${GENERATED_JWT_SECRET}" == "true" ]]; then
  printf '  jwt secret: auto-generated and stored in %s\n' "${CREDENTIAL_FILE}"
else
  printf '  jwt secret: provided externally\n'
fi
