/* Copyright © 2015, Intel Corporation.  All rights reserved. 
 
Redistribution and use in source and binary forms, with or without
modification, are permitted provided that the following conditions are met:
 
-       Redistributions of source code must retain the above copyright notice,
		this list of conditions and the following disclaimer.
-       Redistributions in binary form must reproduce the above copyright 
		notice, this list of conditions and the following disclaimer in the
		documentation and/or other materials provided with the distribution.
-       Neither the name of Intel Corporation nor the names of its contributors
		may be used to endorse or promote products derived from this software
		without specific prior written permission.
 
THIS SOFTWARE IS PROVIDED BY INTEL CORPORATION "AS IS" AND ANY EXPRESS OR
IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF
MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO
EVENT SHALL INTEL CORPORATION BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO,
PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR
BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER
IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) 
ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
POSSIBILITY OF SUCH DAMAGE. */

/*! \file rdseed.c
*  \brief APIs for rdseed library.
*
* This is the rdseed library file which exposes
* different APIs for 32bit and 64bit system.
*/

#include "drng.h"
/* Windows Specific */
#ifdef _WIN32
#include <immintrin.h>
#include <string.h>
#endif /* Windows */

/* Linux Specific */
#ifdef __GNUC__
#ifdef __INTEL_COMPILER
# include <immintrin.h>
#endif
#include <stdint.h>
#include <string.h>
#endif /* Linux */

/*! \def RDSEED_MASK
*    The bit mask used to examine the ebx register returned by cpuid. The 
*   18th bit is set.
*/
#define RDSEED_MASK	0x40000

/*! \def rdseed_support_cache
*   Global variable to keep the drng support value in cache
*/
int rdseed_support_cache = DRNG_SUPPORT_UNKNOWN;

/*! \def retry_counter
*	Global variable to keep track of number of rdseed retries 
*/
unsigned int retry_counter = 0;


#if defined(_WIN64)||defined(_LP64)
# define _IS64BIT
#endif

#ifdef _IS64BIT
typedef uint64_t _wordlen_t;
#else
typedef uint32_t _wordlen_t;
#endif


/* Linux Specific */

/* Mimic the Intel compiler's intrinsics as best we can if we are using gcc */

#ifdef __GNUC__

# define __cpuid(x,y,z) asm volatile("cpuid":"=a"(x[0]),"=b"(x[1]),"=c"(x[2]),"=d"(x[3]):"a"(y),"c"(z))

/* RDSEED isn't a supported instruction until gcc 4.6 */

# ifdef HAVE_RDSEED_IN_GCC

#  define _rdseed_step(x) ({ unsigned char err; asm volatile("rdseed %0; setc %1":"=r"(*x), "=qm"(err)); err; })

#  define _rdseed16_step(x) _rdseed_step(x)
#  define _rdseed32_step(x) _rdseed_step(x)

# else

/* Our version of gcc is too old, so we need to use byte code */

#  define _rdseed16_step(x) ({ unsigned char err; asm volatile(".byte 0x66; .byte 0x0f; .byte 0xc7; .byte 0xf8; setc %1":"=a"(*x), "=qm"(err)); err; })
#  define _rdseed32_step(x) ({ unsigned char err; asm volatile(".byte 0x0f; .byte 0xc7; .byte 0xf8; setc %1":"=a"(*x), "=qm"(err)); err; })

# endif

#ifdef _IS64BIT

# ifdef HAVE_RDSEED_IN_GCC
#  define _rdseed64_step(x) _rdseed_step(x)
# else

/* Our version of gcc is too old, so we need to use byte code */

#  define _rdseed64_step(x) ({ unsigned char err; asm volatile(".byte 0x48; .byte 0x0f; .byte 0xc7; .byte 0xf8; setc %1":"=a"(*x), "=qm"(err)); err; })

# endif

#else

/*
*   The Intel compiler intrinsic for generating a 64-bit rand on a 32-bit 
*   system maps to two 32-bit RDSEED instructions. Because of the way
*   the way the DRNG is implemented you can do this up to a 128-bit value
*   (for crypto purposes) before you no longer have multiplicative 
*   prediction resistance.
*  
*   Note that this isn't very efficient.  If you need 64-bit values
*   you should really be on a 64-bit system.
*/

