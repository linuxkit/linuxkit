//go:build darwin && cgo
// +build darwin,cgo

package main

import vz "github.com/Code-Hex/vz/v3"

func newVirtualizationFrameworkHostMicrophoneConfiguration() (*vz.VirtioSoundDeviceConfiguration, error) {
	audioDevice, err := vz.NewVirtioSoundDeviceConfiguration()
	if err != nil {
		return nil, err
	}

	inputStream, err := vz.NewVirtioSoundDeviceHostInputStreamConfiguration()
	if err != nil {
		return nil, err
	}
	audioDevice.SetStreams(inputStream)

	return audioDevice, nil
}
