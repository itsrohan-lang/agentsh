# 🛠️ agentsh: Maintenance & LTS Guide

> **Target Audience:** Maintainers returning to this project after extended periods.

## Project Summary & Core Mission
`agentsh` (formerly StuntDouble) runs autonomous AI coding agents inside restricted Docker containers with:
- Dropped Linux capabilities (`--cap-drop=ALL`)
- Strict cgroup resource ceilings (`--memory=2g`, `--cpus=1.0`)
- Scoped working directory mount (`$PWD` mounted at `/workspace`)
- Zero-copy Git plumbing snapshots with instant restoration (`agentsh rewind` and `agentsh checkpoint`)
- Host-level zero-trust API credential substitution (`ZeroTrustProxy`)

## Quick Verification (Containerized)
To test the entire codebase without modifying your local system:
```bash
./scripts/test-in-container.sh
```
This builds an isolated Alpine Linux Docker container with Go and Git, executes all unit and integration tests, and destroys the container when finished.

## Repository Layout
- **`cli/`**: Go command-line tool (`agentsh`, aliases `ash`, `sd`, `stuntdouble`).
  - `cli/cmd/`: Cobra subcommands (`run`, `rewind`, `checkpoint`, `guard`, `dlp`, `mock`, `swarm`, `stats`, `monitor`, `serve`, `verify`, `init`).
  - `cli/pkg/snapshot/`: Git plumbing engine (`git read-tree`, `git write-tree`, `git restore`).
  - `cli/pkg/docker/`: Docker SDK integration for spawning isolated agent containers and Keploy sidecars.
  - `cli/pkg/proxy/`: `ZeroTrustProxy` substituting dummy keys for real API keys on egress.
  - `cli/pkg/guard/`: Indirect prompt injection and jailbreak detector.
  - `cli/pkg/dlp/`: Data loss prevention regex engine and secret redactor.
- **`control-plane/`**: Go telemetry, policy distribution, and HMAC-SHA256 audit ledger service on `127.0.0.1:4439`.
- **`dashboard/`**: Next.js 16 SOC mission control dashboard.
- **`docs/`**: Documentation and architectural specifications.
- **`experimental/`**: Archived prototypes (Firecracker microVM launcher, macOS ESF driver, Windows WFP driver, stuntbot). Kept for reference.
- **`npm/`**: Node.js wrapper package (`stuntdouble-sandbox-cli`) with SHA-256 verified binary downloader.

## Key Security Realities & Enforcement
1. **Container Isolation is Enforced:** Accidental damage, memory exhaustion, or runaway loops inside `/workspace` are contained and reversible with `agentsh rewind`.
2. **Network Egress is Open:** Kernel-level egress filtering (`cgroup_skb/egress`) requires Linux eBPF (`cli/pkg/ebpf/bpf_prog.c`). The CLI requires `--allow-unenforced-network` to run, preventing false containment assumptions.
3. **Audit Ledger:** Detects direct tampering with the SQLite database file, but clients holding `STUNTDOUBLE_TOKEN` self-report their actions.

## Building Releases
```bash
# Build CLI binary
make build-cli

# Run docker-compose SOC
make start-soc
```

## Adding Future Features (e.g. Linux eBPF)
When resuming kernel-level enforcement work:
1. Compile `cli/pkg/ebpf/bpf_prog.c` using `bpf2go` with clang and kernel headers.
2. Target the container's cgroup v2 path directly (not host root `/sys/fs/cgroup`).
3. Populate `blocked_ports` from the active policy.
4. Replace `ebpf.ErrUnsupported` with `link.AttachCgroup`.
