// Package ebpf provides kernel-level network egress filtering for sandboxed
// agent containers and host environments.
//
// Enforcement architecture per platform:
//
// 1. Linux:
//   - Uses native eBPF (cgroup_skb/egress) programs attached directly to the
//     target container's cgroup v2 hierarchy via cilium/ebpf.
//   - Non-permitted destination ports (e.g. databases, internal services) are
//     dropped directly in the kernel network stack before packets hit the wire.
//   - Telemetry tracks physical packet drops via an in-kernel array map.
//
// 2. macOS:
//   - Uses Apple EndpointSecurity Framework (ESF) via experimental/mac/src/esf_interceptor.c.
//   - Intercepts Unix Domain Sockets and outbound connections, communicating with the CLI via IPC.
//
// 3. Windows:
//   - Uses Windows Filtering Platform (WFP) kernel callout driver via experimental/windows/src/wfp_interceptor.cpp.
//   - Intercepts FWPS_LAYER_ALE_AUTH_CONNECT_V4/V6 to enforce blocked ports at kernel level.
package ebpf

import "errors"

// ErrUnsupported reports that kernel-level network enforcement is unavailable on the current system,
// or that the required kernel daemon/driver has not been initialized.
var ErrUnsupported = errors.New("ebpf: kernel-level network enforcement is not active; see docs/ENFORCEMENT.md")
