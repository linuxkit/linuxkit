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
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

var hasRdrand, hasRdseed bool

type randInfo struct {
	entropyCount int
	size         int
	buf          uint64
}

func initRand() bool {
	hasRdrand = C.hasrdrand() == 1
	hasRdseed = C.hasrdseed() == 1
	return hasRdrand || hasRdseed
}

func rand() (uint64, error) {
	var x C.uint64_t
	// prefer rdseed as that is correct seed
	if hasRdseed && C.rdseed(&x) == 1 {
		return uint64(x), nil
	}
	// failed rdseed, rdrand better than nothing
	if hasRdrand && C.rdrand(&x) == 1 {
		return uint64(x), nil
	}
	return 0, errors.New("No randomness available")
}

func writeEntropy(random *os.File) (int, error) {
	r, err := rand()
	if err != nil {
		// assume can fail occasionally
		return 0, nil
	}
	const entropy = 64 // they are good random numbers, Brent
	info := randInfo{entropy, 8, r}
	ret, _, err := unix.Syscall(unix.SYS_IOCTL, uintptr(random.Fd()), uintptr(C.rndaddentropy), uintptr(unsafe.Pointer(&info)))
	if ret == 0 {
		return 8, nil
	}
	return 0, err
}
