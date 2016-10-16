package bootstrap

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/docker/infrakit.aws/plugin/instance"
	"github.com/docker/infrakit/spi/group"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"io/ioutil"
	"os"
)

// NewCLI creates a CLI.
func NewCLI() *CLI {
	return &CLI{}
}

// CLI is a CLI for AWS bootstrapping.
type CLI struct {
}

func readConfig(swimFile string) (fakeSWIMSchema, error) {
	swim := fakeSWIMSchema{}
	swimData, err := ioutil.ReadFile(swimFile)
	if err != nil {
		return swim, fmt.Errorf("Failed to read config file: %s", err)
	}

	err = json.Unmarshal(swimData, &swim)
	if err != nil {
		return swim, err
	}

	err = swim.validate()
	if err != nil {
		return swim, err
	}

	swim.applyDefaults()

	return swim, err
}

type clusterIDFlags struct {
	ID clusterID
}

func (c *clusterIDFlags) flags() *pflag.FlagSet {
	clusterIDFlags := pflag.NewFlagSet("cluster ID", pflag.ExitOnError)
	clusterIDFlags.StringVar(&c.ID.region, "region", "", "AWS region")
	clusterIDFlags.StringVar(&c.ID.name, "cluster", "", "Infrakit cluster name")
	return clusterIDFlags
}

func (c *clusterIDFlags) valid() bool {
	return c.ID.region != "" && c.ID.name != ""
}

func abort(format string, args ...interface{}) {
	log.Fatalf(format, args...)
	os.Exit(1)
}

// Command gets the subcommand for the CLI.
func (a *CLI) Command() *cobra.Command {
	cmd := cobra.Command{Use: "aws", Short: "manage Swarm clusters in AWS"}

	cluster := clusterIDFlags{}

	var keyName string

	workerSize := 3

	createCmd := cobra.Command{
		Use:   "create [<swim config>]",
		Short: "create a swarm cluster",
		Run: func(cmd *cobra.Command, args []string) {
			swim := fakeSWIMSchema{}
			if len(args) == 1 {
				if keyName != "" || cluster.ID.name != "" || cluster.ID.region != "" {
					abort("No other cluster-related flags may be set when a SWIM file is used")
				}

				var err error
				swim, err = readConfig(args[0])
				if err != nil {
					abort("Invalid config file: %s", err)
				}
			} else {
				if keyName == "" || !cluster.valid() {
					abort("When creating from flags, --key, --cluster, and --region must be provided")
				}

				instanceConfig := instance.CreateInstanceRequest{
					RunInstancesInput: ec2.RunInstancesInput{
						ImageId: aws.String("ami-2ef48339"),
						KeyName: aws.String(keyName),
						Placement: &ec2.Placement{
							// TODO(wfarner): Picking the AZ like this feels hackish.
							AvailabilityZone: aws.String(cluster.ID.region + "a"),
						},
					},
				}

				swim = fakeSWIMSchema{
					Driver:      "aws",
					ClusterName: cluster.ID.name,
					Groups: []instanceGroup{
						{
							Name:   group.ID("Managers"),
							Type:   managerType,
							Size:   3,
							Config: instanceConfig,
						},
						{
							Name:   group.ID("Workers"),
							Type:   workerType,
							Size:   workerSize,
							Config: instanceConfig,
						},
					},
				}

				err := swim.validate()
				if err != nil {
					abort(err.Error())
				}

				swim.applyDefaults()
			}

			err := bootstrap(swim)
			if err != nil {
				abort(err.Error())
			}
		},
	}
	createCmd.Flags().AddFlagSet(cluster.flags())
	createCmd.Flags().StringVar(&keyName, "key", "", "The existing SSH key in AWS to use for provisioned instances")
	createCmd.Flags().IntVar(&workerSize, "worker_size", workerSize, "Size of worker group")

	cmd.AddCommand(&createCmd)

	var swimFile string
	destroyCmd := cobra.Command{
		Use:   "destroy",
		Short: "destroy a swarm cluster",
		Long: `destroy all resources associated with a SWIM cluster

The cluster may be identified manually or based on the contents of a SWIM file.`,
		Run: func(cmd *cobra.Command, args []string) {
			var id clusterID
			if swimFile == "" {
				if !cluster.valid() {
					abort("Must specify --config or both of --region and --cluster")
				}

				id = cluster.ID
			} else {
				swim, err := readConfig(swimFile)
				if err != nil {
					abort("Invalid config file: %s", err)
				}
				id = swim.cluster()
			}

			err := destroy(id)
			if err != nil {
				abort(err.Error())
			}
		},
	}
	destroyCmd.Flags().StringVar(&swimFile, "config", "", "A SWIM file")

	destroyCmd.Flags().AddFlagSet(cluster.flags())
	cmd.AddCommand(&destroyCmd)

	// Commands that are to be executed ON THE SWARM
	// So in these cases we allow specification of a local api server which can trigger updates.
	// TODO(chungers) -- the api server will basically become the docker engine subsystem later.

	apiEndpoint := ""

	reconfigureCmd := &cobra.Command{
		Use: "reconfigure <swim config>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.Usage()
				os.Exit(1)
			}

			swim, err := readConfig(args[0])
			if err != nil {
				abort("Invalid config file: %s", err)
			}

			// TODO(wfarner): Fetch the existing config and check that the requested change is possible.

			err = swim.push()
			if err != nil {
				abort("Failed to push config: %s", err)
			}
			log.Infof("Configuration pushed")
		},
	}
	reconfigureCmd.Flags().StringVar(&apiEndpoint, "api", apiEndpoint, "Infrakit subsystem api endpoint")
	cmd.AddCommand(reconfigureCmd)

	describeCmd := &cobra.Command{
		Use: "describe <swim config>",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.Usage()
				os.Exit(1)
			}

			swim, err := readConfig(args[0])
			if err != nil {
				abort("Invalid config file: %s", err)
			}

			groups := []string{}
			for _, group := range swim.Groups {
				groups = append(groups, string(group.Name))
			}

			log.Infof("Groups: %s", groups)
		},
	}
	describeCmd.Flags().StringVar(&apiEndpoint, "api", apiEndpoint, "Infrakit subsystem api endpoint")
	cmd.AddCommand(describeCmd)

	statusCmd := &cobra.Command{
		Use: "status",
		Run: func(cmd *cobra.Command, args []string) {
			// TODO(wfarner): Implement.

			log.Infof("Managers: 3 instances")
			log.Infof("Workers: 5 instances")
		},
	}
	statusCmd.Flags().StringVar(&apiEndpoint, "api", apiEndpoint, "Infrakit subsystem api endpoint")
	cmd.AddCommand(statusCmd)

	return &cmd
}

type logger struct {
}

func (l logger) Log(args ...interface{}) {
	log.Println(args)
}
