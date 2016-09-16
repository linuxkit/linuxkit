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

/*! \file drng.h
*  \brief Public header for libdrng.
*
* This is the public header for libdrng.
*/

#ifndef DRNG_H
#define DRNG_H

/* Windows Specific */
#ifdef _WIN32

#if __STDC_VERSION__ >= 199901L
#include <stdint.h>
#else
/* MSVC specific */
typedef unsigned __int16 uint16_t;
typedef unsigned __int32 uint32_t;
typedef unsigned __int64 uint64_t;
#endif

#endif /* Windows */

/* GNUC Specific */
#ifdef __GNUC__

#define HAVE_RDRAND_IN_GCC 1

#include <inttypes.h>
#ifdef _MSC_VER
/* MSVC specific */
typedef unsigned __int16 uint16_t;
typedef unsigned __int32 uint32_t;
typedef unsigned __int64 uint64_t;
#endif

#endif /* GNUC */

#if defined(__GNUC__) || defined(__INTEL_COMPILER) || defined(_WIN64)
# define _NOT_WIN32
#endif

/*! \def DRNG_SUCCESS
*   The rdseed/rdrand call was successful, the hardware was ready, and a random
*   number was returned.
*/
#define DRNG_SUCCESS 1

/*! \def DRNG_NOT_READY
*  The rdseed/rdrand call was unsuccessful, the hardware was not ready, and a
*  random number was not returned.
*/
#define DRNG_NOT_READY -1

/*! \def DRNG_SUPPORTED
* The rdseed/rdrand instruction is supported by the host hardware.
*/
#define DRNG_SUPPORTED -2

/*! \def DRNG_UNSUPPORTED
* The rdseed/rdrand instruction is unsupported by the host hardware.
*/
#define DRNG_UNSUPPORTED -3

/*! \def DRNG_SUPPORT_UNKNOWN
* Whether or not the hardware supports the rdseed/rdrand instruction is unknown
*/
#define DRNG_SUPPORT_UNKNOWN -4


/*! \brief Determines whether or not rdrand is supported by the CPU
*
* This function calls cpuid to determine rdrand support and caches the
* result in a static variable. This prevents calling cpuid on subsequent invocations.
*
* \return bool/int of whether or not rdrand is supported
*/
int RdRand_isSupported();


/*! \brief Determines whether or not rdseed is supported by the CPU
*
* This function calls cpuid to determine rdseed support and caches the
* result in a static variable. This prevents calling cpuid on subsequent invocations.
*
* \return bool/int of whether or not rdseed is supported
*/
int RdSeed_isSupported();


/*! \brief Calls rdseed for a 16-bit result.
*
* This function calls rdseed requesting a 16-bit result. By default, it will
* perform only a single call to rdseed, returning success or failure. On
* success the data is written to memory pointed to by x.  On failure an error
* is returned unless the int retry_count is non-zero, in which case the function
* will retry rdseed until a successful result is obtained, or until the set number of
* retries occurs.
*
* This function also ensures that rdseed is supported by the cpu or fails gracefully.
*
* \param x pointer to memory to store the random result
* \param retry_count int to determine how many rdseed retries should be attempted
*
* \return whether or not the call was successful, or supported at all
*/
int rdseed_16(uint16_t* x, int retry_count);

/*! \brief Calls rdrand for a 16-bit result.
*
* This function calls rdrand requesting a 16-bit result. By default, it will
* perform only a single call to rdrand, returning success or failure. On
* success, the data is written to memory pointed to by x. On failure an error
* is returned unless the int retry is true (non-zero), in which case the function
* will retry rdrand up to 10 times for a successful result before returning an error.
*
* This function also ensures that rdrand is supported by the cpu or fails
* gracefully.
*
* \param x pointer to memory to store the random result
* \param retry int to determine whether or not to loop until rdrand succeeds
*		  or until 10 failed attempts
*
* \return whether or not the call was successful, or supported at all
*/
int rdrand_16(uint16_t* x, int retry);

/*! \brief Calls rdseed for a 32-bit result.
*
* This function calls rdseed requesting a 32-bit result. By default, it will
* perform only a single call to rdseed, returning success or failure. On
* success the data is written to memory pointed to by x.  On failure an error
* is returned unless the int retry_count is non-zero, in which case the function
* will retry rdseed until a successful result is obtained, or until the set number of
* retries occurs.
*
* This function also ensures that rdseed is supported by the cpu or fails gracefully.
*
* \param x pointer to memory to store the random result
* \param retry_count int to determine how many rdseed retries should be attempted
*
* \return whether or not the call was successful, or supported at all
*/
int rdseed_32(uint32_t* x, int retry_count);

/*! \brief Calls rdrand for a 32-bit result.
*
* This function calls rdrand requesting a 32-bit result. By default, it will
* perform only a single call to rdrand, returning success or failure. On
* success, the data is written to memory pointed to by x. On failure an error
* is returned unless the int retry is true (non-zero), in which case the function
* will retry rdrand up to 10 times for a successful result before returning an error.
*
* This function also ensures that rdrand is supported by the cpu or fails
* gracefully.
*
* \param x pointer to memory to store the random result
* \param retry int to determine whether or not to loop until rdrand succeeds
*		  or until 10 failed attempts
*
* \return whether or not the call was successful, or supported at all
*/
int rdrand_32(uint32_t* x, int retry);

