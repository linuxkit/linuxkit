//go:build linux && (386 || amd64)

package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path"
	"testing"
)

// fakeVMware replaces the backdoor hooks with isVM and a guestinfo map; a key
// missing from info reads as "", and err (if set) fails every read
func fakeVMware(t *testing.T, isVM bool, info map[string]string, err error) *int {
	calls := 0
	vm, gi, cfg := vmwareIsVM, vmwareGuestInfo, ConfigPath
	vmwareIsVM = func() (bool, error) { return isVM, nil }
	vmwareGuestInfo = func(key string) (string, error) {
		calls++
		return info[key], err
	}
	ConfigPath = t.TempDir()
	t.Cleanup(func() { vmwareIsVM, vmwareGuestInfo, ConfigPath = vm, gi, cfg })
	return &calls
}

func TestVMwareProbeNotVM(t *testing.T) {
	calls := fakeVMware(t, false, map[string]string{guestUserData: "hello"}, nil)
	if NewVMware().Probe(context.Background()) {
		t.Error("Probe: true outside VMware")
	}
	if *calls != 0 {
		t.Errorf("guestinfo read %d times outside VMware", *calls)
	}
}

func TestVMwareIsVMRecoversFault(t *testing.T) {
	orig := vmwareIsVirtualWorld
	vmwareIsVirtualWorld = func(bool) (bool, error) { panic("signal SIGSEGV") }
	t.Cleanup(func() { vmwareIsVirtualWorld = orig })
	if isVM, err := vmwareIsVM(); isVM || err == nil || err.Error() != "vmware backdoor faulted" {
		t.Errorf("vmwareIsVM: %v, %v; want false, \"vmware backdoor faulted\"", isVM, err)
	}
}

func TestVMwareProbeEncodings(t *testing.T) {
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	zw.Write([]byte("hello"))
	zw.Close()
	for enc, val := range map[string]string{
		"":            "hello\n",
		"base64":      base64.StdEncoding.EncodeToString([]byte("hello")),
		"gzip+base64": base64.StdEncoding.EncodeToString(gz.Bytes()),
	} {
		t.Run(enc, func(t *testing.T) {
			fakeVMware(t, true, map[string]string{guestUserData: val, guestUserData + ".encoding": enc}, nil)
			p := NewVMware()
			if !p.Probe(context.Background()) {
				t.Fatal("Probe: false with userdata set")
			}
			if ud, err := p.Extract(context.Background()); err != nil || string(ud) != "hello" {
				t.Errorf("Extract: %q, %v; want \"hello\"", ud, err)
			}
		})
	}
}

func TestVMwareProbeEmpty(t *testing.T) {
	for _, v := range []string{"", "---"} {
		fakeVMware(t, true, map[string]string{guestMetaData: v, guestUserData: v}, nil)
		if NewVMware().Probe(context.Background()) {
			t.Errorf("Probe: true with metadata and userdata %q", v)
		}
	}
}

func TestVMwareProbeRPCError(t *testing.T) {
	fakeVMware(t, true, map[string]string{guestUserData: "hello"}, errors.New("rpc failed"))
	if NewVMware().Probe(context.Background()) {
		t.Error("Probe: true when guestinfo fails")
	}
}

func TestVMwareExtractWritesFiles(t *testing.T) {
	fakeVMware(t, true, map[string]string{
		guestMetaData:   "instance-id: x",
		guestVendorData: "vendor",
		guestUserData:   "user",
	}, nil)
	if ud, err := NewVMware().Extract(context.Background()); err != nil || string(ud) != "user" {
		t.Fatalf("Extract: %q, %v; want \"user\"", ud, err)
	}
	for name, want := range map[string]string{"metadata": "instance-id: x", "vendordata": "vendor"} {
		if got, err := os.ReadFile(path.Join(ConfigPath, name)); err != nil || string(got) != want {
			t.Errorf("%s: %q, %v; want %q", name, got, err, want)
		}
	}
}
