package main

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/infrakit/discovery"
	"github.com/docker/infrakit/plugin/group/types"
	"github.com/docker/infrakit/spi/flavor"
	flavor_plugin "github.com/docker/infrakit/spi/http/flavor"
	"github.com/docker/infrakit/spi/instance"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
	"strings"
)

func flavorPluginCommand(plugins func() discovery.Plugins) *cobra.Command {

	name := ""
	var flavorPlugin flavor.Plugin

	cmd := &cobra.Command{
		Use:   "flavor",
		Short: "Access flavor plugin",
		PersistentPreRunE: func(c *cobra.Command, args []string) error {

			callable, err := plugins().Find(name)
			if err != nil {
				return err
			}
			flavorPlugin = flavor_plugin.PluginClient(callable)

			return nil
		},
	}
	cmd.PersistentFlags().StringVar(&name, "name", name, "Name of plugin")

	logicalIDs := []string{}
	groupSize := uint(0)
	addAllocationMethodFlags := func(cmd *cobra.Command) {
		cmd.Flags().StringSliceVar(
			&logicalIDs,
			"logical-ids",
			[]string{},
			"Logical IDs to use as the Allocation method")
		cmd.Flags().UintVar(
			&groupSize,
			"size",
			0,
			"Group Size to use as the Allocation method")
	}

	allocationMethodFromFlags := func() types.AllocationMethod {
		ids := []instance.LogicalID{}
		for _, id := range logicalIDs {
			ids = append(ids, instance.LogicalID(id))
		}

		return types.AllocationMethod{
			Size:       groupSize,
			LogicalIDs: ids,
		}
	}

	validate := &cobra.Command{
		Use:   "validate <flavor configuration file>",
		Short: "validate a flavor configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			assertNotNil("no plugin", flavorPlugin)

			if len(args) != 1 {
				cmd.Usage()
				os.Exit(1)
			}

			buff, err := ioutil.ReadFile(args[0])
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			return flavorPlugin.Validate(json.RawMessage(buff), allocationMethodFromFlags())
		},
	}
	addAllocationMethodFlags(validate)

	prepare := &cobra.Command{
		Use:   "prepare <flavor configuration file> <instance Spec JSON file>",
		Short: "prepare provisioning inputs for an instance",
		RunE: func(cmd *cobra.Command, args []string) error {
			assertNotNil("no plugin", flavorPlugin)

			if len(args) != 2 {
				cmd.Usage()
				os.Exit(1)
			}

			flavorProperties, err := ioutil.ReadFile(args[0])
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			buff, err := ioutil.ReadFile(args[1])
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			spec := instance.Spec{}
			if err := json.Unmarshal(buff, &spec); err != nil {
				return err
			}

			spec, err = flavorPlugin.Prepare(
				json.RawMessage(flavorProperties),
				spec,
				allocationMethodFromFlags())
			if err == nil {
				buff, err = json.MarshalIndent(spec, "  ", "  ")
				if err == nil {
					fmt.Println(string(buff))
				}
			}
			return err
		},
	}
	addAllocationMethodFlags(prepare)

	tags := []string{}
	id := ""
	logicalID := ""
	healthy := &cobra.Command{
		Use:   "healthy <flavor configuration file>",
		Short: "checks if an instance is considered healthy",
		RunE: func(cmd *cobra.Command, args []string) error {
			assertNotNil("no plugin", flavorPlugin)

			if len(args) != 1 {
				cmd.Usage()
				os.Exit(1)
			}

			flavorProperties, err := ioutil.ReadFile(args[0])
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			filter := map[string]string{}
			for _, t := range tags {
				p := strings.Split(t, "=")
				if len(p) == 2 {
					filter[p[0]] = p[1]
				} else {
					filter[p[0]] = ""
				}
			}

			desc := instance.Description{}
			if len(filter) > 0 {
				desc.Tags = filter
			}
			if id != "" {
				desc.ID = instance.ID(id)
			}
			if logicalID != "" {
				logical := instance.LogicalID(logicalID)
				desc.LogicalID = &logical
			}

			healthy, err := flavorPlugin.Healthy(json.RawMessage(flavorProperties), desc)
			if err == nil {
				fmt.Printf("%v\n", healthy)
			}
			return err
		},
	}
	healthy.Flags().StringSliceVar(&tags, "tags", tags, "Tags to filter")
	healthy.Flags().StringVar(&id, "id", id, "ID of resource")
	healthy.Flags().StringVar(&logicalID, "logical-id", logicalID, "Logical ID of resource")

	cmd.AddCommand(validate, prepare, healthy)

	return cmd
}
