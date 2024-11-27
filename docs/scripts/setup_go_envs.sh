#!/bin/bash

GO_BASE_DIR="/opt/local/go"

# shellcheck disable=SC2207
versions=($(find "$GO_BASE_DIR" -maxdepth 1 -type l -name '1.*' | awk -F'/' '{print $NF}'))

for version in "${versions[@]}"; do
  if [ -f "/opt/local/envs/go-$version.sh" ]; then
      echo "/opt/local/envs/go-$version.sh has already exists, skip"
      continue
  fi
  cat <<EOF | tee /opt/local/envs/go-$version.sh > /dev/null
export GOROOT=/opt/local/go/$version
export PATH=\$GOROOT/bin:\$PATH
EOF
  echo "Created /opt/local/envs/go-$version.sh"
done


