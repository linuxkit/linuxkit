//go:build !(linux && 386 && amd64)

package main

// ProviderVMware implements VMware provider interface for unsupported architectures
type ProviderVMware struct{}

// NewVMware returns a new VMware Provider
func NewVMware() *ProviderVMware {
	return nil
}

// String implements provider interface
func (p *ProviderVMware) String() string {
	return ""
}

// Probe implements provider interface
func (p *ProviderVMware) Probe() bool {
	return false
}

// Extract implements provider interface
func (p *ProviderVMware) Extract() ([]byte, error) {
	// Get vendor data, if empty do not fail
	return nil, nil
}
