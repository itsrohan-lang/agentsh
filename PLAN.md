# 🗺 agentsh Roadmap

Status of the project, honestly stated. A box is checked only when the feature works
end to end.

---

## Phase 1: Container isolation

**Goal:** one command to run an agent in a restricted container.

* [x] CLI project structure (Go + cobra) — `agentsh` (aliases: `ash`, `sd`)
* [x] `agentsh init` generates `.agentsh.yaml` (also reads `.stuntdouble.yaml`)
* [x] `agentsh run <agent>` spawns the agent and pipes I/O
* [x] `--cap-drop=ALL`, memory and CPU limits, scoped workspace mount
* [x] Zero-Trust API Key substitution proxy (`ZeroTrustProxy`) on host egress
* [x] Enterprise AI Data Loss Prevention (`agentsh dlp scan`) for PII & secret inspection
* [x] Runaway Agent Guardrails (`--max-duration` timeout cap)
* [ ] Linux `cgroup_skb/egress` filter attaching to container cgroup paths
* [ ] Native macOS EndpointSecurity & Windows WFP kernel interceptors (prototypes archived in `experimental/`)

## Phase 2: Mocking

**Goal:** let agents exercise code paths that hit external services, safely.

* [x] Keploy sidecar attached to the agent's network namespace
* [x] `agentsh record` to capture mocks
* [x] `/api/keploy/mock` returns a canned success payload
* [x] Synthetic Auto-Mocking Engine (`agentsh mock generate` and `agentsh mock serve`)

## Phase 3: Telemetry & UX

* [x] Local run counter and `agentsh stats`
* [x] `agentsh monitor` terminal view of the control plane
* [x] Dashboard fed entirely by real API responses
* [x] Named workspace checkpoints (`agentsh checkpoint save/restore/list`)
* [x] Interactive Diff Inspector in Dashboard

## Phase 4: Control plane

* [x] Go service aggregating run counts, with an SQLite audit log
* [x] Bearer-token auth on all endpoints; loopback bind; single-origin CORS
* [x] Policy document served over REST and GraphQL
* [x] Cryptographic HMAC-SHA256 Tamper-Evident Audit Ledger (`/api/audit/verify`)

## Phase 5: Ecosystem

* [x] CI workflow generator (`agentsh ci`)
* [x] Automated CI Security Verification Gate (`agentsh verify`)
* [x] VS Code extension (`vscode-extension/`) with Bearer token auth & startup activation
* [x] Containerized test harness (`Dockerfile.test` and `scripts/test-in-container.sh`)

## Phase 6: Multi-agent

* [x] `agentsh swarm` spawns multiple agent containers on a shared Docker network
* [x] Automated Adversarial Red-Teaming Swarm (`agentsh swarm --redteam`)

## Archived / Non-production

The following experimental stubs have been consolidated into `experimental/` for reference:
- **Custom hypervisor / MicroVM ("StuntOS")**: archived in `experimental/stuntos/`
- **Native macOS ESF Interceptor**: archived in `experimental/mac/`
- **Native Windows WFP Interceptor**: archived in `experimental/windows/`
- **PR Review Bot**: archived in `experimental/stuntbot/`
- **Mock Simulation Script**: archived in `experimental/simulate.sh`

---

## Next Milestone for Future Maintainers

Implement `cgroup_skb/egress` filtering on Linux and wire it to the policy
document via `cilium/ebpf` bindings. See `MAINTENANCE.md`.
