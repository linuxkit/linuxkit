package pkglib

import (
	"fmt"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/registry"
	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/util"
)

// Index create an index for the package tag based on all arch-specific tags in the registry.
func (p Pkg) Index(bos ...BuildOpt) error {
	var bo buildOpts
	for _, fn := range bos {
		if err := fn(&bo); err != nil {
			return err
		}
	}
	name := p.FullTag()

	// Even though we may have pushed the index, we want to be sure that we have an index that includes every architecture on the registry,
	// not just those that were in our local cache. So we use manifest-tool library to build a broad index
	auth, err := registry.GetDockerAuth()
	if err != nil {
		return fmt.Errorf("failed to get auth: %v", err)
	}

	// push based on tag
	fmt.Printf("Pushing index based on all arch-specific images in registry %s\n", name)
	_, _, err = registry.PushManifest(name, auth)
	if err != nil {
		return err
	}

	// push based on release
	if bo.release != "" {
		relTag, err := p.ReleaseTag(bo.release)
		if err != nil {
			return err
		}
		fullRelTag := util.ReferenceExpand(relTag)

		fmt.Printf("Pushing index based on all arch-specific images in registry %s\n", fullRelTag)
		_, _, err = registry.PushManifest(fullRelTag, auth)
		if err != nil {
			return err
		}
	}

	return nil
}
