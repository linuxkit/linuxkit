/*
 * Compatibility layer between Windows and Linux
 */
#ifdef _MSC_VER
#undef UNICODE
#define WIN32_LEAN_AND_MEAN
#include <windows.h>
#include <winsock2.h>
#include <ws2tcpip.h>
#include <hvsocket.h>

#pragma comment (lib, "Ws2_32.lib")
#pragma comment (lib, "Mswsock.lib")
#pragma comment (lib, "AdvApi32.lib")

#else
#include <errno.h>
#include <stdint.h>
#include <unistd.h>
#include <sys/socket.h>
#endif /* !_MSC_VER */

#ifdef _MSC_VER
typedef int socklen_t;
#endif

#ifndef _MSC_VER
/* Compat layer for Linux/Unix */
typedef int SOCKET;

#ifndef SOCKET_ERROR
#define SOCKET_ERROR -1
#endif

#ifndef INVALID_SOCKET
#define INVALID_SOCKET -1
#endif

#define closesocket(_fd) close(_fd)

/* Shutdown flags are different too */
#define SD_SEND    SHUT_WR
#define SD_RECEIVE SHUT_RD
#define SD_BOTH    SHUT_RDWR

#define __cdecl

/* GUID handling  */
typedef struct _GUID {
    uint32_t Data1;
    uint16_t Data2;
    uint16_t Data3;
    uint8_t  Data4[8];
} GUID;

#define DEFINE_GUID(name, l, w1, w2, b1, b2, b3, b4, b5, b6, b7, b8) \
    const GUID name = {l, w1, w2, {b1, b2,  b3,  b4,  b5,  b6,  b7,  b8}}


/* HV Socket definitions */
#define AF_HYPERV 43
#define HV_PROTOCOL_RAW 1

typedef struct _SOCKADDR_HV
{
    unsigned short Family;
    unsigned short Reserved;
    GUID VmId;
    GUID ServiceId;
} SOCKADDR_HV;

DEFINE_GUID(HV_GUID_ZERO,
    0x00000000, 0x0000, 0x0000, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00);
DEFINE_GUID(HV_GUID_BROADCAST,
    0xFFFFFFFF, 0xFFFF, 0xFFFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF);
DEFINE_GUID(HV_GUID_WILDCARD,
    0x00000000, 0x0000, 0x0000, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00);

DEFINE_GUID(HV_GUID_CHILDREN,
    0x90db8b89, 0x0d35, 0x4f79, 0x8c, 0xe9, 0x49, 0xea, 0x0a, 0xc8, 0xb7, 0xcd);
DEFINE_GUID(HV_GUID_LOOPBACK,
    0xe0e16197, 0xdd56, 0x4a10, 0x91, 0x95, 0x5e, 0xe7, 0xa1, 0x55, 0xa8, 0x38);
DEFINE_GUID(HV_GUID_PARENT,
    0xa42e7cda, 0xd03f, 0x480c, 0x9c, 0xc2, 0xa4, 0xde, 0x20, 0xab, 0xb8, 0x78);

#endif

/* Thread wrappers */
#ifdef _MSC_VER
typedef HANDLE THREAD_HANDLE;

static inline int thread_create(THREAD_HANDLE *t, void *(*f)(void *), void *arg)
{
    *t = CreateThread(NULL, 0, f, arg, 0, NULL);
    return 0;
}

static inline int thread_join(THREAD_HANDLE t)
{
    WaitForSingleObject(t, INFINITE);
    return 0;
}

#else
#include <pthread.h>

typedef pthread_t THREAD_HANDLE;

static inline int thread_create(THREAD_HANDLE *t, void *(*f)(void *), void *arg)
{
    return pthread_create(t, NULL, f, arg);
}

static inline int thread_join(THREAD_HANDLE t)
{
    return pthread_join(t, NULL);
}
#endif
