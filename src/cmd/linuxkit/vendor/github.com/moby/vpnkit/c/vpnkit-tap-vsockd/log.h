#include <stdlib.h>
#include <stdio.h>

extern int verbose;
#define ERROR(...)                                                      \
    do {                                                                \
        fprintf(stderr, __VA_ARGS__);                                   \
        fprintf(stderr, "\n");                                          \
    } while (0)
#define INFO(...)                                                       \
    do {                                                                \
        fprintf(stderr, __VA_ARGS__);                                   \
        fprintf(stderr, "\n");                                          \
    } while (0)
#define DBG(...)                                                        \
    do {                                                                \
        if (verbose > 1) {                                              \
            fprintf(stderr, __VA_ARGS__);                               \
            fprintf(stderr, "\n");                                      \
        }                                                               \
    } while (0)
#define TRC(...)                                                        \
    do {                                                                \
        if (verbose > 2) {                                              \
            fprintf(stderr, __VA_ARGS__);                               \
            fprintf(stderr, "\n");                                      \
        }                                                               \
    } while (0)

extern void fatal(const char *msg);
