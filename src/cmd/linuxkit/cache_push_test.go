package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCachePushProgressFlag(t *testing.T) {
	cmd := cachePushCmd()
	flag := cmd.Flags().Lookup("progress")
	require.NotNil(t, flag)
	require.Equal(t, "auto", flag.DefValue)
	require.NoError(t, cmd.Flags().Set("progress", "plain"))
	require.Equal(t, "plain", flag.Value.String())
}
func TestCachePushRejectsInvalidProgressBeforeOpeningCache(t *testing.T) {
	cmd := cachePushCmd()
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	cmd.SetArgs([]string{"--progress=invalid", "example.test/repo:tag"})
	require.ErrorContains(t, cmd.Execute(), "invalid push progress mode")
}

func TestCachePushTracingFlagsAreOptIn(t *testing.T) {
	cmd := cachePushCmd()
	trace := cmd.Flags().Lookup("trace-upload")
	require.NotNil(t, trace)
	require.Equal(t, "false", trace.DefValue)
	interval := cmd.Flags().Lookup("trace-interval")
	require.NotNil(t, interval)
	require.Equal(t, "5s", interval.DefValue)
	cmd.SetArgs([]string{"--trace-upload", "--progress=tty", "example.test/repo:tag"})
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	require.ErrorContains(t, cmd.Execute(), "incompatible")
}
