#ifndef GETPWENTR_H
#define GETPWENTR_H

#include <stddef.h> // For size_t
#include <pwd.h>    // For struct passwd

#ifdef __cplusplus
extern "C" {
#endif

int getpwent_r(struct passwd *pwbuf, char *buf, size_t buflen, struct passwd **pwbufp);

#ifdef __cplusplus
}
#endif

#endif // GETPWENTR_H