int _rdseed64_step (uint64_t *x);

int _rdseed64_step (uint64_t *x) 
{
	uint32_t xlow, xhigh;
	int rv;

	if ( (rv= _rdseed32_step(&xlow)) != DRNG_SUCCESS ) return rv;
	if ( (rv= _rdseed32_step(&xhigh)) != DRNG_SUCCESS ) return rv;

	*x= (uint64_t) xlow | ((uint64_t)xhigh<<32);

	return DRNG_SUCCESS;
}

# endif

#endif /* GNUC */

/*! \brief Queries cpuid to see if rdseed is supported and caches the result
 *
 * rdseed support in a CPU is determined by examining the 18th bit of the ebx
 * register after calling cpuid.
 * 
 * \return bool of whether or not rdseed is supported
 */
int RdSeed_cpuid()
{
	/* Are we on an Intel processor? */
	unsigned int info[4] = { -1, -1, -1, -1 };

#ifdef _WIN32
	__cpuid(info, /*feature bits*/0);
#endif

#ifdef __GNUC__
	__cpuid(info, /*feature bits*/0, 0);
#endif
	if (memcmp((void *)&info[1], (void *) "Genu", 4) != 0 ||
		memcmp((void *)&info[3], (void *) "ineI", 4) != 0 ||
		memcmp((void *)&info[2], (void *) "ntel", 4) != 0) {
		return 0;
	}

	
	/* Do we have RDSEED? */

	unsigned int info_rdseed[4] = { -1, -1, -1, -1 };

#ifdef _WIN32
	__cpuid(info_rdseed, /*feature bits*/7);
#endif

#ifdef __GNUC__
	__cpuid(info_rdseed, /*feature bits*/7, 0);
#endif

	unsigned int ebx = info_rdseed[1];
	if ((ebx & RDSEED_MASK) == RDSEED_MASK)
		return 1;
	else
		return 0;
}


int RdSeed_isSupported()
{
	int supported = rdseed_support_cache;

	if (supported == DRNG_SUPPORT_UNKNOWN)
	{

		if (RdSeed_cpuid())
			supported = DRNG_SUPPORTED;
		else
			supported = DRNG_UNSUPPORTED;

		rdseed_support_cache = supported;

	}
	
	return (supported == DRNG_SUPPORTED) ? 1 : 0;
}

int rdseed_16(uint16_t* x, int retry_count)
{
	if (RdSeed_isSupported())
	{

		if (_rdseed16_step(x))
			return DRNG_SUCCESS;
		else
		{
			retry_counter = retry_count;
			while (retry_counter > 0)
			{
				retry_counter--;
				if (_rdseed16_step(x))
					return DRNG_SUCCESS;
			}

			return DRNG_NOT_READY;
		}
	}
	else
		return RdSeed_isSupported();
}


int rdseed_32(uint32_t* x, int retry_count)
{
	if (RdSeed_isSupported())
	{

		if (_rdseed32_step(x))
			return DRNG_SUCCESS;
		else
		{
			retry_counter = retry_count;
			while (retry_counter > 0)
			{
				retry_counter--;
				if (_rdseed32_step(x))
					return DRNG_SUCCESS;
			}

			return DRNG_NOT_READY;
		}
	}
	else
		return RdSeed_isSupported();
}

#ifdef _NOT_WIN32

int rdseed_64(uint64_t* x, int retry_count)
{
	if (RdSeed_isSupported())
	{

		if (_rdseed64_step(x))
			return DRNG_SUCCESS;
		else
		{
			retry_counter = retry_count;
			while (retry_counter > 0)
			{
				retry_counter--;
				if (_rdseed64_step(x))
					return DRNG_SUCCESS;
			}

			return DRNG_NOT_READY;
		}
	}
	else
		return RdSeed_isSupported();
}

int rdseed_get_n_64(unsigned int n, uint64_t *dest, unsigned int skip, unsigned int max_retries)
{
	int success;
	unsigned int i;
	unsigned int success_count = 0;
	retry_counter = max_retries;

	if (skip)
	{
		n = n - skip;
		dest = &(dest[skip]);
		success_count = skip;
	}

	for (i = 0; i<n; i++)
	{
		success = rdseed_64(dest, retry_counter);
		if (success != DRNG_SUCCESS) return ((success == DRNG_UNSUPPORTED) ? success : success_count);
		dest = &(dest[1]);
		success_count++;
	}
	return success_count;
}
 
