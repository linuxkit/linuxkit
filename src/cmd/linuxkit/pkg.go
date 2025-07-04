package main

import (
	"errors"

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
	)

	cmd := &cobra.Command{
		Use:   "pkg",
		Short: "package building and pushing",
		Long:  `Package building and pushing.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			pkglibConfig = pkglib.PkglibConfig{
				BuildYML:   buildYML,
				Hash:       hash,
				HashCommit: hashCommit,
				HashPath:   hashPath,
				Dirty:      dirty,
				Dev:        devMode,
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

	cmd.PersistentFlags().StringVar(&argOrg, "org", piBase.Org, "Override the hub org")
	cmd.PersistentFlags().StringVar(&buildYML, "build-yml", defaultPkgBuildYML, "Override the name of the yml file")
	cmd.PersistentFlags().StringVar(&hash, "hash", "", "Override the image hash (default is to query git for the package's tree-sh)")
	cmd.PersistentFlags().StringVar(&tag, "tag", piBase.Tag, "Override the tag using fixed strings and/or text templates. Acceptable are .Hash for the hash")
	cmd.PersistentFlags().StringVar(&hashCommit, "hash-commit", defaultPkgCommit, "Override the git commit to use for the hash")
	cmd.PersistentFlags().StringVar(&hashPath, "hash-path", "", "Override the directory to use for the image hash, must be a parent of the package dir (default is to use the package dir)")
	cmd.PersistentFlags().BoolVar(&dirty, "force-dirty", false, "Force the pkg(s) to be considered dirty")
	cmd.PersistentFlags().BoolVar(&devMode, "dev", false, "Force org and hash to $USER and \"dev\" respectively")

	cmd.PersistentFlags().StringSliceVar(&registryCreds, "registry-creds", nil, "Registry auths to use for building images, format is <registry>=<username>:<password> OR <registry>=<registry-token>. If no username is provided, it is treated as a registry token. <registry> must be a URL, e.g. 'https://index.docker.io/'. May be provided as many times as desired. Will override anything in your default.")
	return cmd
}
