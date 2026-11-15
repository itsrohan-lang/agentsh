# agentsh macOS Native Endpoint Security (ESF) Interceptor

This directory contains the native macOS kernel interceptor driver for `agentsh`.

## Architecture
On macOS, Docker containers run inside a virtual machine (hypervisor), which means host-level sockets (such as local PostgreSQL `/tmp/.s.PGSQL.5432` or MySQL sockets) cannot be filtered by container eBPF alone. 

The `agentsh-esf` interceptor hooks directly into Apple's native **EndpointSecurity Framework (ESF)**. When an AI agent running natively or in an unenforced container attempts to connect to local database sockets or unauthorized ports, the macOS kernel pauses the thread and asks `agentsh-esf` for authorization. If the target is unauthorized, the kernel immediately denies the connection (`ES_AUTH_RESULT_DENY`).

## Building & Signing
Because EndpointSecurity operates in the macOS security perimeter, the binary requires root permissions and the Apple EndpointSecurity entitlement:

```bash
# 1. Compile the interceptor
make

# 2. Sign with entitlements (or with your Apple Developer ID)
make sign

# 3. Launch as root daemon
sudo ./bin/agentsh-esf
```

## IPC Control
The daemon exposes an IPC control socket at `/var/run/agentsh_esf.sock` where the `agentsh` CLI automatically communicates to query drop statistics and configure blocked destinations.
