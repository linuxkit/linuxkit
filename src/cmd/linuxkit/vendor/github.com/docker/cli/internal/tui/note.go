// FIXME(thaJeztah): remove once we are a module; the go:build directive prevents go from downgrading language version to go1.16:
//go:build go1.23

package tui

import (
	"fmt"
	"strings"

	"github.com/morikuni/aec"
)

var InfoHeader = Str{
	Plain: " Info -> ",
	Fancy: aec.Bold.Apply(aec.LightCyanB.Apply(aec.BlackF.Apply("i")) + " " + aec.LightCyanF.Apply("Info → ")),
}

func (o Output) PrintNote(format string, args ...any) {
	if o.isTerminal {
		// TODO: Handle all flags
		format = strings.ReplaceAll(format, "--platform", ColorFlag.Apply("--platform"))
	}

	header := o.Sprint(InfoHeader)

	_, _ = fmt.Fprint(o, "\n", header)
	s := fmt.Sprintf(format, args...)
	for idx, line := range strings.Split(s, "\n") {
		if idx > 0 {
			_, _ = fmt.Fprint(o, strings.Repeat(" ", Width(header)))
		}

		l := line
		if o.isTerminal {
			l = aec.Italic.Apply(l)
		}
		_, _ = fmt.Fprintln(o, l)
	}
}
