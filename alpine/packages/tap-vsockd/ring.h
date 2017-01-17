#include <unistd.h>
#include <sys/uio.h>

/* A fixed-size circular buffer */
struct ring;

/* Allocate a circular buffer with the given payload size.
   Size must be < INT_MAX / 2. */
extern struct ring *ring_allocate(int size);

/* Signal that new data is been produced */
extern void ring_producer_advance(struct ring *ring, int n);

/* Signal that data has been consumed */
extern void ring_consumer_advance(struct ring *ring, int n);

/* The producer sends Eof. This will cause ring_consumer_wait_available
   and ring_producer_wait_available to return an error. */
extern void ring_producer_eof(struct ring *ring);

/* Wait for n bytes of space for new data to become available. If
   ring_producer_eof has been called, return non-zero. If space is available
   then fill the first *iovec_len entries of the iovec and set *iovec_len to
   the number of iovecs used. */
extern int ring_producer_wait_available(
	struct ring *ring, size_t n, struct iovec *iovec, int *iovec_len
);

/* Wait for n bytes to become available for reading. If ring_producer_eof has
   been called, return non-zero. If data is available then fill the first
   *iovec_len entries of the iovec and set *iovec_len to the number of iovecs
   used. */
extern int ring_consumer_wait_available(
	struct ring *ring, size_t n, struct iovec *iovec, int *iovec_len
);
