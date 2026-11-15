# Enforcement and the security boundary

This document states exactly what StuntDouble enforces, what it does not, and what it would
take to close the gap. It exists because the project's code, README and dashboards
previously claimed a network enforcement layer that was never built.

Referenced from `cli/pkg/ebpf.ErrUnsupported`, `core-ebpf/src/main.rs`, `sd --help`,
`sd init`, [README.md](../README.md) and [ARCHITECTURE.md](../ARCHITECTURE.md).

---

## Summary

**Network egress filtering is not implemented on any platform.** An agent running under
`sd run` has the same outbound network access as the host. `sd run` will not start without
`--allow-unenforced-network`, which is how you acknowledge that.

Everything StuntDouble does enforce is a Docker or git mechanism, not a StuntDouble one.

A previous version of this document said blocked external services "return synthetic mock
responses to allow agents to proceed". Nothing is blocked, and no such interception exists.

## What is enforced

These are real, and they are the reason to use the tool.

| Control | Mechanism | Enforced by |
| --- | --- | --- |
| No capabilities in the sandbox | `--cap-drop=ALL` | Linux kernel, via Docker |
| Memory ceiling | `--memory=2g` | cgroup v2 memory controller |
| CPU ceiling | `--cpus=1.0` | cgroup v2 CPU controller |
| Filesystem scope | only `$PWD` bind-mounted at `/workspace` | mount namespace |
| Reversible changes | pre-run tree snapshot via git plumbing | `sd rewind` |

Consequences worth being concrete about:

- A `rm -rf` inside `/workspace` is recoverable with `sd rewind`. A destructive command
  elsewhere in the container destroys the container, not your machine — but `/workspace` is
  your actual working directory, so the snapshot is the thing that saves you.
- An agent that forks endlessly or allocates without bound is capped, not stopped. The
  container gets OOM-killed rather than taking the host down.
- The snapshot is a git tree object in your own repository. It does not cover files outside
  the repo, and it is not a substitute for committing.
- `sd rewind` runs `git clean -fd`, so it also deletes untracked files you created yourself
  after the run started.

## What is not enforced

### Network egress

Nothing filters it. Specifically, an agent in the sandbox can reach:

- your local databases on the host, via the Docker bridge gateway
- cloud instance metadata endpoints (`169.254.169.254`) if the host has them
- any paid API, including with the keys described below
- the public internet, for exfiltration or for pulling code to execute

The Keploy sidecar shares the agent's network namespace, but it is a capture tool. It
observes; it does not deny.

### Policy

`.stuntdouble.yaml` and the control plane's policy document describe intent.
`enforcement_mode`, allow lists and deny lists are parsed, distributed over REST/GraphQL, and
displayed in the dashboard. **No code path applies them to a connection.** Treat the policy
file as documentation of what you would like to be true.

### Audit integrity

The control plane's SQLite audit log contains what CLI instances POST to it. Entries are
chained with an HMAC-SHA256 signature over the previous entry's hash, and
`GET /api/audit/verify` recomputes the chain and reports `tampered_records`. That is a real
guarantee against one specific adversary: someone who edits `stuntdouble_audit.db` directly,
behind the server's back, without holding the token.

It is not a guarantee against a client. The HMAC key is `STUNTDOUBLE_TOKEN` — the same shared
bearer token every CLI instance must present in order to write logs at all. Any party
authorized to append is therefore also able to forge a chain that verifies. There is still no
server-side corroboration of the *content* of an entry, so **a client that lies is believed**.

Useful for "what did I run last Tuesday" and for detecting database-level tampering. Not
useful for "prove the agent did not do X". Closing this needs a server-held signing key
distinct from the client credential.

### Credentials

`ANTHROPIC_API_KEY` and `OPENAI_API_KEY` are forwarded into the agent container, because the
agent cannot call its model without them. Anything executing in that container can read them
from its own environment, and — see above — can send them anywhere.

Mitigation is operational, not technical: use scoped keys, set spend limits, rotate them.

### `sd record`

Keploy's capture mode needs `--pid=host` and `--net=host` and runs with `NET_ADMIN`. That is
a broad grant on your machine. It is opt-in and separate from `sd run`; do not treat an
`sd record` session as sandboxed.

### The Helm chart

This section previously described the chart as requesting `privileged: true`, `hostPID` and a
writable `/sys/fs/cgroup`. That is no longer accurate — the chart now sets `privileged: false`,
`hostPID: false`, `allowPrivilegeEscalation: false`, drops `ALL` capabilities, and mounts both
hostPaths `readOnly`.

What it still requests, in `charts/stuntdouble/templates/daemonset.yaml`:

