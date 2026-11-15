#!/usr/bin/env bash
#
# Containerized Test Runner for agentsh
# Builds and runs all tests inside an isolated Docker container so the host system
# is never modified or affected.
#
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

echo "🐳 Building agentsh isolated test container..."
docker build -t agentsh-test-runner -f Dockerfile.test .

echo "🚀 Executing test suite inside container sandbox..."
docker run --rm agentsh-test-runner
