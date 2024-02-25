package spec

// PackageResolver is an interface for resolving a template into a proper tagged package name
type PackageResolver func(path string) (tag string, err error)
