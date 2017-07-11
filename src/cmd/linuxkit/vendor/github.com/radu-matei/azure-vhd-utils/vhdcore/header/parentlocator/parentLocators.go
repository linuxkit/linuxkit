package parentlocator

// ParentLocators type represents the parent locator collection (parent-hard-disk-locator-info
// collection). The collection entries store an absolute byte offset in the file where the parent
// locator for a differencing hard disk is stored. This field is used only for differencing disks
// and should be set to zero for dynamic disks.
//
type ParentLocators []*ParentLocator

// GetAbsoluteParentPath returns the absolute path to the parent differencing hard disk
//
func (p ParentLocators) GetAbsoluteParentPath() string {
	return p.getParentPath(PlatformCodeW2Ku)
}

// GetRelativeParentPath returns the relative path to the parent differencing hard disk
//
func (p ParentLocators) GetRelativeParentPath() string {
	return p.getParentPath(PlatformCodeW2Ru)
}

// getParentPath returns path to the parent differencing hard disk corresponding to the
// given platform code
//
func (p ParentLocators) getParentPath(code PlatformCode) string {
	for _, l := range p {
		if l.PlatformCode == code {
			return l.PlatformSpecificFileLocator
		}
	}

	return ""
}
