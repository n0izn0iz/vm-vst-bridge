package memconn

// #cgo LDFLAGS: -lsetupapi
/*
#include <windows.h>
#include <SetupAPI.h>

#include <stdio.h>
#include <stdlib.h>
#include <stdbool.h>
#include <malloc.h>
#include <initguid.h>
#include "./debug.h"
#include "membuf.h"

DEFINE_GUID (GUID_DEVINTERFACE_IVSHMEM,
    0xdf576976,0x569d,0x4672,0x95,0xa0,0xf5,0x7e,0x4e,0xa0,0xb2,0x10);
// {df576976-569d-4672-95a0-f57e4ea0b210}

typedef UINT16 IVSHMEM_PEERID;
typedef UINT64 IVSHMEM_SIZE;

#define IVSHMEM_CACHE_NONCACHED     0
#define IVSHMEM_CACHE_CACHED        1
#define IVSHMEM_CACHE_WRITECOMBINED 2


// This structure is for use with the IOCTL_IVSHMEM_REQUEST_MMAP IOCTL
typedef struct IVSHMEM_MMAP_CONFIG
{
    UINT8 cacheMode; // the caching mode of the mapping, see IVSHMEM_CACHE_* for options
}
IVSHMEM_MMAP_CONFIG, *PIVSHMEM_MMAP_CONFIG;

// This structure is for use with the IOCTL_IVSHMEM_REQUEST_MMAP IOCTL
typedef struct IVSHMEM_MMAP
{
    IVSHMEM_PEERID peerID;  // our peer id
    IVSHMEM_SIZE   size;    // the size of the memory region
    PVOID          ptr;     // pointer to the memory region
    UINT16         vectors; // the number of vectors available
}
IVSHMEM_MMAP, *PIVSHMEM_MMAP;

// This structure is for use with the IOCTL_IVSHMEM_RING_DOORBELL IOCTL
typedef struct IVSHMEM_RING
{
    IVSHMEM_PEERID peerID;  // the id of the peer to ring
    UINT16         vector;  // the doorbell to ring
}
IVSHMEM_RING, *PIVSHMEM_RING;

// This structure is for use with the IOCTL_IVSHMEM_REGISTER_EVENT IOCTL
// Please Note:
//   - The IVSHMEM driver has a hard limit of 32 events.
//   - Events that are singleShot are released after they have been set.
//   - At this time repeating events are only released when the driver device
//     handle is closed, closing the event handle doesn't release it from the
//     drivers list. While this won't cause a problem in the driver, it will
//     cause you to run out of event slots.
typedef struct IVSHMEM_EVENT
{
    UINT16  vector;     // the vector that triggers the event
    HANDLE  event;      // the event to trigger
    BOOLEAN singleShot; // set to TRUE if you want the driver to only trigger this event once
}
IVSHMEM_EVENT, *PIVSHMEM_EVENT;

#define IOCTL_IVSHMEM_REQUEST_PEERID CTL_CODE(FILE_DEVICE_UNKNOWN, 0x800, METHOD_BUFFERED, FILE_ANY_ACCESS)
#define IOCTL_IVSHMEM_REQUEST_SIZE   CTL_CODE(FILE_DEVICE_UNKNOWN, 0x801, METHOD_BUFFERED, FILE_ANY_ACCESS)
#define IOCTL_IVSHMEM_REQUEST_MMAP   CTL_CODE(FILE_DEVICE_UNKNOWN, 0x802, METHOD_BUFFERED, FILE_ANY_ACCESS)
#define IOCTL_IVSHMEM_RELEASE_MMAP   CTL_CODE(FILE_DEVICE_UNKNOWN, 0x803, METHOD_BUFFERED, FILE_ANY_ACCESS)
#define IOCTL_IVSHMEM_RING_DOORBELL  CTL_CODE(FILE_DEVICE_UNKNOWN, 0x804, METHOD_BUFFERED, FILE_ANY_ACCESS)
#define IOCTL_IVSHMEM_REGISTER_EVENT CTL_CODE(FILE_DEVICE_UNKNOWN, 0x805, METHOD_BUFFERED, FILE_ANY_ACCESS)

typedef struct IVSHMEM_SCREAM_HEADER
{
    UINT32 magic;
    UINT16 writeIdx;
    UINT8  offset;    //position of the 1st chunk
    UINT16 maxChunks; //how many chunks
    UINT32 chunkSize; //the size of a chunk
}
IVSHMEM_SCREAM_HEADER, *PIVSHMEM_SCREAM_HEADER;

HANDLE initialize()
{
    HDEVINFO deviceInfoSet;
    PSP_DEVICE_INTERFACE_DETAIL_DATA infData = NULL;
    SP_DEVICE_INTERFACE_DATA deviceInterfaceData;
    HANDLE handle = NULL;

    deviceInfoSet = SetupDiGetClassDevs(NULL, NULL, NULL, DIGCF_PRESENT | DIGCF_ALLCLASSES | DIGCF_DEVICEINTERFACE);
    ZeroMemory(&deviceInterfaceData, sizeof(SP_DEVICE_INTERFACE_DATA));
    deviceInterfaceData.cbSize = sizeof(SP_DEVICE_INTERFACE_DATA);

    while (true)
    {
        if (SetupDiEnumDeviceInterfaces(deviceInfoSet, NULL, &GUID_DEVINTERFACE_IVSHMEM, 0, &deviceInterfaceData) == FALSE)
        {
            DWORD error = GetLastError();
            if (error == ERROR_NO_MORE_ITEMS)
            {
                DEBUG_ERROR("Unable to enumerate the device, is it attached?");
                break;
            }

            DEBUG_ERROR("SetupDiEnumDeviceInterfaces failed");
            break;
        }

        DWORD reqSize = 0;
        SetupDiGetDeviceInterfaceDetail(deviceInfoSet, &deviceInterfaceData, NULL, 0, &reqSize, NULL);
        if (!reqSize)
        {
            DEBUG_ERROR("SetupDiGetDeviceInterfaceDetail");
            break;
        }

        infData = malloc(reqSize);
        ZeroMemory(infData, reqSize);
        infData->cbSize = sizeof(SP_DEVICE_INTERFACE_DETAIL_DATA);
        if (!SetupDiGetDeviceInterfaceDetail(deviceInfoSet, &deviceInterfaceData, infData, reqSize, NULL, NULL))
        {
            DEBUG_ERROR("SetupDiGetDeviceInterfaceDetail");
            break;
        }

        handle = CreateFile(infData->DevicePath, 0, 0, NULL, OPEN_EXISTING, 0, 0);
        if (handle == INVALID_HANDLE_VALUE)
        {
            DEBUG_ERROR("CreateFile returned INVALID_HANDLE_VALUE");
            handle = NULL;
            break;
        }

        break;
    }

    if (infData)
        free(infData);

    SetupDiDestroyDeviceInfoList(deviceInfoSet);
    return handle;
}

UINT64 getSize(HANDLE handle)
{
  if (handle == NULL)
    return 0;

  IVSHMEM_SIZE size;
  if (!DeviceIoControl(handle, IOCTL_IVSHMEM_REQUEST_SIZE, NULL, 0, &size, sizeof(IVSHMEM_SIZE), NULL, NULL))
  {
    DEBUG_ERROR("DeviceIoControl Failed: %d", (int)GetLastError());
    return 0;
  }

  return size;
}

bool getMap(HANDLE handle, IVSHMEM_MMAP* mapPtr)
{
  if (handle == NULL)
    return false;

// this if define can be removed later once everyone is un the latest version
// old versions of the IVSHMEM driver ignore the input argument, as such this
// is completely backwards compatible
#if defined(IVSHMEM_CACHE_WRITECOMBINED)
  IVSHMEM_MMAP_CONFIG config;
  config.cacheMode = IVSHMEM_CACHE_WRITECOMBINED;
#endif

  IVSHMEM_MMAP map;
  ZeroMemory(&map, sizeof(IVSHMEM_MMAP));
  if (!DeviceIoControl(
    handle,
    IOCTL_IVSHMEM_REQUEST_MMAP,
#if defined(IVSHMEM_CACHE_WRITECOMBINED)
    &config, sizeof(IVSHMEM_MMAP_CONFIG),
#else
    NULL   , 0,
#endif
    &map   , sizeof(IVSHMEM_MMAP       ),
    NULL, NULL))
  {
    DEBUG_ERROR("DeviceIoControl Failed: %d", (int)GetLastError());
    return false;
  }

  *mapPtr = map;
  ZeroMemory(map.ptr, getSize(handle));
  return true;
}



void readBytes(membuf* mb, void* dst, uint64_t dstOffset, uint64_t start, size_t len) {
	memcpy(((char*)dst) + dstOffset, (&mb->data) + start, len);
}

void writeBytes(membuf* mb, void* src, uint64_t srcOffset, uint64_t start, size_t len) {
	memcpy((&mb->data) + start, ((char*)src) + srcOffset, len);
}
*/
import "C"
import (
	"fmt"
	"unsafe"

	"go.uber.org/zap"
)

