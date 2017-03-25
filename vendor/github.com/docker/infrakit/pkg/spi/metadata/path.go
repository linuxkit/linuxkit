package metadata

var (
	// NullPath means no path
	NullPath = Path([]string{})
)

// Path is used to identify a particle of metadata.  The path can be strings separated by / as in a URL.
type Path []string

// Clean scrubs the path to remove any empty string or . or .. and collapse the path into a concise form.
// It's similar to path/filepath.Clean in the standard lib.
func (p Path) Clean() Path {
	this := []string(p)
	copy := []string{}
	for _, v := range this {
		switch v {
		case "", ".":
		case "..":
			if len(copy) == 0 {
				copy = append(copy, "..")
			} else {
				copy = copy[0 : len(copy)-1]
				if len(copy) == 0 {
					return NullPath
				}
			}
		default:
			copy = append(copy, v)

		}
	}
	return Path(copy)
}

// Len returns the length of the path
func (p Path) Len() int {
	return len([]string(p))
}

// Index returns the ith component in the path
func (p Path) Index(i int) *string {
	if p.Len() <= i {
		return nil
	}
	copy := []string(p)[i]
	return &copy
}

// Shift returns a new path that's shifted i positions to the left -- ith child of the head at index=0
func (p Path) Shift(i int) Path {
	len := p.Len() - i
	if len <= 0 {
		return Path([]string{})
	}
	new := make([]string, len)
	copy(new, []string(p)[i:])
	return Path(new)
}

// Dir returns the 'dir' of the path
func (p Path) Dir() Path {
	pp := p.Clean()
	if len(pp) > 1 {
		return p[0 : len(pp)-1]
	}
	return Path([]string{"."})
}

// Base returns the base of the path
func (p Path) Base() string {
	pp := p.Clean()
	return pp[len(pp)-1]
}

// Join joins the input as a child of this path
func (p Path) Join(child string) Path {
	return p.Sub(Path([]string{child}))
}

// Sub joins the child to the parent
func (p Path) Sub(child Path) Path {
	pp := p.Clean()
	return Path(append(pp, []string(child)...))
}

// Rel returns a new path that is a child of the input from this path.
// e.g. For a path a/b/c/d Rel(a/b/) returns c/d.  NullPath is returned if
// the two are not relative to one another.
func (p Path) Rel(path Path) Path {
	this := []string(p.Clean())
	parent := []string(path.Clean())
	if len(this) < len(parent) {
		return NullPath
	}
	for i := 0; i < len(parent); i++ {
		if parent[i] != this[i] {
			return NullPath
		}
	}
	return Path(this[len(parent):])
}
