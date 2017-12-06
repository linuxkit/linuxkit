package progress

const (
	escape = "\x1b"
	reset  = escape + "[0m"
	red    = escape + "[31m" // nolint: unused, varcheck
	green  = escape + "[32m"
)