func initShmem(path string, size *C.size_t, err *C.int) *C.membuf {
	*err = 0

	handle := C.initialize()
	if handle == C.HANDLE(C.NULL) {
		panic("failed to initialize ivshmem")
	}

	var m C.IVSHMEM_MMAP
	gotMap := C.getMap(handle, &m)
	if !gotMap {
		panic("failed to get ivshmem map")
	}

	*size = m.size

	return (*C.membuf)(m.ptr)
}

func readBytes(mb *C.membuf, dst unsafe.Pointer, dstOffset C.uint64_t, start C.uint64_t, len C.size_t) {
	C.readBytes(mb, dst, dstOffset, start, len)
}

func writeBytes(mb *C.membuf, src unsafe.Pointer, srcOffset C.uint64_t, start C.uint64_t, len C.size_t) {
	C.writeBytes(mb, src, srcOffset, start, len)
}

func newMembuf(shmemPath string, ringSize int, offset int, logger *zap.Logger) (*C.membuf, C.size_t) {
	if len(shmemPath) <= 0 || offset < 0 || ringSize < 4 || ringSize%2 != 0 {
		panic(fmt.Sprint("invalid parameter(s): shmemPath: ", shmemPath, ", ringSize: ", ringSize, ", offset: ", offset))
	}

	var ret C.int
	var memSize C.size_t
	mem := initShmem(shmemPath, &memSize, &ret)
	if ret != 0 {
		panic(fmt.Sprint("failed to init shmem: ret=", ret))
	}
	logger.Debug("ivshmem:",
		zap.Int("memSizeMiB", int(memSize/1024/1024)),
		zap.Int("offsetB", offset),
		zap.Int("ringSizeB", ringSize),
	)

	// FIXME: compute real maximum
	if C.size_t(ringSize) > (memSize/2)-C.size_t(offset) {
		panic("ring too big for ivshmem")
	}

	return mem, memSize
}
