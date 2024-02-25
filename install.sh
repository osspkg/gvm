#!/usr/bin/env bash

export GVM_DIR="$HOME/.gvm"

app_has() {
  type "$1" > /dev/null 2>&1
}

app_check() {
  if ! app_has "$1"; then
    echo "Error: $1 not found"
    exit 1
  fi
}

git_clone(){
  git clone --quiet --branch "master" --single-branch https://github.com/osspkg/gvm.git "${GVM_DIR}/.tmp"
}

get_profile_path(){
  local PROFILE_PATH
  PROFILE_PATH=''
  for PROFILE in ".bashrc" ".bash_profile" ".profile"
  do
    if [ -f "${HOME}/${PROFILE}" ]; then
      PROFILE_PATH="${HOME}/${PROFILE}"
      break
    fi
  done
  if [ -z "${PROFILE_PATH}" ]; then
    echo "Failed profile detect"
    exit 1
  fi
  echo "${PROFILE_PATH}"
}

setup_profile(){
  local PROFILE_PATH
  PROFILE_PATH=$(get_profile_path)

  ENV_STRING="\\n[ -s \"\$HOME/.gvm/env.sh\" ] && \\. \"\$HOME/.gvm/env.sh\"  # This loads gvm\\n\\n"

  if ! command grep -qc '/.gvm/env.sh' "${PROFILE_PATH}"; then
    command printf "${ENV_STRING}" >> "${PROFILE_PATH}"
  fi
}

app_check "curl"
app_check "git"

mkdir -p "$GVM_DIR"
mkdir -p "${GVM_DIR}/.cache"
rm -rf "${GVM_DIR}/.tmp"
mkdir -p "${GVM_DIR}/.tmp"
if ! git_clone; then
  echo "Failed clone GVM"
  exit 1
fi
rm -rf "${GVM_DIR}"/.tmp/.git*
cp -rlf "${GVM_DIR}"/.tmp/* "${GVM_DIR}"/
rm -rf "${GVM_DIR}/.tmp"

setup_profile

command gvm default "1.22.0"

echo "Done! Please reboot system!"