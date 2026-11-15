#pragma once

#include <ntddk.h>
#include <fwpsk.h>
#include <fwpmk.h>

#define AGENTSH_TAG 'hSgA'
#define AGENTSH_DEVICE_NAME      L"\\Device\\AgentshWFP"
#define AGENTSH_DOS_DEVICE_NAME  L"\\DosDevices\\AgentshWFP"

#define IOCTL_AGENTSH_SET_BLOCKED_PORTS CTL_CODE(FILE_DEVICE_NETWORK, 0x801, METHOD_BUFFERED, FILE_ANY_ACCESS)
#define IOCTL_AGENTSH_GET_STATS         CTL_CODE(FILE_DEVICE_NETWORK, 0x802, METHOD_BUFFERED, FILE_ANY_ACCESS)
#define IOCTL_AGENTSH_RESET_STATS       CTL_CODE(FILE_DEVICE_NETWORK, 0x803, METHOD_BUFFERED, FILE_ANY_ACCESS)

#define MAX_BLOCKED_PORTS 256

typedef struct _AGENTSH_STATS {
    UINT64 TotalDrops;
} AGENTSH_STATS, *PAGENTSH_STATS;

// Port Definitions for Destructive AI Targeting
#define POSTGRES_PORT 5432
#define MONGO_PORT 27017
#define MYSQL_PORT 3306
#define REDIS_PORT 6379

// Driver Entry & Unload Methods
extern "C" NTSTATUS DriverEntry(PDRIVER_OBJECT DriverObject, PUNICODE_STRING RegistryPath);
extern "C" VOID DriverUnload(PDRIVER_OBJECT DriverObject);

// WFP Callout Functions
VOID AgentshCalloutClassifyV4(
    const FWPS_INCOMING_VALUES0* inFixedValues,
    const FWPS_INCOMING_METADATA_VALUES0* inMetaValues,
    VOID* layerData,
    const VOID* classifyContext,
    const FWPS_FILTER1* filter,
    UINT64 flowContext,
    FWPS_CLASSIFY_OUT0* classifyOut
);

VOID AgentshCalloutClassifyV6(
    const FWPS_INCOMING_VALUES0* inFixedValues,
    const FWPS_INCOMING_METADATA_VALUES0* inMetaValues,
    VOID* layerData,
    const VOID* classifyContext,
    const FWPS_FILTER1* filter,
    UINT64 flowContext,
    FWPS_CLASSIFY_OUT0* classifyOut
);

NTSTATUS AgentshCalloutNotify(
    FWPS_CALLOUT_NOTIFY_TYPE notifyType,
    const GUID* filterKey,
    FWPS_FILTER1* filter
);
