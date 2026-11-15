# agentsh WFP Kernel Driver & Service (Windows)

Because eBPF is native to Linux, `agentsh` uses the **Windows Filtering Platform (WFP)** to achieve native kernel-level network interception on Windows hosts.

## Architecture
- **`wfp_interceptor.cpp`**: A native Windows NT Kernel-mode driver (`AgentshWFP.sys`). It hooks into both `FWPS_LAYER_ALE_AUTH_CONNECT_V4` and `FWPS_LAYER_ALE_AUTH_CONNECT_V6`. When an AI agent attempts an outbound TCP connection, the NT kernel evaluates the packet against the driver's dynamic blocked-ports registry. Unauthorized ports (e.g. 5432, 27017, 3306, 6379) are blocked (`FWP_ACTION_BLOCK`) at the kernel layer before packets ever reach the network.
- **`wfp_service.cpp`**: A Windows userland service that bridges `\\.\AgentshWFP` device IOCTLs with a named pipe `\\.\pipe\AgentshWFP` consumed by `agentsh.exe`.

## Compilation & Installation
*Requires MSVC (Visual Studio 2022) with C++ Desktop Development and Windows Driver Kit (WDK).*

1. **Compile the Driver**:
   ```cmd
   cl.exe /kernel /I"include" /c src\wfp_interceptor.cpp
   link.exe /subsystem:native /driver /entry:DriverEntry wfp_interceptor.obj fwpuclnt.lib fwpkclnt.lib /out:AgentshWFP.sys
   ```

2. **Compile the Userland Service**:
   ```cmd
   cl.exe src\wfp_service.cpp /Fe:agentsh-wfp-service.exe
   ```

3. **Install & Start the Driver** (Test Signing mode required for unsigned local testing: `bcdedit /set testsigning on`):
   ```cmd
   sc create AgentshWFP type= kernel binPath= "%CD%\AgentshWFP.sys"
   sc start AgentshWFP
   agentsh-wfp-service.exe
   ```
