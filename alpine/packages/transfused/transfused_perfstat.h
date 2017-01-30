#ifndef _TRANSFUSED_PERFSTAT_H_
#define _TRANSFUSED_PERFSTAT_H_

#include <inttypes.h>

struct connection;
struct parameters;

typedef struct {
	uint64_t id;
	uint64_t start;
	uint64_t stop;
} perfstat_t;

struct perfstats {
	uint32_t len;
	uint32_t nothing;
	struct perfstats *next;
	perfstat_t perfstat[0];
};

typedef struct perfstats perfstats_t;

int perfstat_open(uint64_t unique, struct connection *conn);
int perfstat_close(uint64_t unique, struct connection *conn);
void *start_perfstat(struct parameters *params, char *req, size_t len);
void *stop_perfstat(struct parameters *params, char *req, size_t len);

#endif /* _TRANSFUSED_PERFSTAT_H_ */
