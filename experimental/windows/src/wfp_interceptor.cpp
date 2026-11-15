#include "../include/wfp_interceptor.h"
#include <initguid.h>

// {4A7E2B10-3D9C-4E8A-9B1F-2C3D4E5F6A7B}
DEFINE_GUID(
    AGENTSH_CALLOUT_V4_GUID,
    0x4a7e2b10, 0x3d9c, 0x4e8a, 0x9b, 0x1f, 0x2c, 0x3d, 0x4e, 0x5f, 0x6a, 0x7b
);

// {4A7E2B10-3D9C-4E8A-9B1F-2C3D4E5F6A7C}
DEFINE_GUID(
    AGENTSH_CALLOUT_V6_GUID,
    0x4a7e2b10, 0x3d9c, 0x4e8a, 0x9b, 0x1f, 0x2c, 0x3d, 0x4e, 0x5f, 0x6a, 0x7c
);

// Global Driver State
static PDEVICE_OBJECT g_DeviceObject = NULL;
static UINT32 g_CalloutIdV4 = 0;
static UINT32 g_CalloutIdV6 = 0;
static UINT64 g_TotalDrops = 0;

static UINT16 g_BlockedPorts[MAX_BLOCKED_PORTS] = { POSTGRES_PORT, MONGO_PORT, MYSQL_PORT, REDIS_PORT };
static ULONG g_BlockedPortsCount = 4;
static KSPIN_LOCK g_ConfigLock;

static BOOLEAN IsPortBlocked(UINT16 port) {
    KLOCK_QUEUE_HANDLE lockHandle;
    KeAcquireInStackQueuedSpinLock(&g_ConfigLock, &lockHandle);
    for (ULONG i = 0; i < g_BlockedPortsCount; i++) {
        if (g_BlockedPorts[i] == port) {
            KeReleaseInStackQueuedSpinLock(&lockHandle);
            return TRUE;
        }
    }
    KeReleaseInStackQueuedSpinLock(&lockHandle);
    return FALSE;
}

VOID AgentshCalloutClassifyV4(
    const FWPS_INCOMING_VALUES0* inFixedValues,
    const FWPS_INCOMING_METADATA_VALUES0* inMetaValues,
    VOID* layerData,
    const VOID* classifyContext,
    const FWPS_FILTER1* filter,
    UINT64 flowContext,
    FWPS_CLASSIFY_OUT0* classifyOut
) {
    UNREFERENCED_PARAMETER(inMetaValues);
    UNREFERENCED_PARAMETER(layerData);
    UNREFERENCED_PARAMETER(classifyContext);
    UNREFERENCED_PARAMETER(filter);
    UNREFERENCED_PARAMETER(flowContext);

    if (inFixedValues->layerId != FWPS_LAYER_ALE_AUTH_CONNECT_V4) {
        classifyOut->actionType = FWP_ACTION_PERMIT;
        return;
    }

    UINT16 remotePort = inFixedValues->incomingValue[FWPS_FIELD_ALE_AUTH_CONNECT_V4_IP_REMOTE_PORT].value.uint16;

    if (IsPortBlocked(remotePort)) {
        classifyOut->actionType = FWP_ACTION_BLOCK;
        classifyOut->rights &= ~FWPS_RIGHT_ACTION_VALID;
        InterlockedIncrement64((LONG64*)&g_TotalDrops);
        KdPrint(("[agentsh WFP] 🚨 BLOCKED IPv4 outbound connection to port %d\n", remotePort));
        return;
    }

    classifyOut->actionType = FWP_ACTION_PERMIT;
}

