# Experimental & Archived Prototypes

This directory contains experimental, exploratory, and archived components that are not part of the core `agentsh` production runtime:

- **`stuntos/`**: Conceptual buildroot configuration and AWS Firecracker microVM launch script. Kept for historical reference.
- **`stuntbot/`**: Experimental Probot PR review integration bot.
- **`mac/`**: Complete macOS native EndpointSecurity framework (ESF) interceptor (`esf_interceptor.c`) with IPC control socket (`/var/run/agentsh_esf.sock`), Makefile, and entitlement definition.
- **`windows/`**: Complete Windows Filtering Platform (WFP) kernel callout driver (`wfp_interceptor.cpp`) and userland controller service (`wfp_service.cpp`) with IOCTL port configuration and named pipe IPC.
- **`simulate.sh`**: Legacy demo mock script.
- **`stunt-seccomp.json`**: Reference Docker seccomp filter.

These components are preserved for long-term reference and future research, but are not actively built or maintained.
