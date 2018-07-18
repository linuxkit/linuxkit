/*-
 * Copyright (c) 2015 xhyve developers
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions
 * are met:
 * 1. Redistributions of source code must retain the above copyright
 *    notice, this list of conditions and the following disclaimer.
 * 2. Redistributions in binary form must reproduce the above copyright
 *    notice, this list of conditions and the following disclaimer in the
 *    documentation and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY NETAPP, INC ``AS IS'' AND
 * ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED.  IN NO EVENT SHALL NETAPP, INC OR CONTRIBUTORS BE LIABLE
 * FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
 * DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS
 * OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION)
 * HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT
 * LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY
 * OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF
 * SUCH DAMAGE.
 *
 */

/* makeshift callout implementation based on OSv and FreeBSD */

#include <stdio.h>
#include <stdint.h>
#include <stdbool.h>
#include <stdlib.h>
#include <errno.h>
#include <pthread.h>
#include <sys/time.h>
#include <mach/mach.h>
#include <mach/mach_time.h>

#include <xhyve/support/misc.h>
#include <xhyve/vmm/vmm_callout.h>

#define callout_cmp(a, b) ((a)->timeout < (b)->timeout)

static mach_timebase_info_data_t timebase_info;
static pthread_t callout_thread;
static pthread_mutex_t callout_mtx;
static pthread_cond_t callout_cnd;
static struct callout *callout_queue;
static bool work;
static bool initialized = false;

static inline uint64_t nanos_to_abs(uint64_t nanos) {
  return (nanos * timebase_info.denom) / timebase_info.numer;
}

static inline uint64_t abs_to_nanos(uint64_t abs) {
  return (abs * timebase_info.numer) / timebase_info.denom;
}

static inline uint64_t sbt2mat(sbintime_t sbt) {
  uint64_t s, ns;

  s = (((uint64_t) sbt) >> 32);
  ns = (((uint64_t) 1000000000) * (uint32_t) sbt) >> 32;

  return (nanos_to_abs((s * 1000000000) + ns));
}

static inline void mat_to_ts(uint64_t mat, struct timespec *ts) {
  uint64_t ns;

  ns = abs_to_nanos(mat);

  ts->tv_sec = (ns / 1000000000);
  ts->tv_nsec = (ns % 1000000000);
}

void binuptime(struct bintime *bt) {
  uint64_t ns;

  ns = abs_to_nanos(mach_absolute_time());

  bt->sec = (ns / 1000000000);
  bt->frac = (((ns % 1000000000) * (((uint64_t) 1 << 63) / 500000000)));
}

void getmicrotime(struct timeval *tv) {
  uint64_t ns, sns;

  ns = abs_to_nanos(mach_absolute_time());

  sns = (ns / 1000000000);
  tv->tv_sec = (long) sns;
  tv->tv_usec = (int) ((ns - sns) / 1000);
}

static void callout_insert(struct callout *c) {
  struct callout *node = callout_queue;

  if (!node) {
    callout_queue = c;
    c->prev = NULL;
    c->next = NULL;
    c->queued = 1;
    return;
  }

  if (callout_cmp(c, node)) {
    node->prev = c;
    c->prev = NULL;
    c->next = node;
    callout_queue = c;
    c->queued = 1;
    return;
  }

  while (node->next) {
    if (callout_cmp(c, node->next)) {
      c->prev = node;
      c->next = node->next;
      node->next->prev = c;
      node->next = c;
      c->queued = 1;
      return;
    }
    node = node->next;
  }

  c->prev = node;
  c->next = NULL;
  node->next = c;
  c->queued = 1;
}

static void callout_remove(struct callout *c) {
  if (!c->queued) {
    return;
  }

  if (c->prev) {
    c->prev->next = c->next;
  } else {
    callout_queue = c->next;
  }

  if (c->next) {
    c->next->prev = c->prev;
  }

  c->prev = NULL;
  c->next = NULL;
  c->queued = 0;
}

static void *callout_thread_func(UNUSED void *arg) {
  struct callout *c;
  struct timespec ts;
  uint64_t delta, mat;
  int ret;

  pthread_setname_np("callout");

  pthread_mutex_lock(&callout_mtx);

  while (true) {
    /* wait for work */
    while (!callout_queue) {
      pthread_cond_wait(&callout_cnd, &callout_mtx);
    };

    /* get the callout with the nearest timout */
    c = callout_queue;

    if (!(c->flags & (CALLOUT_ACTIVE | CALLOUT_PENDING))) {
      abort();
    }

    /* wait for timeout */
    ret = 0;
    while ((ret != ETIMEDOUT) && !work) {
      mat = mach_absolute_time();
      if (mat >= c->timeout) {
        /* XXX: it might not be worth sleeping for very short timeouts */
        ret = ETIMEDOUT;
        break;
      }

      delta = c->timeout - mat;
      mat_to_ts(delta, &ts);
      ret = pthread_cond_timedwait_relative_np(&callout_cnd, &callout_mtx, &ts);
    };

    work = false;

    if (!(ret == ETIMEDOUT) || !c->queued) {
      continue;
    }

    /* dispatch */
    c->flags &= ~CALLOUT_PENDING;

    pthread_mutex_unlock(&callout_mtx);
    c->callout(c->argument);
    pthread_mutex_lock(&callout_mtx);

    /* note: after the handler has been invoked the callout structure can look
     *       much differently, the handler may have rescheduled the callout or
     *       even freed it.
     *
     *       if the callout is still enqueued it means that it hasn't been
     *       freed by the user
     *
     *       reset || drain || !stop
     */

    if (c->queued) {
      /* if the callout hasn't been rescheduled, remove it */
      if (((c->flags & CALLOUT_PENDING) == 0) || (c->flags & CALLOUT_WAITING)) {
        c->flags |= CALLOUT_COMPLETED;
        callout_remove(c);
      }
    }
  }

  return NULL;
}

