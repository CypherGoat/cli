#!/bin/bash
set -e

BIN_DIR="/usr/local/bin"
BINARY_NAME="cg-cli"
REMOTE_URL="https://github.com/CypherGoat/cli/releases/download/v1/cg-v1-linux-amd64"
TMP_DIR=$(mktemp -d)
TMP_FILE="${TMP_DIR}/${BINARY_NAME}"

echo "Installing CypherGoat CLI..."

echo "Downloading CypherGoat CLI..."
curl -fsSL "${REMOTE_URL}" -o "${TMP_FILE}"

chmod +x "${TMP_FILE}"

echo "Moving binary to ${BIN_DIR}..."
if [ -w "${BIN_DIR}" ]; then
    mv "${TMP_FILE}" "${BIN_DIR}/${BINARY_NAME}"

    rm -rf "${TMP_DIR}"

echo "CypherGoat CLI has been successfully installed!"
echo "Run '${BINARY_NAME}' to get started."
else
    echo "Elevated permissions required to install to ${BIN_DIR}"
    sudo mv "${TMP_FILE}" "${BIN_DIR}/${BINARY_NAME}"

        rm -rf "${TMP_DIR}"

fi

