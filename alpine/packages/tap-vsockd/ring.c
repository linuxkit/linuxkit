#include <stdlib.h>
#include <pthread.h>
#include <unistd.h>
#include <sys/uio.h>
#include <errno.h>
#include <assert.h>
#include <stdio.h>
#include "ring.h"

extern void fatal(const char *msg);


/* A fixed-size circular buffer.

   The producer and consumer are positive integers from 0 to 2 * size-1.
	 Adds are modulo 2 * size. This effectively uses one bit to distinguish
	 the case where the buffer is empty (consumer == producer) from the case
	 where the buffer is full (consumer + size == producer). */
struct ring {
	int producer;        /* Next sequence number to be written */
	int consumer;        /* Next sequence number to be read */
	int last;            /* Sequence number of end of stream or -1 */
	int size;            /* Maximum number of buffered bytes */
	pthread_cond_t c;
	pthread_mutex_t m;
	char *data;
};

struct ring *ring_allocate(int size)
{
	struct ring *ring = (struct ring*)malloc(sizeof(struct ring));
	if (!ring) {
		fatal("Failed to allocate ring buffer metadata");
	}
	ring->data = (char*)malloc(size);
	if (!ring->data) {
		fatal("Failed to allocate ring buffer data");
	}
	int err = 0;
	if ((err = pthread_cond_init(&ring->c, NULL)) != 0) {
		errno = err;
		fatal("Failed to create condition variable");
	}
	if ((err = pthread_mutex_init(&ring->m, NULL)) != 0) {
		errno = err;
		fatal("Failed to create mutex");
	}
	ring->size = size;
	ring->producer = ring->consumer = 0;
	ring->last = -1;
	return ring;
}

#define RING_DATA_AVAILABLE(r)    \
  ((r->producer >= r->consumer) ? \
   (r->producer - r->consumer)  : \
   (2 * r->size + r->producer - r->consumer))
#define RING_FREE_REQUESTS(r) (r->size - RING_DATA_AVAILABLE(r))

#define RING_GET(r, seq) (&(r->data[seq % r->size]))

/* Signal that new data is been produced */
void ring_producer_advance(struct ring *ring, int n)
{
	int err = 0;
	assert(n >= 0);
	if ((err = pthread_mutex_lock(&ring->m)) != 0) {
		errno = err;
		fatal("Failed to lock mutex");
	}
	ring->producer = (ring->producer + n) % (2 * ring->size);
	if ((err = pthread_cond_broadcast(&ring->c)) != 0) {
		errno = err;
		fatal("Failed to signal condition variable");
	}
	if ((err = pthread_mutex_unlock(&ring->m)) != 0) {
		errno = err;
		fatal("Failed to unlock mutex");
	}
	return;
}

/* Signal that data has been consumed */
void ring_consumer_advance(struct ring *ring, int n)
{
	int err = 0;
	assert(n >= 0);
	if ((err = pthread_mutex_lock(&ring->m)) != 0) {
		errno = err;
		fatal("Failed to lock mutex");
	}
	ring->consumer = (ring->consumer + n) % (2 * ring->size);

	if ((err = pthread_cond_broadcast(&ring->c)) != 0) {
		errno = err;
		fatal("Failed to signal condition variable");
	}
	if ((err = pthread_mutex_unlock(&ring->m)) != 0) {
		errno = err;
		fatal("Failed to unlock mutex");
	}
	return;
}

/* The producer sends Eof */
void ring_producer_eof(struct ring *ring)
{
	int err = 0;
	if ((err = pthread_mutex_lock(&ring->m)) != 0) {
		errno = err;
		fatal("Failed to lock mutex");
	}
	ring->last = ring->producer - 1;
	if ((err = pthread_cond_broadcast(&ring->c)) != 0) {
		errno = err;
		fatal("Failed to signal condition variable");
	}
	if ((err = pthread_mutex_unlock(&ring->m)) != 0) {
		errno = err;
		fatal("Failed to unlock mutex");
	}
	return;
}

/* Wait for n bytes to become available. If the ring has shutdown, return
   non-zero. If data is available then return zero and fill in the first
	 iovec_len entries of the iovec. */
