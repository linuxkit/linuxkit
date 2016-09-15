#include <fcntl.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/ioctl.h>
#include <linux/random.h>

#include "drng.h"

#define BUFSIZE 1024
#define MAX_RETRY_LIMIT 10

int main() {
	struct rand_pool_info *info = malloc(sizeof(struct rand_pool_info) + BUFSIZE);
	unsigned char *buffer = (unsigned char *)info->buf;
	int fd, ret, r;
	int entropy_count = 0;

	if (! info)
		return 1;

	fd = open("/dev/random", O_RDWR);
	if (fd == -1)
		return 1;

	if (RdSeed_isSupported()) {
		r = rdseed_get_bytes(BUFSIZE, buffer, 0, MAX_RETRY_LIMIT);
		if (r <= 0)
			return 1;
		entropy_count = r * 8; /* number of bits produced */
	} else if (RdRand_isSupported()) {
		r = rdrand_get_bytes(BUFSIZE, buffer);
		if (r != DRNG_SUCCESS)
			return 1;
		r = BUFSIZE; /* always provides amount requested on success */
		entropy_count = r / 64; /* uses PRNG so underlying there are fewer bits */
	} else {
		return 1;
	}

	info->entropy_count = entropy_count;
	info->buf_size = r;
	ret = ioctl(fd, RNDADDENTROPY, &info);
	if (ret != 0)
		return 1;
}
