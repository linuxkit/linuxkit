package util

// RunStop is an operation that may be Run (synchronously) and interrupted by calling Stop.
type RunStop interface {
	Run()

	Stop()
}
