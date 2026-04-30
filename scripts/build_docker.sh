#!/bin/bash
set -euo pipefail

readonly APP_NAME="octopus"
readonly MAIN_DIR="./"
readonly OUTPUT_DIR="build"

readonly BUILD_TIME="$(TZ='Asia/Shanghai' date +'%F %T %z')"
readonly GIT_AUTHOR="bestrui"
readonly GIT_VERSION="$(git describe --tags --abbrev=0 2>/dev/null || echo 'dev')"
readonly COMMIT_ID="$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"

readonly LDFLAGS="-X 'github.com/bestruirui/octopus/internal/conf.Version=${GIT_VERSION}' \
                  -X 'github.com/bestruirui/octopus/internal/conf.BuildTime=${BUILD_TIME}' \
                  -X 'github.com/bestruirui/octopus/internal/conf.Author=${GIT_AUTHOR}' \
                  -X 'github.com/bestruirui/octopus/internal/conf.Commit=${COMMIT_ID}' \
                  -s -w"

log_info()    { echo "ℹ️  $1"; }
log_success() { echo "✅ $1"; }
log_error()   { echo "❌ $1" >&2; }
log_step()    { echo -e "\n🔧 $1\n────────────────────────────────────────"; }

command_exists() { command -v "$1" >/dev/null 2>&1; }

ensure_commands() {
  for cmd in go python3 node pnpm git curl; do
    if ! command_exists "$cmd"; then
      log_error "$cmd is required"
      exit 1
    fi
  done
}

prepare_output_dirs() {
  mkdir -p "$OUTPUT_DIR"/{bin,docker}
}

build_frontend() {
  log_step "Building frontend"
  cd web
  pnpm install
  NEXT_PUBLIC_APP_VERSION="$GIT_VERSION" pnpm run build
  cd ..
  rm -rf static/out
  mv web/out static/
  log_success "Frontend built"
}

update_price() {
  log_step "Updating price"
  python3 scripts/updatePrice.py
  log_success "Price updated"
}

build_linux_targets() {
  log_step "Building linux binaries"
  local archs=("x86_64:amd64" "arm64:arm64" "armv7:arm" "x86:386")

  for pair in "${archs[@]}"; do
    local arch="${pair%%:*}"
    local go_arch="${pair#*:}"
    local output_file="$OUTPUT_DIR/bin/${APP_NAME}-linux-${arch}"

    GOOS=linux GOARCH="$go_arch" CGO_ENABLED=0 \
      go build -o "$output_file" -ldflags="$LDFLAGS" -tags=jsoniter "$MAIN_DIR"

    log_success "Built linux/$arch"
  done
}

prepare_docker_binaries() {
  log_step "Preparing docker binaries"
  local platforms=("x86_64:linux/amd64" "x86:linux/386" "armv7:linux/arm/v7" "arm64:linux/arm64")

  for platform in "${platforms[@]}"; do
    local arch="${platform%%:*}"
    local docker_platform="${platform#*:}"
    local platform_dir="$OUTPUT_DIR/docker/$docker_platform"
    mkdir -p "$platform_dir"
    cp "$OUTPUT_DIR/bin/${APP_NAME}-linux-${arch}" "$platform_dir/$APP_NAME"
  done

  log_success "Docker binaries ready"
}

main() {
  ensure_commands
  prepare_output_dirs
  build_frontend
  update_price
  build_linux_targets
  prepare_docker_binaries
  log_success "Docker build artifacts prepared in $OUTPUT_DIR/docker/"
}

main "$@"
