#!/bin/bash
set -euo pipefail

echo "=========================================================="
echo "🛡️  agentsh Containerized Test Harness"
echo "=========================================================="
echo ""

echo "▶ 1. Running CLI unit tests..."
cd /workspace/cli
go test -v ./pkg/snapshot ./pkg/dlp ./pkg/guard ./pkg/mock ./pkg/api ./pkg/config ./cmd

echo ""
echo "▶ 2. Running Control Plane tests..."
cd /workspace/control-plane
go test -v ./...

echo ""
echo "▶ 3. Verifying binary build..."
cd /workspace/cli
go build -o /tmp/agentsh main.go
/tmp/agentsh --help > /dev/null
echo "✅ Binary builds and executes successfully."

echo ""
echo "▶ 4. Running Integration Test: Workspace Snapshot & Rewind in isolated sandbox..."
TEMP_WORKSPACE=$(mktemp -d)
cd "$TEMP_WORKSPACE"
git init -q
git config user.name "agentsh-test"
git config user.email "test@agentsh.local"

echo "original content" > app.txt
git add app.txt
git commit -qm "initial"

echo "untracked file before snapshot" > config.json
/tmp/agentsh run sh --allow-unenforced-network -c "echo 'agent created this' > rogue.sh; echo 'modified content' > app.txt" || true

# Test rewind
/tmp/agentsh rewind
if [ -f "rogue.sh" ]; then
    echo "❌ Error: rogue.sh was not cleaned by rewind!"
    exit 1
fi
if [ "$(cat app.txt)" != "original content" ]; then
    echo "❌ Error: app.txt was not restored!"
    exit 1
fi
echo "✅ Workspace snapshot and rewind verified."

echo ""
echo "=========================================================="
echo "🎉 ALL TESTS PASSED SUCCESSFULLY IN CONTAINER HARNESS"
echo "=========================================================="
