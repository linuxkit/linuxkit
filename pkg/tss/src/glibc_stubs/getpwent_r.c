/*
 * getpwent_r stub
 *
 * This is not really reentrant... but then again, neither is getpwent_r
 * because getpwent_r is a GNU extension, and not posix compliant,
 * a program using getpwent_r() will fail on a system with POSIX-compliant libc,
 *   e.g. musl libc on Alpine
 *
 * This library simply stubs it through
 * It does nothing but:
 * 1. populate the pwbuf with the data
 * 2. populate pwbufp with the pointer to *pwbuf
 * 3. return correct error codes
 *
 * It was created to get trousers libtspi to work with POSIX-compliant musl libc
 * when that is fixed - https://sourceforge.net/p/trousers/bugs/211/ - this will
 * be unnecessary
 */

#include <errno.h>
#include <stddef.h>
#include <pwd.h>
#include <string.h>

struct passwd *pwp;
int getpwent_r(struct passwd *pwbuf, char *buf, size_t buflen, struct passwd **pwbufp)
{
         struct passwd *pw;
         // if NULL, we had an error, return the appropriate error code
         if ((pw = getpwent()) == NULL) {
                  return ERANGE;
         }
         // so really we should memcpy mot just the (struct passwd), but everything it points to as well
         // in practice, we just copy the (struct passwd) because this isn't really thread-safe anyways
         memcpy(pwbuf, pw, sizeof(*pw));
         *pwbufp = pwbuf;
         return 0;
}
