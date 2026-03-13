#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/install-my-prom-codex.sh

Builds my-prom and installs the binary and skill assets into CODEX_HOME.

Environment:
  CODEX_HOME  Target Codex home directory. Defaults to $HOME/.codex.
EOF
}

if [[ $# -gt 1 ]]; then
  usage >&2
  exit 1
fi

if [[ ${1:-} == "--help" || ${1:-} == "-h" ]]; then
  usage
  exit 0
fi

if [[ $# -eq 1 ]]; then
  printf 'Unknown argument: %s\n\n' "$1" >&2
  usage >&2
  exit 1
fi

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "${SCRIPT_DIR}/.." && pwd)"
CODEX_HOME="${CODEX_HOME:-${HOME}/.codex}"

SKILL_NAME="my-prom"
SOURCE_SKILL_PATH="${REPO_ROOT}/docs/${SKILL_NAME}/SKILL.md"
SOURCE_EXAMPLE_PATH="${REPO_ROOT}/docs/${SKILL_NAME}/my-prom-example.yaml"
SOURCE_README_PATH="${REPO_ROOT}/src/cmd/${SKILL_NAME}/README.md"
BUILT_BINARY_PATH="${REPO_ROOT}/bin/${SKILL_NAME}"
INSTALL_BIN_DIR="${CODEX_HOME}/bin"
INSTALL_BIN_PATH="${INSTALL_BIN_DIR}/${SKILL_NAME}"
INSTALL_SKILL_DIR="${CODEX_HOME}/skills/${SKILL_NAME}"
INSTALL_SKILL_PATH="${INSTALL_SKILL_DIR}/SKILL.md"
INSTALL_CONFIG_DIR="${HOME}/.config"
INSTALL_CONFIG_PATH="${INSTALL_CONFIG_DIR}/my-prom.yaml"

require_file() {
  local path="$1"

  if [[ ! -f "${path}" ]]; then
    printf 'Required file not found: %s\n' "${path}" >&2
    exit 1
  fi
}

render_skill() {
  local tmp_path
  tmp_path="$(mktemp "${TMPDIR:-/tmp}/my-prom-skill.XXXXXX")"

  MY_PROM_BIN_PATH="${INSTALL_BIN_PATH}" MY_PROM_README_PATH="${SOURCE_README_PATH}" perl -0pe '
    s/If it is not on `PATH`, use the provided absolute binary path\./sprintf(q{If it is not on `PATH`, use `%s`.}, $ENV{MY_PROM_BIN_PATH})/e;
    s{If you need build, test, or lint instructions, read \[README\.md\]\(\.\./\.\./src/cmd/my-prom/README\.md\)\.}{sprintf(q{If you need build, test, or lint instructions, read `%s`.}, $ENV{MY_PROM_README_PATH})}e;
    s/If the binary is not on `PATH`, replace `my-prom` with the provided absolute path\./sprintf(q{If the binary is not on `PATH`, replace `my-prom` with `%s`.}, $ENV{MY_PROM_BIN_PATH})/e;
  ' "${SOURCE_SKILL_PATH}" > "${tmp_path}"

  install -m 0644 "${tmp_path}" "${INSTALL_SKILL_PATH}"
  rm -f "${tmp_path}"
}

require_file "${SOURCE_SKILL_PATH}"
require_file "${SOURCE_EXAMPLE_PATH}"
require_file "${SOURCE_README_PATH}"
require_file "${REPO_ROOT}/Makefile"

printf 'Building %s...\n' "${SKILL_NAME}"
make -C "${REPO_ROOT}" build CMD="${SKILL_NAME}"
require_file "${BUILT_BINARY_PATH}"

mkdir -p "${INSTALL_BIN_DIR}" "${INSTALL_SKILL_DIR}" "${INSTALL_CONFIG_DIR}"

install -m 0755 "${BUILT_BINARY_PATH}" "${INSTALL_BIN_PATH}"

config_status="Installed"
if [[ -e "${INSTALL_CONFIG_PATH}" ]]; then
  config_status="Skipped existing"
else
  install -m 0644 "${SOURCE_EXAMPLE_PATH}" "${INSTALL_CONFIG_PATH}"
fi

render_skill

printf 'Installed binary: %s\n' "${INSTALL_BIN_PATH}"
printf 'Installed skill: %s\n' "${INSTALL_SKILL_PATH}"
printf '%s example config: %s\n' "${config_status}" "${INSTALL_CONFIG_PATH}"
printf 'Add %s to PATH if you want to run my-prom directly from the shell.\n' "${INSTALL_BIN_DIR}"
