/* AF_HYPERV definitions and utilities */

#include <errno.h>
#include <stdint.h>
#include <unistd.h>
#include <sys/socket.h>

/* GUID handling  */
typedef struct _GUID {
	uint32_t Data1;
	uint16_t Data2;
	uint16_t Data3;
	uint8_t Data4[8];
} GUID;

#define DEFINE_GUID(name, l, w1, w2, b1, b2, b3, b4, b5, b6, b7, b8) \
    const GUID name = {l, w1, w2, {b1, b2,  b3,  b4,  b5,  b6,  b7,  b8}}

/* Helper macros for parsing/printing GUIDs */
#define GUID_FMT "%08x-%04hx-%04hx-%02x%02x-%02x%02x%02x%02x%02x%02x"
#define GUID_ARGS(_g)                                               \
    (_g).Data1, (_g).Data2, (_g).Data3,                             \
    (_g).Data4[0], (_g).Data4[1], (_g).Data4[2], (_g).Data4[3],     \
    (_g).Data4[4], (_g).Data4[5], (_g).Data4[6], (_g).Data4[7]
#define GUID_SARGS(_g)                                              \
    &(_g).Data1, &(_g).Data2, &(_g).Data3,                          \
    &(_g).Data4[0], &(_g).Data4[1], &(_g).Data4[2], &(_g).Data4[3], \
    &(_g).Data4[4], &(_g).Data4[5], &(_g).Data4[6], &(_g).Data4[7]

extern int parseguid(const char *s, GUID *g);

/* HV Socket definitions */
#define AF_HYPERV 43
#define HV_PROTOCOL_RAW 1

typedef struct _SOCKADDR_HV {
	unsigned short Family;
	unsigned short Reserved;
	GUID VmId;
	GUID ServiceId;
} SOCKADDR_HV;

extern const GUID HV_GUID_ZERO;
extern const GUID HV_GUID_BROADCAST;
extern const GUID HV_GUID_WILDCARD;
extern const GUID HV_GUID_CHILDREN;
extern const GUID HV_GUID_LOOPBACK;
extern const GUID HV_GUID_PARENT;