VOID AgentshCalloutClassifyV6(
    const FWPS_INCOMING_VALUES0* inFixedValues,
    const FWPS_INCOMING_METADATA_VALUES0* inMetaValues,
    VOID* layerData,
    const VOID* classifyContext,
    const FWPS_FILTER1* filter,
    UINT64 flowContext,
    FWPS_CLASSIFY_OUT0* classifyOut
) {
    UNREFERENCED_PARAMETER(inMetaValues);
    UNREFERENCED_PARAMETER(layerData);
    UNREFERENCED_PARAMETER(classifyContext);
    UNREFERENCED_PARAMETER(filter);
    UNREFERENCED_PARAMETER(flowContext);

    if (inFixedValues->layerId != FWPS_LAYER_ALE_AUTH_CONNECT_V6) {
        classifyOut->actionType = FWP_ACTION_PERMIT;
        return;
    }

    UINT16 remotePort = inFixedValues->incomingValue[FWPS_FIELD_ALE_AUTH_CONNECT_V6_IP_REMOTE_PORT].value.uint16;

    if (IsPortBlocked(remotePort)) {
        classifyOut->actionType = FWP_ACTION_BLOCK;
        classifyOut->rights &= ~FWPS_RIGHT_ACTION_VALID;
        InterlockedIncrement64((LONG64*)&g_TotalDrops);
        KdPrint(("[agentsh WFP] 🚨 BLOCKED IPv6 outbound connection to port %d\n", remotePort));
        return;
    }

    classifyOut->actionType = FWP_ACTION_PERMIT;
}

NTSTATUS AgentshCalloutNotify(
    FWPS_CALLOUT_NOTIFY_TYPE notifyType,
    const GUID* filterKey,
    FWPS_FILTER1* filter
) {
    UNREFERENCED_PARAMETER(notifyType);
    UNREFERENCED_PARAMETER(filterKey);
    UNREFERENCED_PARAMETER(filter);
    return STATUS_SUCCESS;
}

NTSTATUS DispatchCreateClose(PDEVICE_OBJECT DeviceObject, PIRP Irp) {
    UNREFERENCED_PARAMETER(DeviceObject);
    Irp->IoStatus.Status = STATUS_SUCCESS;
    Irp->IoStatus.Information = 0;
    IoCompleteRequest(Irp, IO_NO_INCREMENT);
    return STATUS_SUCCESS;
}

NTSTATUS DispatchDeviceControl(PDEVICE_OBJECT DeviceObject, PIRP Irp) {
    UNREFERENCED_PARAMETER(DeviceObject);
    PIO_STACK_LOCATION irpSp = IoGetCurrentIrpStackLocation(Irp);
    NTSTATUS status = STATUS_SUCCESS;
    ULONG_PTR bytesReturned = 0;

    switch (irpSp->Parameters.DeviceIoControl.IoControlCode) {
    case IOCTL_AGENTSH_GET_STATS:
        if (irpSp->Parameters.DeviceIoControl.OutputBufferLength >= sizeof(AGENTSH_STATS)) {
            PAGENTSH_STATS stats = (PAGENTSH_STATS)Irp->AssociatedIrp.SystemBuffer;
            stats->TotalDrops = (UINT64)InterlockedCompareExchange64((LONG64*)&g_TotalDrops, 0, 0);
            bytesReturned = sizeof(AGENTSH_STATS);
        } else {
            status = STATUS_BUFFER_TOO_SMALL;
        }
        break;

    case IOCTL_AGENTSH_RESET_STATS:
        InterlockedExchange64((LONG64*)&g_TotalDrops, 0);
        bytesReturned = 0;
        break;

    case IOCTL_AGENTSH_SET_BLOCKED_PORTS: {
        ULONG inLen = irpSp->Parameters.DeviceIoControl.InputBufferLength;
        if (inLen > 0 && inLen <= (MAX_BLOCKED_PORTS * sizeof(UINT16))) {
            UINT16* ports = (UINT16*)Irp->AssociatedIrp.SystemBuffer;
            ULONG count = inLen / sizeof(UINT16);

            KLOCK_QUEUE_HANDLE lockHandle;
            KeAcquireInStackQueuedSpinLock(&g_ConfigLock, &lockHandle);
            RtlCopyMemory(g_BlockedPorts, ports, inLen);
            g_BlockedPortsCount = count;
            KeReleaseInStackQueuedSpinLock(&lockHandle);

            bytesReturned = 0;
        } else {
            status = STATUS_INVALID_PARAMETER;
        }
        break;
    }

    default:
        status = STATUS_INVALID_DEVICE_REQUEST;
        break;
    }

    Irp->IoStatus.Status = status;
    Irp->IoStatus.Information = bytesReturned;
    IoCompleteRequest(Irp, IO_NO_INCREMENT);
    return status;
}