void callout_init(struct callout *c, int mpsafe) {
  if (!mpsafe) {
    abort();
  }

  memset(c, 0, sizeof(struct callout));

  if (pthread_cond_init(&c->wait, NULL)) {
    abort();
  }
}

static int callout_stop_safe_locked(struct callout *c, int drain) {
  int result = 0;

  if ((drain) && (pthread_self() != callout_thread) && (callout_pending(c) ||
    (callout_active(c) && !callout_completed(c))))
  {
    if (c->flags & CALLOUT_WAITING) {
      abort();
    }

    /* wait for callout */
    c->flags |= CALLOUT_WAITING;
    work = true;

    pthread_cond_signal(&callout_cnd);

    while (!(c->flags & CALLOUT_COMPLETED)) {
      pthread_cond_wait(&c->wait, &callout_mtx);
    }

    c->flags &= ~CALLOUT_WAITING;
    result = 1;
  }

  callout_remove(c);

  /* clear flags */
  c->flags &= ~(CALLOUT_ACTIVE | CALLOUT_PENDING | CALLOUT_COMPLETED |
    CALLOUT_WAITING);

  return (result);
}

int callout_stop_safe(struct callout *c, int drain) {
  pthread_mutex_lock(&callout_mtx);
  callout_stop_safe_locked(c, drain);
  pthread_mutex_unlock(&callout_mtx);
  return 0;
}

int callout_reset_sbt(struct callout *c, sbintime_t sbt,
  UNUSED sbintime_t precision, void (*ftn)(void *), void *arg, int flags)
{
  int result;
  bool is_next_timeout;

  is_next_timeout = false;

  pthread_mutex_lock(&callout_mtx);

  if (!((flags == 0) || (flags == C_ABSOLUTE)) || (c->flags !=0)) {
    /* FIXME */
    //printf("XHYVE: callout_reset_sbt 0x%08x 0x%08x\r\n", flags, c->flags);
    //abort();
  }

  c->timeout = sbt2mat(sbt);

  if (flags != C_ABSOLUTE) {
    c->timeout += mach_absolute_time();
  }

  result = callout_stop_safe_locked(c, 0);

  c->callout = ftn;
  c->argument = arg;
  c->flags |= (CALLOUT_PENDING | CALLOUT_ACTIVE);

  callout_insert(c);

  if (c == callout_queue) {
    work = true;
    is_next_timeout = true;
  }

  pthread_mutex_unlock(&callout_mtx);

  if (is_next_timeout) {
    pthread_cond_signal(&callout_cnd);
    is_next_timeout = false;
  }

  return (result);
}

void callout_system_init(void) {
  if (initialized) {
    return;
  }

  mach_timebase_info(&timebase_info);

  if (pthread_mutex_init(&callout_mtx, NULL)) {
    abort();
  }

  if (pthread_cond_init(&callout_cnd, NULL)) {
    abort();
  }

  callout_queue = NULL;
  work = false;

  if (pthread_create(&callout_thread, /*&attr*/ NULL, &callout_thread_func,
    NULL))
  {
    abort();
  }

  initialized = true;
}

//static void callout_queue_print(void) {
//  struct callout *node;
//
//  pthread_mutex_lock(&callout_mtx);
//  for (node = callout_queue; node; node = node->next) {
//    printf("t:%llu -> ", abs_to_nanos(node->timeout));
//    if (!node->next) {
//      break;
//    }
//  }
//  pthread_mutex_unlock(&callout_mtx);
//  printf("NULL\n");
//}

//void fire (void *arg) {
//  printf("fire!\n");
//}
//
//int main(void) {
//  struct callout a;
//  sbintime_t sbt;
//  printf("xhyve_timer\n");
//  callout_system_init();
//  callout_init(&a, 1);
//  sbt = ((sbintime_t) (((uint64_t) 3) << 32));
//  callout_reset_sbt(&a, sbt, 0, &fire, NULL, 0);
//  while (1);
//  return 0;
//}
