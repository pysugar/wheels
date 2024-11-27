#!/bin/bash

GO_BASE_DIR="/opt/local/go"

# shellcheck disable=SC2207
# shellcheck disable=SC2012
versions=($(ls -d "$GO_BASE_DIR"/1.*.* 2>/dev/null | awk -F'/' '{print $NF}'))

for version in "${versions[@]}"; do
    major_minor=$(echo "$version" | awk -F. '{print $1"."$2}')
    symlink_path="$GO_BASE_DIR/$major_minor"
    target_path="$GO_BASE_DIR/$version"

    if [ ! -d "$target_path" ]; then
        echo "Target directory $target_path is not exists, skip"
        continue
    fi

    if [ -L "$symlink_path" ]; then
        existing_target=$(readlink "$symlink_path")
        if [ "$existing_target" == "$version" ]; then
            echo "symlink $symlink_path has already exists $version, noop"
        else
            echo "symlink $symlink_path has already exists, target $existing_target, skip"
        fi
    elif [ -e "$symlink_path" ]; then
        echo "$symlink_path is not symlink"
    else
        ln -s "$version" "$symlink_path"
        echo "$symlink_path -> $version has created successful"
    fi
done
