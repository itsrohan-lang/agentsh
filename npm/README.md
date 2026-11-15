# agentsh

A restricted container shell & instant rollback environment for autonomous AI coding agents.

## Installation

```bash
npm install -g agentsh
```

The package automatically downloads the appropriate native binary for your platform (Linux, macOS, Windows) during installation, with cryptographic SHA-256 checksum verification.

## Usage

After installation, the `agentsh`, `ash`, `sd`, and `stuntdouble` commands are available:

```bash
# Run an agent in a sandboxed container
agentsh run claude --allow-unenforced-network
# or with short alias:
ash run claude --allow-unenforced-network

# Roll back agent workspace modifications
agentsh rewind

# Create and manage workspace checkpoints
agentsh checkpoint save my-checkpoint
agentsh checkpoint list
agentsh checkpoint restore my-checkpoint

# Show help
agentsh --help
```

## What it does

`agentsh` runs autonomous coding agents inside a restricted Docker container with:
- Dropped Linux capabilities (`--cap-drop=ALL`)
- CPU & memory resource limits
- Scoped working directory mount (`/workspace`)
- Zero-copy Git-based snapshot rollback (`agentsh rewind`)
- Zero-trust host egress proxy substituting dummy API keys

**Network egress filtering is NOT implemented.** Sandboxed agents can still reach the network. See [ENFORCEMENT.md](https://github.com/itsrohan-lang/agentsh/blob/main/docs/ENFORCEMENT.md) for what is and is not enforced.

## Platform support

Pre-built binaries are available for:
- Linux (x64, ARM64)
- macOS (x64, ARM64)
- Windows (x64)

If your platform isn't supported, [build from source](https://github.com/itsrohan-lang/agentsh).

## Documentation

Full documentation, deployment guides, and enforcement details:

**https://github.com/itsrohan-lang/agentsh**

## License

MIT
