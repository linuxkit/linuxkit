package main

// #cgo CFLAGS: -mrdrnd -mrdseed
// #include <immintrin.h>
// #include <x86intrin.h>
// #include <stdint.h>
// #include <cpuid.h>
// #include <linux/random.h>
// #include <sys/ioctl.h>
//
// int hasrdrand() {
//   unsigned int eax, ebx, ecx, edx;
//   __get_cpuid(1, &eax, &ebx, &ecx, &edx);
//
//   return ((ecx & bit_RDRND) == bit_RDRND);
// }
//
// int hasrdseed() {
//   unsigned int eax, ebx, ecx, edx;
//   __get_cpuid(7, &eax, &ebx, &ecx, &edx);
//
//   return ((ebx & bit_RDSEED) == bit_RDSEED);
// }
//
// int rdrand(uint64_t *val) {
//   return _rdrand64_step((unsigned long long *)val);
// }
//
// int rdseed(uint64_t *val) {
//   return _rdseed64_step((unsigned long long *)val);
// }
//
// int rndaddentropy = RNDADDENTROPY;
//
import "C"

import (
	"errors"
	"flag"
)

var disableRdrand = flag.Bool("disable-rdrand", false, "Disable use of RDRAND")
var disableRdseed = flag.Bool("disable-rdseed", false, "Disable use of RDSEED")

var hasRdrand, hasRdseed bool

func initRand() bool {
	hasRdrand = C.hasrdrand() == 1 && !*disableRdrand
	hasRdseed = C.hasrdseed() == 1 && !*disableRdseed
	return hasRdrand || hasRdseed
}

func rand() (uint64, error) {
	var x C.uint64_t
	// prefer rdseed as that is correct seed
	if hasRdseed && C.rdseed(&x) == 1 && !*disableRdseed {
		return uint64(x), nil
	}
	// failed rdseed, rdrand better than nothing
	if hasRdrand && C.rdrand(&x) == 1 && !*disableRdrand {
		return uint64(x), nil
	}
	return 0, errors.New("No randomness available")
}
