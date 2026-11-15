#include <windows.h>
#include <stdio.h>
#include <string.h>

#define AGENTSH_DEVICE_PATH L"\\\\.\\AgentshWFP"
#define AGENTSH_PIPE_NAME   L"\\\\.\\pipe\\AgentshWFP"

#define IOCTL_AGENTSH_SET_BLOCKED_PORTS CTL_CODE(FILE_DEVICE_NETWORK, 0x801, METHOD_BUFFERED, FILE_ANY_ACCESS)
#define IOCTL_AGENTSH_GET_STATS         CTL_CODE(FILE_DEVICE_NETWORK, 0x802, METHOD_BUFFERED, FILE_ANY_ACCESS)

typedef struct _AGENTSH_STATS {
    UINT64 TotalDrops;
} AGENTSH_STATS, *PAGENTSH_STATS;

int main() {
    printf("🛡️ [agentsh WFP Service] Connecting to kernel driver %ls...\n", AGENTSH_DEVICE_PATH);

    HANDLE hDevice = CreateFileW(
        AGENTSH_DEVICE_PATH,
        GENERIC_READ | GENERIC_WRITE,
        0,
        NULL,
        OPEN_EXISTING,
        FILE_ATTRIBUTE_NORMAL,
        NULL
    );

    if (hDevice == INVALID_HANDLE_VALUE) {
        fprintf(stderr, "❌ Failed to open kernel driver device. Is the AgentshWFP driver installed and running? Error: %lu\n", GetLastError());
        return 1;
    }

    printf("✅ Connected to kernel driver. Creating Named Pipe %ls for agentsh CLI...\n", AGENTSH_PIPE_NAME);

    while (1) {
        HANDLE hPipe = CreateNamedPipeW(
            AGENTSH_PIPE_NAME,
            PIPE_ACCESS_DUPLEX,
            PIPE_TYPE_BYTE | PIPE_READMODE_BYTE | PIPE_WAIT,
            PIPE_UNLIMITED_INSTANCES,
            1024,
            1024,
            0,
            NULL
        );

        if (hPipe == INVALID_HANDLE_VALUE) {
            fprintf(stderr, "❌ Failed to create named pipe. Error: %lu\n", GetLastError());
            break;
        }

        if (ConnectNamedPipe(hPipe, NULL) || GetLastError() == ERROR_PIPE_CONNECTED) {
            char buffer[1024];
            DWORD bytesRead = 0;

            if (ReadFile(hPipe, buffer, sizeof(buffer) - 1, &bytesRead, NULL) && bytesRead > 0) {
                buffer[bytesRead] = '\0';

                if (strstr(buffer, "get_stats")) {
                    AGENTSH_STATS stats = { 0 };
                    DWORD bytesReturned = 0;
                    DeviceIoControl(hDevice, IOCTL_AGENTSH_GET_STATS, NULL, 0, &stats, sizeof(stats), &bytesReturned, NULL);

                    char resp[128];
                    snprintf(resp, sizeof(resp), "{\"drop_count\": %llu}\n", (unsigned long long)stats.TotalDrops);
                    DWORD bytesWritten = 0;
                    WriteFile(hPipe, resp, (DWORD)strlen(resp), &bytesWritten, NULL);
                } else if (strstr(buffer, "set_blocked_ports")) {
                    // Set default ports
                    UINT16 defaultPorts[] = { 5432, 27017, 3306, 6379 };
                    DWORD bytesReturned = 0;
                    DeviceIoControl(hDevice, IOCTL_AGENTSH_SET_BLOCKED_PORTS, defaultPorts, sizeof(defaultPorts), NULL, 0, &bytesReturned, NULL);

                    const char* ack = "{\"status\":\"ok\"}\n";
                    DWORD bytesWritten = 0;
                    WriteFile(hPipe, ack, (DWORD)strlen(ack), &bytesWritten, NULL);
                }
            }
        }

        DisconnectNamedPipe(hPipe);
        CloseHandle(hPipe);
    }

    CloseHandle(hDevice);
    return 0;
}
