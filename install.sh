#!/usr/bin/env bash

set -euo pipefail

readonly repository="osspkg/gvm"
readonly home_dir="${GVM_HOME:-${HOME}/.gvm}"
readonly bin_dir="${home_dir}/bin"
readonly cache_dir="${home_dir}/.cache"
readonly temp_dir="$(mktemp -d)"
trap 'rm -rf "${temp_dir}"' EXIT

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

platform_os() {
  case "$(uname -s)" in
    Linux) echo "linux" ;;
    Darwin) echo "darwin" ;;
    *) echo "unsupported operating system: $(uname -s)" >&2; exit 1 ;;
  esac
}

platform_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    arm64|aarch64) echo "arm64" ;;
    *) echo "unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
}

append_profile_block() {
  local profile="$1"
  touch "${profile}"
  if grep -Fq '# >>> gvm >>>' "${profile}"; then
    return
  fi
  cat >>"${profile}" <<'PROFILE'

# >>> gvm >>>
export GVM_HOME="${GVM_HOME:-$HOME/.gvm}"
case ":${PATH:-}:" in
  *":${GVM_HOME}/bin:"*) ;;
  *) export PATH="${GVM_HOME}/bin${PATH:+:${PATH}}" ;;
esac
# <<< gvm <<<
PROFILE
}

require_command curl
require_command tar

os_name="$(platform_os)"
arch_name="$(platform_arch)"
api_response="${temp_dir}/release.json"
curl --fail --silent --show-error --location \
  "https://api.github.com/repos/${repository}/releases/latest" \
  -H 'Accept: application/vnd.github+json' -H 'User-Agent: gvm-installer' \
  -o "${api_response}"
release_tag="$(sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "${api_response}" | head -n 1)"
if [ -z "${release_tag}" ]; then
  echo "latest gvm release has no tag" >&2
  exit 1
fi
release_version="${release_tag#v}"
archive_name="gvm_${release_version}_${os_name}_${arch_name}.tar.gz"
archive_path="${temp_dir}/${archive_name}"
curl --fail --silent --show-error --location \
  "https://github.com/${repository}/releases/download/${release_tag}/${archive_name}" \
  -o "${archive_path}"

mkdir -p "${bin_dir}" "${cache_dir}/bin" "${cache_dir}/pkg" "${cache_dir}/src"
tar -xzf "${archive_path}" -C "${temp_dir}"
install -m 0755 "${temp_dir}/gvm" "${bin_dir}/gvm"
install -m 0755 "${temp_dir}/go" "${bin_dir}/go"

append_profile_block "${HOME}/.profile"
append_profile_block "${HOME}/.bashrc"
append_profile_block "${HOME}/.zshrc"

echo "gvm ${release_tag} installed in ${home_dir}"
echo "Open a new shell or source your profile to use gvm and go."
