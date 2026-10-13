package main

import "testing"

func TestVirtualizationFrameworkAudioInputFlag(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want bool
	}{
		{name: "enabled by default", want: true},
		{name: "can be disabled", args: []string{"--audio-input=false"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, called := false, false
			cmd := newVirtualizationFrameworkCmd(func(cfg virtualizationFramwworkConfig, _ string) error {
				got = cfg.audioInput
				called = true
				return nil
			})
			cmd.SetArgs(append(tt.args, "fixture"))
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if !called {
				t.Fatal("virtualization runner was not called")
			}
			if got != tt.want {
				t.Fatalf("config.audioInput = %t, want %t", got, tt.want)
			}
		})
	}
}
