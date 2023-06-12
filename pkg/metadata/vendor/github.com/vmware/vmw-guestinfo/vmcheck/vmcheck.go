// Copyright 2016-2017 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate go run asm.go -out vmcheck_amd64.s -arch amd64
//go:generate go run asm.go -out vmcheck_386.s -arch 386

package vmcheck

import (
	"encoding/binary"

	"github.com/vmware/vmw-guestinfo/bdoor"
)

type platform struct {
	cpuid       func(uint32, uint32) (uint32, uint32, uint32, uint32)
	accessPorts func() error
	knock       func() (bool, error)
}

var defaultPlatform = &platform{
	cpuid:       cpuid_low,
	accessPorts: openPortsAccess,
	knock:       bdoorKnock,
}

// From https://github.com/intel-go/cpuid/blob/master/cpuidlow_amd64.s
// Get the CPU ID low level leaf values.
func cpuid_low(arg1, arg2 uint32) (eax, ebx, ecx, edx uint32)

func bdoorKnock() (bool, error) {
	bp := &bdoor.BackdoorProto{}

	bp.CX.AsUInt32().SetWord(bdoor.CommandGetVersion)
	out := bp.InOut()
	// if there is no device, we get back all 1s
	return (0xffffffff != out.AX.AsUInt32().Word()) && (0 != out.AX.AsUInt32().Word()), nil
}

func (p *platform) isVirtualWorld(ignoreAccessErrors bool) (bool, error) {
	// Test the HV bit is set
	if !p.isVirtualCPU() {
		return false, nil
	}

	// Test if backdoor port is available.
	return p.hypervisorPortCheck(ignoreAccessErrors)
}

func (p *platform) isVirtualCPU() bool {
	HV := uint32(1 << 31)
	_, _, c, _ := p.cpuid(0x1, 0)
	if (c & HV) != HV {
		return false
	}

	_, b, c, d := p.cpuid(0x40000000, 0)

	buf := make([]byte, 12)
	binary.LittleEndian.PutUint32(buf, b)
	binary.LittleEndian.PutUint32(buf[4:], c)
	binary.LittleEndian.PutUint32(buf[8:], d)

	if string(buf) != "VMwareVMware" {
		return false
	}

	return true
}

// hypervisorPortCheck tests the availability of the backdoor port
// to the hypervisor, opportunistically tweaking I/O access level first.
func (p *platform) hypervisorPortCheck(ignoreAccessErrors bool) (bool, error) {
	// Privilege level 3 to access all ports above 0x3ff
	if err := p.accessPorts(); err != nil && !ignoreAccessErrors {
		return false, err
	}

	return p.knock()
}

// IsVirtualCPU checks if the cpu is a virtual CPU running on ESX.  It checks for
// the HV bit in the ECX register of the CPUID leaf 0x1.  Intel and AMD CPUs
// reserve this bit to indicate if the CPU is running in a HV. See
// https://en.wikipedia.org/wiki/CPUID#EAX.3D1:_Processor_Info_and_Feature_Bits
// for details.  If this bit is set, the reserved cpuid levels are used to pass
// information from the HV to the guest.  In ESX, this is the repeating string
// "VMwareVMware".
func IsVirtualCPU() bool {
	return defaultPlatform.isVirtualCPU()
}

// isVirtualWorld returns `true` if running in a VM and the backdoor is available.
// It also tries to elevate I/O privileges for the calling thread, which in
// some cases may be forbidden by the system (e.g Linux in `kernel_lockdown` mode
// does not allow `iopl` calls); the `ignoreAccessErrors` parameter allows
// to control library behavior in order to treat such errors as non-fatal.
func IsVirtualWorld(ignoreAccessErrors bool) (bool, error) {
	return defaultPlatform.isVirtualWorld(ignoreAccessErrors)
}
