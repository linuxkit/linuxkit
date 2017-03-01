package main

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/discovery"
	"github.com/docker/infrakit/spi/group"
	group_plugin "github.com/docker/infrakit/spi/http/group"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

const (
	// DefaultGroupPluginName specifies the default name of the group plugin if name flag isn't specified.
	DefaultGroupPluginName = "group"
)

func groupPluginCommand(plugins func() discovery.Plugins) *cobra.Command {

	name := DefaultGroupPluginName
	var groupPlugin group.Plugin

	cmd := &cobra.Command{
		Use:   "group",
		Short: "Access group plugin",
		PersistentPreRunE: func(c *cobra.Command, args []string) error {

			callable, err := plugins().Find(name)
			if err != nil {
				return err
			}
			groupPlugin = group_plugin.PluginClient(callable)

			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&name, "name", name, "Name of plugin")

	watch := &cobra.Command{
		Use:   "watch <group configuration>",
		Short: "watch a group",
		RunE: func(cmd *cobra.Command, args []string) error {
			assertNotNil("no plugin", groupPlugin)

			if len(args) != 1 {
				cmd.Usage()
				os.Exit(1)
			}

			buff, err := ioutil.ReadFile(args[0])
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			spec := group.Spec{}
			if err := json.Unmarshal(buff, &spec); err != nil {
				return err
			}

			err = groupPlugin.WatchGroup(spec)
			if err == nil {
				fmt.Println("watching", spec.ID)
			}
			return err
		},
	}

	unwatch := &cobra.Command{
		Use:   "unwatch <group ID>",
		Short: "unwatch a group",
		RunE: func(cmd *cobra.Command, args []string) error {
			assertNotNil("no plugin", groupPlugin)

			if len(args) != 0 {
				cmd.Usage()
				os.Exit(1)
			}

			groupID := group.ID(args[0])
			err := groupPlugin.UnwatchGroup(groupID)

			if err == nil {
				fmt.Println("unwatched", groupID)
			}
			return err
		},
	}

	var quiet bool
	inspect := &cobra.Command{
		Use:   "inspect <group ID>",
		Short: "inspect a group",
		RunE: func(cmd *cobra.Command, args []string) error {
			assertNotNil("no plugin", groupPlugin)

			if len(args) != 1 {
				cmd.Usage()
				os.Exit(1)
			}

			groupID := group.ID(args[0])
			desc, err := groupPlugin.InspectGroup(groupID)

			if err == nil {
				if !quiet {
					fmt.Printf("%-30s\t%-30s\t%-s\n", "ID", "LOGICAL", "TAGS")
				}
				for _, d := range desc.Instances {
					logical := "  -   "
					if d.LogicalID != nil {
						logical = string(*d.LogicalID)
					}

					printTags := []string{}
					for k, v := range d.Tags {
						printTags = append(printTags, fmt.Sprintf("%s=%s", k, v))
					}
					sort.Strings(printTags)

					fmt.Printf("%-30s\t%-30s\t%-s\n", d.ID, logical, strings.Join(printTags, ","))
				}
			}
			return err
		},
	}
	inspect.Flags().BoolVarP(&quiet, "quiet", "q", false, "Print rows without column headers")

	describe := &cobra.Command{
		Use:   "describe <group configuration file>",
		Short: "describes the steps to perform an update",
		RunE: func(cmd *cobra.Command, args []string) error {
			assertNotNil("no plugin", groupPlugin)

			if len(args) != 1 {
				cmd.Usage()
				os.Exit(1)
			}

			buff, err := ioutil.ReadFile(args[0])
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			spec := group.Spec{}
			if err := json.Unmarshal(buff, &spec); err != nil {
				return err
			}

			desc, err := groupPlugin.DescribeUpdate(spec)
			if err == nil {
				fmt.Println(desc)
			}
			return err
		},
	}

	update := &cobra.Command{
		Use:   "update [group configuration]",
		Short: "update group (update < file or update filename)",
		RunE: func(cmd *cobra.Command, args []string) error {
			assertNotNil("no plugin", groupPlugin)

			if len(args) != 1 {
				cmd.Usage()
				os.Exit(1)
			}

			buff, err := ioutil.ReadFile(args[0])
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			spec := group.Spec{}
			if err := json.Unmarshal(buff, &spec); err != nil {
				return err
			}

			// TODO - make this not block, but how to get status?
			err = groupPlugin.UpdateGroup(spec)
			if err == nil {
				fmt.Println("update", spec.ID, "completed")
			}
			return err
		},
	}

	stop := &cobra.Command{
		Use:   "stop <group ID>",
		Short: "stop updating a group",
		RunE: func(cmd *cobra.Command, args []string) error {
			assertNotNil("no plugin", groupPlugin)

			if len(args) != 1 {
				cmd.Usage()
				os.Exit(1)
			}

			groupID := group.ID(args[0])
			err := groupPlugin.StopUpdate(groupID)

			if err == nil {
				fmt.Println("update", groupID, "stopped")
			}
			return err
		},
	}

	destroy := &cobra.Command{
		Use:   "destroy <group ID>",
		Short: "destroy a group",
		RunE: func(cmd *cobra.Command, args []string) error {
			assertNotNil("no plugin", groupPlugin)

			if len(args) != 1 {
				cmd.Usage()
				os.Exit(1)
			}

			groupID := group.ID(args[0])
			err := groupPlugin.DestroyGroup(groupID)

			if err == nil {
				fmt.Println("destroy", groupID, "initiated")
			}
			return err
		},
	}

	cmd.AddCommand(watch, unwatch, inspect, describe, update, stop, destroy)

	return cmd
}
