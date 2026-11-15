# 📚 agentsh Documentation Site

This is the Next.js frontend repository for the official `agentsh` documentation and landing site.

## 🚀 Getting Started

First, install dependencies and run the development server:

```bash
cd docs
npm install
npm run dev
```

Open [http://localhost:3000](http://localhost:3000) with your browser to see the result.

## 🛠️ Command Reference

The `agentsh` CLI (aliases: `ash`, `sd`, `stuntdouble`) runs AI coding agents in a restricted Docker container.

> Network egress filtering is **not implemented**. The sandbox provides container
> isolation only. See [ENFORCEMENT.md](./ENFORCEMENT.md) for the full boundary.

### `agentsh init`
Writes an `.agentsh.yaml` config file in the current directory.
- **Usage**: Run this at the root of the codebase where the agent will operate.

### `agentsh run <agent> [args...]`
The primary command. Runs the agent inside a Docker container with
`--cap-drop=ALL`, a 2g memory cap, a 1.0 CPU cap, and only the working directory
mounted at `/workspace`. Snapshots the workspace first so `agentsh rewind` can undo it.
- **Arguments**: `<agent>` — `claude`, `sh`, `bash`, or any npm package name.
  Anything after the agent name is forwarded to it verbatim.
- **Options**:
  - `--allow-unenforced-network`: Required. Acknowledges that egress filtering is
    not active. `agentsh run` refuses to start without it.
  - `--env, -e`: Base runtime image (default: `node:20-alpine`).
- **Example**: `agentsh run claude --allow-unenforced-network --env python:3.11-alpine`

### `agentsh rewind`
Restores the workspace to the snapshot taken at the last `agentsh run`.

### `agentsh checkpoint [save|restore|list]`
Manages named zero-copy workspace checkpoints.

### `agentsh guard prompt <target>`
Analyzes prompt strings or context files for indirect prompt injection vectors.

### `agentsh dlp scan <target>`
Scans and redacts sensitive API credentials and PII.

### `agentsh record <agent>`
Records database and API traffic with a Keploy sidecar to generate mocks.

### `agentsh stats`
Shows the local run count from `.agentsh.telemetry.json` / `.stuntdouble.telemetry.json`.
