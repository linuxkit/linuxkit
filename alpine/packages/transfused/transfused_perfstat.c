#include <stdlib.h>
#include <string.h>
#include "transfused.h"
#include "transfused_log.h"

uint64_t now(parameters_t *params)
{
	uint64_t ns_in_s = 1000000000;
	struct timespec now;
	if (clock_gettime(CLOCK_MONOTONIC, &now))
		die(1, params, "now", "");
	return (uint64_t)now.tv_sec * ns_in_s + (uint64_t)now.tv_nsec;
}

size_t size_of_perfstats(perfstats_t *p)
{
	size_t len = 0;

	while (p) {
		len += sizeof(perfstat_t) * p->len + sizeof(perfstats_t);
		p = p->next;
	}

	return len;
}

int perfstat_open(uint64_t unique, connection_t *conn)
{
	size_t sz;
	perfstats_t *old_perfstats = conn->perfstats;
	perfstats_t *stats = conn->perfstats;
	perfstat_t stat;

	if (!conn->perfstat)
		return 0;

	lock("perfstat lock: perfstat_open", &conn->perfstat_lock);
	if (conn->perfstat) {
		if (!stats || stats->len >= PERFSTATS_PER_SEGMENT) {
			sz = sizeof(perfstats_t);
			sz += PERFSTATS_PER_SEGMENT * sizeof(perfstat_t);
			stats = must_malloc("perfstats",sz);
			stats->next = old_perfstats;
			stats->len = 0;
			conn->perfstats = stats;
		}

		stat = (perfstat_t) {
			.id = unique,
			.start = now(conn->params),
			.stop = 0
		};
		stats->perfstat[stats->len] = stat;
		stats->len++;
	}
	unlock("perfstat unlock: perfstat_close", &conn->perfstat_lock);

	return 0;
}

int perfstat_close_locked(uint64_t unique, parameters_t *params,
			  perfstats_t *perfstats, int to_check)
{
	int i;
	perfstat_t *stat;

	if (!perfstats)
		return 1;

	i = perfstats->len - 1;
	while (i >= 0 && to_check > 0) {
		stat = &perfstats->perfstat[i];
		if (stat->id == unique) {
			stat->stop = now(params);
			return 0;
		} else {
			i--;
			to_check--;
		}
	}

	if (to_check && !perfstat_close_locked(unique, params, perfstats->next,
					       to_check))
		return 0;
	return 1;
}

int perfstat_close(uint64_t unique, connection_t *conn)
{
	int rc = 0;
	pthread_mutex_t *perfstat_lock = &conn->perfstat_lock;

	if (!conn->perfstat)
		return 0;

	lock("perfstat lock: perfstat_close", perfstat_lock);
	if (conn->perfstat)
		rc = perfstat_close_locked(unique, conn->params,
					   conn->perfstats,
					   MAX_PERFSTAT_CHECK);
	unlock("perfstat unlock: perfstat_close", perfstat_lock);

	return rc;
}

void *start_perfstat(parameters_t *params, char *req, size_t len)
{
	char *reply;
	uint16_t id = *((uint16_t *) req);
	char *mount = (char *) req + 2;
	connection_t *conn = find_connection(params->connections, mount,
					     len - 2);
	if (conn == NULL)
		return (void *)error_reply(id, "Mount %s unknown", mount);

	lock("perfstat lock: start_perfstat", &conn->perfstat_lock);
	conn->perfstat = 1;
	unlock("perfstat lock: start_perfstat", &conn->perfstat_lock);

	reply = (char *)must_malloc("start_perfstat", 8);
	*((uint32_t *)reply) = 16;
	*((uint16_t *) (reply + 4)) = PERFSTAT_REPLY;
	*((uint16_t *) (reply + 6)) = id;
	*((uint64_t *) (reply + 8)) = now(params);

	return (void *)reply;
}

void copy_and_free_perfstats(perfstats_t *p, char *buf)
{
	size_t len;
	perfstats_t *p_next;

	while (p) {
		p_next = p->next;
		len = p->len * sizeof(perfstat_t);
		memcpy(buf, p->perfstat, len);
		buf += len;
		free(p);
		p = p_next;
	}
}

void *stop_perfstat(parameters_t *params, char *req, size_t len)
{
	char *reply;
	uint16_t id = *((uint16_t *) req);
	char *mount = (char *) req + 2;
	connection_t *conn = find_connection(params->connections, mount,
					     len - 2);
	if (conn == NULL)
		return (void *)error_reply(id, "Mount %s unknown", mount);

	size_t out_len = 16;

	lock("perfstat lock: stop_perfstat", &conn->perfstat_lock);
	conn->perfstat = 0;

	out_len += size_of_perfstats(conn->perfstats);

	reply = (char *)must_malloc("stop_perfstat", out_len);
	*((uint32_t *)reply) = out_len;
	*((uint16_t *) (reply + 4)) = PERFSTAT_REPLY;
	*((uint16_t *) (reply + 6)) = id;
	*((uint64_t *) (reply + 8)) = now(params);

	copy_and_free_perfstats(conn->perfstats, reply + 16);

	unlock("perfstat lock: stop_perfstat", &conn->perfstat_lock);

	return (void *)reply;
}