extern "C" NTSTATUS DriverEntry(PDRIVER_OBJECT DriverObject, PUNICODE_STRING RegistryPath) {
    UNREFERENCED_PARAMETER(RegistryPath);
    KdPrint(("[agentsh WFP] Initializing agentsh Zero-Trust Kernel Driver...\n"));

    KeInitializeSpinLock(&g_ConfigLock);

    UNICODE_STRING devName, dosName;
    RtlInitUnicodeString(&devName, AGENTSH_DEVICE_NAME);
    RtlInitUnicodeString(&dosName, AGENTSH_DOS_DEVICE_NAME);

    NTSTATUS status = IoCreateDevice(
        DriverObject,
        0,
        &devName,
        FILE_DEVICE_NETWORK,
        FILE_DEVICE_SECURE_OPEN,
        FALSE,
        &g_DeviceObject
    );
    if (!NT_SUCCESS(status)) {
        return status;
    }

    status = IoCreateSymbolicLink(&dosName, &devName);
    if (!NT_SUCCESS(status)) {
        IoDeleteDevice(g_DeviceObject);
        return status;
    }

    DriverObject->DriverUnload = DriverUnload;
    DriverObject->MajorFunction[IRP_MJ_CREATE] = DispatchCreateClose;
    DriverObject->MajorFunction[IRP_MJ_CLOSE] = DispatchCreateClose;
    DriverObject->MajorFunction[IRP_MJ_DEVICE_CONTROL] = DispatchDeviceControl;

    // Register IPv4 Callout
    FWPS_CALLOUT0 calloutV4 = { 0 };
    calloutV4.calloutKey = AGENTSH_CALLOUT_V4_GUID;
    calloutV4.classifyFn = AgentshCalloutClassifyV4;
    calloutV4.notifyFn = AgentshCalloutNotify;

    status = FwpsCalloutRegister0(DriverObject, &calloutV4, &g_CalloutIdV4);
    if (!NT_SUCCESS(status)) {
        IoDeleteSymbolicLink(&dosName);
        IoDeleteDevice(g_DeviceObject);
        return status;
    }

    // Register IPv6 Callout
    FWPS_CALLOUT0 calloutV6 = { 0 };
    calloutV6.calloutKey = AGENTSH_CALLOUT_V6_GUID;
    calloutV6.classifyFn = AgentshCalloutClassifyV6;
    calloutV6.notifyFn = AgentshCalloutNotify;

    status = FwpsCalloutRegister0(DriverObject, &calloutV6, &g_CalloutIdV6);
    if (!NT_SUCCESS(status)) {
        FwpsCalloutUnregisterById0(g_CalloutIdV4);
        IoDeleteSymbolicLink(&dosName);
        IoDeleteDevice(g_DeviceObject);
        return status;
    }

    KdPrint(("[agentsh WFP] Active! IPv4 & IPv6 database ports are now natively blackholed.\n"));
    return STATUS_SUCCESS;
}

extern "C" VOID DriverUnload(PDRIVER_OBJECT DriverObject) {
    UNREFERENCED_PARAMETER(DriverObject);
    KdPrint(("[agentsh WFP] Unloading agentsh Kernel Driver...\n"));

    if (g_CalloutIdV6 != 0) {
        FwpsCalloutUnregisterById0(g_CalloutIdV6);
    }
    if (g_CalloutIdV4 != 0) {
        FwpsCalloutUnregisterById0(g_CalloutIdV4);
    }

    UNICODE_STRING dosName;
    RtlInitUnicodeString(&dosName, AGENTSH_DOS_DEVICE_NAME);
    IoDeleteSymbolicLink(&dosName);

    if (g_DeviceObject != NULL) {
        IoDeleteDevice(g_DeviceObject);
    }
}