#endif

int rdseed_get_n_32(unsigned int n, uint32_t *dest, unsigned int skip, unsigned int max_retries)
{
	int success;
	unsigned int i;
	unsigned int success_count = 0;
	retry_counter = max_retries;

	if (skip)
	{
		n = n - skip;
		dest = &(dest[skip]);
		success_count = skip;
	}


	for (i = 0; i<n; i++)
	{
		success = rdseed_32(dest, retry_counter);
		if (success != DRNG_SUCCESS) return ((success == DRNG_UNSUPPORTED) ? success : success_count);
		dest = &(dest[1]);
		success_count++;
	}
	return success_count;
}

int rdseed_get_bytes(unsigned int n, unsigned char *dest, unsigned int skip, unsigned int max_retries)
{
	unsigned char *start;
	unsigned char *residualstart;
	_wordlen_t *blockstart;
	_wordlen_t i, temprand;
	unsigned int count;
	unsigned int residual;
	unsigned int startlen;
	unsigned int length;
	int success;
	unsigned int success_count = 0;
	unsigned int buffsize = n;
	retry_counter = max_retries;

	if (skip)
	{
		n = n - skip;
		dest = &(dest[skip]);
		success_count = skip;
	}

	/* Compute the address of the first 32- or 64- bit aligned block in the destination buffer, depending on whether we are in 32- or 64-bit mode */
	start = dest;
	if (((_wordlen_t)start % (_wordlen_t) sizeof(_wordlen_t)) == 0)
	{
		blockstart = (_wordlen_t *)start;
		count = n;
		startlen = 0;
	}
	else
	{
		blockstart = (_wordlen_t *)(((_wordlen_t)start & ~(_wordlen_t) (sizeof(_wordlen_t)-1) )+(_wordlen_t)sizeof(_wordlen_t));
		count = n - (sizeof(_wordlen_t) - (unsigned int)((_wordlen_t)start % sizeof(_wordlen_t)));
		startlen = (unsigned int)((_wordlen_t)blockstart - (_wordlen_t)start);
	}

	/* Compute the number of 32- or 64- bit blocks and the remaining number of bytes */
	residual = count % sizeof(_wordlen_t);
	length = count/sizeof(_wordlen_t);
	if (residual != 0)
	{
		residualstart = (unsigned char *)(blockstart + length);
	}

	/* Get a temporary random number for use in the residuals. Failout if retry fails */
	if (startlen > 0)
	{
#ifdef _IS64BIT
		if ((success = rdseed_64((uint64_t *)&temprand, retry_counter)) != DRNG_SUCCESS) return ((success == DRNG_UNSUPPORTED) ? success : success_count);
#else
		if ((success = rdseed_32((uint32_t *)&temprand, retry_counter)) != DRNG_SUCCESS) return ((success == DRNG_UNSUPPORTED) ? success : success_count);
#endif
	}

	/* populate the starting misaligned block */
	for (i = 0; i<startlen; i++)
	{
		start[i] = (unsigned char)(temprand & 0xff);
		temprand = temprand >> 8;
	}

	/* populate the central aligned block. Fail out if retry fails */

#ifdef _IS64BIT
	if ( (success = rdseed_get_n_64(length, (uint64_t *)(blockstart), 0, retry_counter)) < length)
	{
		success_count += success * 8;
		return success_count;
	}

#else
	if ((success = rdseed_get_n_32(length, (uint32_t *)(blockstart), 0, retry_counter)) < length)
	{
		success_count += success * 4;
		return success_count;
	}

#endif
	/* populate the final misaligned block */
	if (residual > 0)
	{
#ifdef _IS64BIT
		if ((success = rdseed_64((uint64_t *)&temprand, retry_counter)) != DRNG_SUCCESS) return success_count;
#else
		if ((success = rdseed_32((uint32_t *)&temprand, retry_counter)) != DRNG_SUCCESS) return success_count;
#endif

		for (i = 0; i<residual; i++)
		{
			residualstart[i] = (unsigned char)(temprand & 0xff);
			temprand = temprand >> 8;
		}
	}

	return buffsize;
}