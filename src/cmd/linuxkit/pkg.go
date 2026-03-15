package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/linuxkit/linuxkit/src/cmd/linuxkit/pkglib"
	"github.com/spf13/cobra"
)

var (
	pkglibConfig  pkglib.PkglibConfig
	registryCreds []string
)

func pkgCmd() *cobra.Command {
	var (
		argDisableCache bool
		argEnableCache  bool
		argNoNetwork    bool
		argNetwork      bool
		argOrg          string
		buildYML        string
		hash            string
		hashCommit      string
		hashPath        string
		dirty           bool
		devMode         bool
		tag             string
		hashDir         string
		strictDeps      bool
	)

	cmd := &cobra.Command{
		Use:   "pkg",
		Short: "package building and pushing",
		Long:  `Package building and pushing.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if parent := cmd.Parent(); parent != nil {
				if parent.PersistentPreRunE != nil {
					if err := parent.PersistentPreRunE(parent, args); err != nil {
						return err
					}
				}
			}

			pkglibConfig = pkglib.PkglibConfig{
				BuildYML:   buildYML,
				Hash:       hash,
				HashCommit: hashCommit,
				HashPath:   hashPath,
				Dirty:      dirty,
				Dev:        devMode,
				HashDir:    hashDir,
				StrictDeps: strictDeps,
			}
			if cmd.Flags().Changed("disable-cache") && cmd.Flags().Changed("enable-cache") {
				return errors.New("cannot set but disable-cache and enable-cache")
			}

			if cmd.Flags().Changed("nonetwork") && cmd.Flags().Changed("network") {
				return errors.New("cannot set but nonetwork and network")
			}

			// these should be set only for overrides
			if cmd.Flags().Changed("disable-cache") {
				pkglibConfig.DisableCache = &argDisableCache
			}
			if cmd.Flags().Changed("enable-cache") {
				val := !argEnableCache
				pkglibConfig.DisableCache = &val
			}
			if cmd.Flags().Changed("nonetwork") {
				val := !argNoNetwork
				pkglibConfig.Network = &val
			}
			if cmd.Flags().Changed("network") {
				pkglibConfig.Network = &argNetwork
			}
			if cmd.Flags().Changed("org") {
				pkglibConfig.Org = &argOrg
			} else if org := os.Getenv(envVarPkgOrg); org != "" {
				pkglibConfig.Org = &org
			}
			if cmd.Flags().Changed("tag") {
				pkglibConfig.Tag = tag
			}

			return nil
		},
	}

	// because there is an alias 'pkg push' for 'pkg build --push', we need to add the build command first
	buildCmd := pkgBuildCmd()
	cmd.AddCommand(buildCmd)
	cmd.AddCommand(pkgBuilderCmd())
	cmd.AddCommand(pkgPushCmd(buildCmd))
	cmd.AddCommand(pkgShowTagCmd())
	cmd.AddCommand(pkgManifestCmd())
	cmd.AddCommand(pkgRemoteTagCmd())

	// These override fields in pkgInfo default below, bools are in both forms to allow user overrides in either direction.
	// These will apply to all packages built.
	piBase := pkglib.NewPkgInfo()
	cmd.PersistentFlags().BoolVar(&argDisableCache, "disable-cache", piBase.DisableCache, "Disable build cache")
	cmd.PersistentFlags().BoolVar(&argEnableCache, "enable-cache", !piBase.DisableCache, "Enable build cache")
	cmd.PersistentFlags().BoolVar(&argNoNetwork, "nonetwork", !piBase.Network, "Disallow network use during build")
	cmd.PersistentFlags().BoolVar(&argNetwork, "network", piBase.Network, "Allow network use during build")

	cmd.PersistentFlags().StringVar(&argOrg, "org", piBase.Org, fmt.Sprintf("Override the hub org. Also read from env var %s; CLI flag takes precedence.", envVarPkgOrg))
	cmd.PersistentFlags().StringVar(&buildYML, "build-yml", defaultPkgBuildYML, "Override the name of the yml file")
	cmd.PersistentFlags().StringVar(&hash, "hash", "", "Override the image hash (default is to query git for the package's tree-sh)")
	cmd.PersistentFlags().StringVar(&tag, "tag", piBase.Tag, "Override the tag using fixed strings and/or text templates. Acceptable are .Hash for the hash")
	cmd.PersistentFlags().StringVar(&hashCommit, "hash-commit", defaultPkgCommit, "Override the git commit to use for the hash")
	cmd.PersistentFlags().StringVar(&hashPath, "hash-path", "", "Override the directory to use for the image hash, must be a parent of the package dir (default is to use the package dir)")
	cmd.PersistentFlags().BoolVar(&dirty, "force-dirty", false, "Force the pkg(s) to be considered dirty")
	cmd.PersistentFlags().BoolVar(&devMode, "dev", false, "Force org and hash to $USER and \"dev\" respectively")
	cmd.PersistentFlags().StringVar(&hashDir, "hash-dir", "", "Directory containing per-package .hash manifest files (written by show-tag --hash-dir). When set, @lkt: dep tags are read from these files instead of being recursively computed, enabling correct version-specific tag propagation (e.g. ZFS_VERSION) without dependency cycles.")
	cmd.PersistentFlags().BoolVar(&strictDeps, "strict-deps", false, "Error if a dep's .hash file is absent from --hash-dir (default: fall back to NewFromConfig)")

	cmd.PersistentFlags().StringSliceVar(&registryCreds, "registry-creds", nil, "Registry auths to use for building images, format is <registry>=<username>:<password> OR <registry>=<registry-token-base64>; do NOT forget to base64 encode it. If no username is provided, it is treated as a registry token. <registry> must be a URL, e.g. 'https://index.docker.io/'. May be provided as many times as desired. Will override anything in your default.")
	return cmd
}
