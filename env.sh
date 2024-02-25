#!/usr/bin/env bash

export GVM_DIR="${HOME}/.gvm"
export GVM_CACHE="${GVM_DIR}/.cache"
export GVM_BIN="${GVM_DIR}/bin"

go_version(){
  local RC_PATH
  if [ -f "$(pwd)/.gvmrc" ]; then
    RC_PATH="$(pwd)/.gvmrc"
  else
    RC_PATH="${GVM_DIR}/.gvmrc"
  fi
  if [ -f "${RC_PATH}" ]; then
    cat < "${RC_PATH}" | sed -e 's/^[[:space:]]*//'
  fi

  echo ""
}

clean_env_path(){
  local ORIG_PATH
  ORIG_PATH="$PATH"
  local NEW_PATH
  NEW_PATH=""

  while IFS=':' read -ra SRC; do
    for i in "${SRC[@]}"; do
      if [[ "${i}" != *"${GVM_DIR}"* ]]; then
        NEW_PATH="${NEW_PATH}:${i}"
      fi
    done
  done <<< "${ORIG_PATH}"

  export PATH="${NEW_PATH}"
}

go_env(){
  local ver
  ver=$1

  if [ -n "${ver}" ]; then
    export GOROOT="${GVM_CACHE}/go${ver}/go"
    export GOPATH="${GVM_CACHE}/modules"
    export GODEBUG="netdns=go+2"
    export GOVERSION="go${ver}"

    export GVM_GO="${ver}"
    export PATH="${PATH}:${GOPATH}/bin:${GOROOT}/bin"
  fi

  export PATH="${GVM_BIN}:${PATH}"
}

clean_env_path
go_env "$(go_version)"