int ring_producer_wait_available(
	struct ring *ring, size_t n, struct iovec *iovec, int *iovec_len
) {
	int ret = 1;
	int err = 0;
	if ((err = pthread_mutex_lock(&ring->m)) != 0) {
		errno = err;
		fatal("Failed to lock mutex");
	}
	while ((RING_FREE_REQUESTS(ring) < n) && (ring->last == -1)) {
		if ((err = pthread_cond_wait(&ring->c, &ring->m)) != 0) {
			errno = err;
			fatal("Failed to wait on condition variable");
		}
	}
	if (ring->last != -1) {
		goto out;
	}
	char *producer = RING_GET(ring, ring->producer);
	char *consumer = RING_GET(ring, ring->consumer);
	assert (producer >= RING_GET(ring, 0));
	assert (producer <= RING_GET(ring, ring->size-1));
	assert (consumer >= RING_GET(ring, 0));
	assert (consumer <= RING_GET(ring, ring->size-1));
	if (*iovec_len <= 0) {
		ret = 0;
		fprintf(stderr, "no iovecs\n");
		goto out;
	}
	if (consumer > producer) {
		/* producer has not wrapped around the buffer yet */
		iovec[0].iov_base = producer;
		iovec[0].iov_len = consumer - producer;
		assert(iovec[0].iov_len > 0);
		*iovec_len = 1;
		ret = 0;
		goto out;
	}
	/* consumer has wrapped around, so the first chunk is from the producer to
	   the end of the buffer */
	iovec[0].iov_base = producer;
	iovec[0].iov_len = ring->size - (int) (producer - RING_GET(ring, 0));
	assert(iovec[0].iov_len > 0);
	if (*iovec_len == 1) {
		ret = 0;
		goto out;
	}
	*iovec_len = 1;
	/* also include the chunk from the beginning of the buffer to the consumer */
	iovec[1].iov_base = RING_GET(ring, 0);
	iovec[1].iov_len = consumer - RING_GET(ring, 0);
	if (iovec[1].iov_len > 0) {
		/* ... but don't bother if it's zero */
		*iovec_len = 2;
	}
	ret = 0;
out:
	if ((err = pthread_mutex_unlock(&ring->m)) != 0) {
		errno = err;
		fatal("Failed to unlock mutex");
	}
	if (ret == 0) {
		for (int i = 0; i < *iovec_len; i++) {
			assert(iovec[i].iov_base >= (void*)RING_GET(ring, 0));
			assert(iovec[i].iov_base + iovec[i].iov_len - 1 <= (void*)RING_GET(ring, ring->size - 1));
		}
	}
	return ret;
}

/* Wait for n bytes to become available. If the ring has shutdown, return
   non-zero. If data is available then return zero and fill in the first
	 iovec_len entries of the iovec. */
int ring_consumer_wait_available(
	struct ring *ring, size_t n, struct iovec *iovec, int *iovec_len
) {

	int ret = 1;
	int err = 0;
	if ((err = pthread_mutex_lock(&ring->m)) != 0) {
		errno = err;
		fatal("Failed to lock mutex");
	}
	while ((RING_DATA_AVAILABLE(ring) < n) && (ring->last == -1)) {
		if ((err = pthread_cond_wait(&ring->c, &ring->m)) != 0) {
			errno = err;
			fatal("Failed to wait on condition variable");
		}
	}
	if (ring->last != -1) {
		goto out;
	}
	char *producer = RING_GET(ring, ring->producer);
	char *consumer = RING_GET(ring, ring->consumer);
	assert (producer >= RING_GET(ring, 0));
	assert (producer <= RING_GET(ring, ring->size-1));
	assert (consumer >= RING_GET(ring, 0));
	assert (consumer <= RING_GET(ring, ring->size-1));
	if (*iovec_len <= 0) {
		ret = 0;
		goto out;
	}
	if (producer > consumer) {
		/* producer has not wrapped around the buffer yet */
		iovec[0].iov_base = consumer;
		iovec[0].iov_len = producer - consumer;
		assert(iovec[0].iov_len > 0);
		*iovec_len = 1;
		ret = 0;
		goto out;
	}
	/* producer has wrapped around, so the first chunk is from the consumer to
	   the end of the buffer */
	iovec[0].iov_base = consumer;
	iovec[0].iov_len = ring->size - (int) (consumer - RING_GET(ring, 0));
	assert(iovec[0].iov_len > 0);
	if (*iovec_len == 1) {
		ret = 0;
		goto out;
	}
	*iovec_len = 1;
	/* also include the chunk from the beginning of the buffer to the producer */
	iovec[1].iov_base = RING_GET(ring, 0);
	iovec[1].iov_len = producer - RING_GET(ring, 0);
	if (iovec[1].iov_len > 0) {
		/* ... but don't bother if its zero */
		*iovec_len = 2;
	}
	ret = 0;
out:
	if ((err = pthread_mutex_unlock(&ring->m)) != 0) {
		errno = err;
		fatal("Failed to unlock mutex");
	}
	if (ret == 0) {
		for (int i = 0; i < *iovec_len; i++) {
			assert(iovec[i].iov_base >= (void*)RING_GET(ring, 0));
			assert(iovec[i].iov_base + iovec[i].iov_len - 1 <= (void*)RING_GET(ring, ring->size - 1));
		}
	}
	return ret;
}
