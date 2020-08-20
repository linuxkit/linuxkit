package testdata

import hclog "github.com/hashicorp/go-hclog"

func badHCLog() {
	l := hclog.L()

	l.Info("ok", "key", "val")
	l.Info("bad", "key")
}