/*! \brief Calls rdseed for a 64-bit result.
*
* This function calls rdseed requesting a 64-bit result. By default, it will
* perform only a single call to rdseed, returning success or failure. On
* success the data is written to memory pointed to by x.  On failure an error
* is returned unless the int retry_count is non-zero, in which case the function
* will retry rdseed until a successful result is obtained, or until the set number of
* retries occurs.
*
* This function also ensures that rdseed is supported by the cpu or fails gracefully.
*
* \param x pointer to memory to store the random result
* \param retry_count int to determine how many rdseed retries should be attempted
*
* \return whether or not the call was successful, or supported at all
*/
int rdseed_64(uint64_t* x, int retry_count);

/*! \brief Calls rdrand for a 64-bits result.
*
* This function calls rdrand requesting a 64-bit result. By default, it will
* perform only a single call to rdrand, returning success or failure. On
* success, the data is written to memory pointed to by x. On failure an error
* is returned unless the int retry is true (non-zero), in which case the function
* will retry rdrand up to 10 times for a successful result before returning an error.
*
* This function also ensures that rdrand is supported by the cpu or fails
* gracefully.
*
* \param x pointer to memory to store the random result
* \param retry int to determine whether or not to loop until rdrand succeeds
*		  or until 10 failed attempts
*
* \return whether or not the call was successful, or supported at all
*/
int rdrand_64(uint64_t* x, int retry);

/*! \brief Calls rdseed to obtain multiple 64-byte results.
*
* This function calls rdseed requesting multiple 64-bit results. On
* success, the data is written to memory pointed to by x. If a call to rdseed
* fails to return a value this function will retry it if int max_retries is non-zero,
* but if the total retry count exceeds max_retries then it will return the total
* number of the 64-bit results it was able to generate.
*
* The int skip parameter is provided as a convenience to the user to resume 
* filling the buffer where it left off if a previous operation did not complete.
* If it is set, the function will appended (n - skip) values to the end of
* the partially-filled buffer pointed to by x.
*
* \param n total number of 64-bit random seeds to generate
* \param x pointer to memory buffer to fill with 64-bit random seeds
* \param max_retries total number of retries that will be made by multiple rdseed_64 call
* \param skip int to determine index of array to start from
* \return total number of results generated or error number
*/
int rdseed_get_n_64(unsigned int n, uint64_t* x, unsigned int skip, unsigned int max_retries);

/*! \brief Calls rdrand to obtain multiple 64-byte results.
*
* This function calls rdrand requesting multiple 64-byte results. On
* success, the data is written to memory pointed to by x. This function
* calls rdrand_64 and if any of those invocations fail, this function
* fails. It returns the same values as rdrand_64.
*
* \param n total number of 64-bit random values to generate
* \param x pointer to memory buffer to fill with 64-bit random values
*/
int rdrand_get_n_64(unsigned int n, uint64_t* x);

/*! \brief Calls rdseed to obtain multiple 32-byte results.
*
* This function calls rdseed requesting multiple 32-bit results. On
* success, the data is written to memory pointed to by x. If a call to rdseed
* fails to return a value this function will retry it if int max_retries is non-zero,
* but if the total retry count exceeds max_retries then it will return the total
* number of the 32-bit results it was able to generate.
*
* The int skip parameter is provided as a convenience to the user to resume 
* filling the buffer where it left off if a previous operation did not complete.
* If it is set, the function will appended (n - skip) values to the end of
* the partially-filled buffer pointed to by x.
*
* \param n total number of 32-bit random seeds to generate
* \param x pointer to memory buffer to fill with 32-bit random seeds
* \param max_retries total number of retries that will be made by multiple rdseed_32 call
* \param skip int to determine index of array to start from
* \return total number of results generated or error number
*/
int rdseed_get_n_32(unsigned int n, uint32_t* x, unsigned int skip, unsigned int max_retries);


/*! \brief Calls rdrand to obtain multiple 32-byte results.
*
* This function calls rdrand requesting multiple 32-byte results. On
* success, the data is written to memory pointed to by x. This function
* calls rdrand_32 and if any of those invocations fail, this function
* fails. It returns the same values as rdrand_32.
*
* \param n total number of 32-bit random values to generate
* \param x pointer to memory buffer to fill with 32-bit random values
*/
int rdrand_get_n_32(unsigned int n, uint32_t* x);

/*! \brief Calls rdseed to fill a buffer of arbitrary size with random bytes.
*
* This function calls rdseed requesting multiple 64- or 32-bit results to
* fill a buffer of arbitrary size. If a call to rdseed
* fails to return a value this function will retry it if int max_retries is non-zero,
* but if the total retry count exceeds max_retries then it will return the total
* number of the bytes written to the buffer pointed to be x.
*
* The int skip parameter is provided as a convenience to the user to resume 
* filling the buffer where it left off if a previous operation did not complete.
* If it is set, the function will appended (n - skip) bytes to the end of
* the partially-filled buffer pointed to by x.
*
* \param n size of the buffer to fill with random bytes
* \param buffer pointer to memory to store the random result
* \param skip int to determine index of array to start from, to make the code re-entrant
* \param max_retries total number of retries that will be made by multiple rdseed_32 call
*
* \return total number or bytes generated if rdseed is supported
*/

int rdseed_get_bytes(unsigned int n, unsigned char *buffer, unsigned int skip, unsigned int max_retries);

/*! \brief Calls rdrand to fill a buffer of arbitrary size with random bytes.
*
* This function calls rdrand requesting multiple 64- or 32-bit results to
* fill a buffer of arbitrary size.
*
* \param n size of the buffer to fill with random bytes
* \param buffer pointer to memory to store the random result
*
* \return whether or not the call was successful, or supported at all
*/

int rdrand_get_bytes(unsigned int n, unsigned char *buffer);

#endif // DRNG_H
