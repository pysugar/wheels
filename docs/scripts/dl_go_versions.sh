#!/bin/bash

INSTALL_DIR="/opt/local/go"
CURL_PROXY=""

if [ ! -d "$INSTALL_DIR" ]; then
    sudo mkdir -p "$INSTALL_DIR"
    sudo chown "$(whoami):sudo" "$INSTALL_DIR"
fi

while IFS= read -r url || [ -n "$url" ]; do

    [[ -z "$url" || "$url" == \#* ]] && continue

    echo "Processing URL: $url"

    filename="${url##*/}"

    if [[ "$filename" =~ go([0-9]+\.[0-9]+(\.[0-9]+)?).*\.tar\.gz ]]; then
        version="${BASH_REMATCH[1]}"
    else
        echo "Error: Cannot extract version from filename $filename"
        continue
    fi

    if [ -d "${INSTALL_DIR}/${version}" ]; then
        echo ${INSTALL_DIR}/${version}" already exists, skipping download."
        continue
    fi

    TEMP_DIR="/tmp/go-install-$version"
    mkdir -p "$TEMP_DIR"

    cd "$TEMP_DIR" || exit

    if [ -f "$filename" ]; then
        echo "File $filename already exists, skipping download."
    else
        echo "Downloading $filename..."
        # wget -q "$url" -O "$filename"
        curl -x "$CURL_PROXY" -L "$url" -o "$filename"
        if [ $? -ne 0 ]; then
            echo "Error: Failed to download $url"
            # shellcheck disable=SC2164
            cd - > /dev/null
            rm -rf "$TEMP_DIR"
            continue
        fi
    fi

    echo "Extracting $filename..."
    tar -xzf "$filename"

    if [ ! -d "go" ]; then
        echo "Error: Extracted directory 'go' not found."
        # shellcheck disable=SC2164
        cd - >/dev/null
        rm -rf "$TEMP_DIR"
        continue
    fi

    TARGET_DIR="$INSTALL_DIR/$version"

    if [ -d "$TARGET_DIR" ]; then
        echo "Go version $version is already installed at $TARGET_DIR."
        # shellcheck disable=SC2164
        cd - >/dev/null
        rm -rf "$TEMP_DIR"
        continue
    fi

    mv go "$TARGET_DIR"
    echo "Go $version installed successfully at $TARGET_DIR."
    # shellcheck disable=SC2164
    cd - >/dev/null
    rm -rf "$TEMP_DIR"

done < "go-versions.txt"
