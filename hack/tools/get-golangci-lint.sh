#!/bin/bash

set -euo pipefail

# Define the URL for downloading the golangci-lint archive
curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(pwd)/bin "$1"