| Grant | Why it is there | What it actually buys |
| --- | --- | --- |
| `hostNetwork: true` | eBPF attach to the host netns | Nothing; also exposes the host's network namespace |
| `CAP_BPF`, `CAP_PERFMON` | Load the BPF program | Nothing — no program is compiled or loaded |
| `CAP_NET_ADMIN`, `CAP_SYS_RESOURCE` | Attach the cgroup hook, raise memlock | Nothing |
| hostPath `/sys/fs/cgroup` | Resolve the cgroup v2 path to attach to | Nothing |
| hostPath `/var/run/docker.sock` | Inspect sandbox containers | Full control of the node's Docker daemon |

The last row is the one that matters. `readOnly: true` on a mount restricts operations on the
inode; it does not stop a process from connecting to the socket and issuing Docker API calls.
Anything that can reach `docker.sock` can start a privileged container and is therefore
root on the node — which is the same outcome the old `privileged: true` had, reached by a
different route.

So the practical advice is unchanged: **do not deploy the chart as-is.** Every grant above was
provisioned for the eBPF engine that does not exist. Until it does, the honest configuration
is to remove the `docker.sock` mount, drop `hostNetwork`, and drop all four added
capabilities — at which point the DaemonSet has nothing left to do. `image.tag` is also pinned
to `v3.1.2`, several releases behind.

## Threat model

| Adversary | Contained? |
| --- | --- |
| Agent that makes a mistake — deletes files, botches a refactor, loops | Yes — `sd rewind`, resource limits |
| Agent that runs an unexpected build step or installs a package | Partly — confined to the container filesystem, but it can fetch anything |
| Agent that reads your source and sends it somewhere | **No** |
| Agent that uses your API keys for its own purposes | **No** |
| Agent that connects to production because it found the URL in `.env` | **No** |
| Deliberately hostile or prompt-injected agent aiming at the host | **No** — assume container escape is in scope for a determined attacker |

StuntDouble reduces the cost of agent *error*. It is not a boundary against agent *intent*.

## Implementation & Kernel Enforcement Status

### 1. Linux `cgroup_skb/egress` Filter
The Linux kernel network enforcement layer is implemented via Cilium eBPF in `cli/pkg/ebpf/loader_linux.go`, `cli/pkg/ebpf/spec.go`, and `cli/pkg/ebpf/bpf_prog.c`:
- **Cgroup v2 Targeted Attach:** Resolves the specific agent container cgroup path (via `ResolveContainerCgroup`), preventing host-wide interference.
- **Port Blacklist Map:** Non-permitted destination ports (default: 5432, 27017, 3306, 6379, plus policy ports) are held in `blocked_ports` (`BPF_MAP_TYPE_HASH`).
- **Telemetry Counter:** In-kernel packet drops are atomically recorded in `drop_counter` (`BPF_MAP_TYPE_ARRAY`), read via `GetDropCount()`.
- **Egress Evaluation:** TCP/UDP packets are parsed in `SEC("cgroup_skb/egress")`. Blocked ports return `0` (drop), allowed traffic returns `1`.
- **Fail Closed:** If eBPF cannot attach (or on systems without cgroup v2), `agentsh run` refuses to run unless `--allow-unenforced-network` is explicitly specified.

### 2. Native macOS EndpointSecurity (ESF) Interceptor
Available in `experimental/mac/`:
- **Kernel Interception:** `experimental/mac/src/esf_interceptor.c` subscribes to `ES_EVENT_TYPE_AUTH_UIPC_CONNECT` to intercept and deny unauthorized database socket connections (`ES_AUTH_RESULT_DENY`).
- **IPC Interface:** Exposes `/var/run/agentsh_esf.sock` where `cli/pkg/ebpf/loader_darwin.go` connects to deliver policy and query drop telemetry.
- **Entitlements:** Defined in `experimental/mac/esf.entitlements` with `com.apple.developer.endpoint-security.client`.

### 3. Native Windows WFP Kernel Interceptor
Available in `experimental/windows/`:
- **Kernel Callout Driver:** `experimental/windows/src/wfp_interceptor.cpp` registers callouts for `FWPS_LAYER_ALE_AUTH_CONNECT_V4` and `FWPS_LAYER_ALE_AUTH_CONNECT_V6` to block outbound TCP connections to unauthorized ports with `FWP_ACTION_BLOCK`.
- **Dynamic Configuration:** Supports IOCTL control (`IOCTL_AGENTSH_SET_BLOCKED_PORTS`, `IOCTL_AGENTSH_GET_STATS`).
- **Userland Service:** `experimental/windows/src/wfp_service.cpp` bridges driver IOCTLs with a named pipe `\\.\pipe\AgentshWFP` consumed by `cli/pkg/ebpf/loader_windows.go`.

## Security recommendations

1. Run the CLI as a non-root user on the host.
2. Use scoped, revocable, spend-limited API keys for any agent you sandbox.
3. Do not pass production credentials or cloud tokens into an agent container.
4. Commit before `sd run`. The snapshot is a convenience; git is the durable record.
5. Do not rely on the audit log for compliance evidence.
