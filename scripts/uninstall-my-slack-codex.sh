#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: scripts/uninstall-my-slack-codex.sh

Removes the my-slack binary and skill assets from CODEX_HOME.

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

CODEX_HOME="${CODEX_HOME:-${HOME}/.codex}"

SKILL_NAME="my-slack"
INSTALL_BIN_PATH="${CODEX_HOME}/bin/${SKILL_NAME}"
INSTALL_SKILL_DIR="${CODEX_HOME}/skills/${SKILL_NAME}"
INSTALL_CONFIG_PATH="${HOME}/my-slack.yaml"

remove_file() {
  local path="$1"

  if [[ -f "${path}" ]]; then
    rm -f "${path}"
    printf 'Removed: %s\n' "${path}"
  else
    printf 'Not found (skipping): %s\n' "${path}"
  fi
}

remove_dir() {
  local path="$1"

  if [[ -d "${path}" ]]; then
    rm -rf "${path}"
    printf 'Removed directory: %s\n' "${path}"
  else
    printf 'Not found (skipping): %s\n' "${path}"
  fi
}

remove_file "${INSTALL_BIN_PATH}"
remove_dir  "${INSTALL_SKILL_DIR}"
remove_file "${INSTALL_CONFIG_PATH}"

printf 'Uninstall complete.\n'
