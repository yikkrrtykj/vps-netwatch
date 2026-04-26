#!/bin/bash

set -euo pipefail

ROOT_DIR="$(realpath "$(dirname "${BASH_SOURCE[0]}")/..")"
TEMPLATES_FILE="$ROOT_DIR/service/singleton/frontend-templates.yaml"

download_and_extract() {
  local repository="$1"
  local version="$2"
  local targetDir="$3"
  local TMP_DIR

  TMP_DIR="$(mktemp -d)"

  echo "Downloading from repository: $repository, version: $version"

  pushd "$TMP_DIR" || exit

  for attempt in 1 2 3 4 5; do
    rm -f dist.zip
    if curl -fL --retry 3 --retry-all-errors --connect-timeout 20 -o "dist.zip" "$repository/releases/download/$version/dist.zip" && unzip -tq dist.zip >/dev/null; then
      break
    fi

    if [ "$attempt" -eq 5 ]; then
      echo "Failed to download a valid frontend bundle from $repository@$version" >&2
      exit 1
    fi

    echo "Invalid frontend bundle, retrying ($attempt/5)..." >&2
    sleep $((attempt * 3))
  done

  [ -e "$targetDir" ] && rm -r "$targetDir"
  unzip -q dist.zip
  mv dist "$targetDir"

  rm "dist.zip"
  popd || exit
}

count=$(yq eval '. | length' "$TEMPLATES_FILE")

for i in $(seq 0 $(("$count"-1))); do
  path=$(yq -r ".[$i].path" "$TEMPLATES_FILE")
  repository=$(yq -r ".[$i].repository" "$TEMPLATES_FILE")
  version=$(yq -r ".[$i].version" "$TEMPLATES_FILE")

  if [[ -n $path && -n $repository && -n $version ]]; then
    download_and_extract "$repository" "$version" "$ROOT_DIR/cmd/dashboard/$path"
  fi
done
