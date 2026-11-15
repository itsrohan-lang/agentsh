# Experimental & Archived Prototypes

This directory contains experimental, exploratory, and archived components that are not part of the core `agentsh` production runtime:

- **`stuntos/`**: Conceptual buildroot configuration and AWS Firecracker microVM launch script. Kept for historical reference.
- **`stuntbot/`**: Experimental Probot PR review integration bot.
- **`mac/`**: Prototype macOS EndpointSecurity framework interceptor. (Not compiled or attached by the CLI; macOS kernel interception requires special Developer ID entitlements).
- **`windows/`**: Prototype Windows Filtering Platform (WFP) driver interceptor. (Not compiled or attached by the CLI).
- **`simulate.sh`**: Legacy demo mock script.
- **`stunt-seccomp.json`**: Reference Docker seccomp filter.

These components are preserved for long-term reference and future research, but are not actively built or maintained.
