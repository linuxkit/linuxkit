package instance

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/docker/infrakit/spi/instance"
	"github.com/spf13/pflag"
	"log"
	"os"
)

type options struct {
	region          string
	accessKeyID     string
	secretAccessKey string
	sessionToken    string
	retries         int
}

// Builder is a ProvisionerBuilder that creates an AWS instance provisioner.
type Builder struct {
	Config  client.ConfigProvider
	options options
}

// Flags returns the flags required.
func (b *Builder) Flags() *pflag.FlagSet {
	flags := pflag.NewFlagSet("aws", pflag.PanicOnError)
	flags.StringVar(&b.options.region, "region", "", "AWS region")
	flags.StringVar(&b.options.accessKeyID, "access-key-id", "", "IAM access key ID")
	flags.StringVar(&b.options.secretAccessKey, "secret-access-key", "", "IAM access key secret")
	flags.StringVar(&b.options.sessionToken, "session-token", "", "AWS STS token")
	flags.IntVar(&b.options.retries, "retries", 5, "Number of retries for AWS API operations")
	return flags
}

// BuildInstancePlugin creates an instance Provisioner configured with the Flags.
func (b *Builder) BuildInstancePlugin() (instance.Plugin, error) {
	if b.Config == nil {
		providers := []credentials.Provider{
			&ec2rolecreds.EC2RoleProvider{Client: ec2metadata.New(session.New())},
			&credentials.EnvProvider{},
			&credentials.SharedCredentialsProvider{},
		}

		if (len(b.options.accessKeyID) > 0 && len(b.options.secretAccessKey) > 0) || len(b.options.sessionToken) > 0 {
			staticCreds := credentials.StaticProvider{
				Value: credentials.Value{
					AccessKeyID:     b.options.accessKeyID,
					SecretAccessKey: b.options.secretAccessKey,
					SessionToken:    b.options.sessionToken,
				},
			}
			providers = append(providers, &staticCreds)
		}

		if b.options.region == "" {
			log.Println("region not specified, attempting to discover from EC2 instance metadata")
			region, err := GetRegion()
			if err != nil {
				return nil, errors.New("Unable to determine region")
			}

			log.Printf("Defaulting to local region %s\n", region)
			b.options.region = region
		}

		b.Config = session.New(aws.NewConfig().
			WithRegion(b.options.region).
			WithCredentials(credentials.NewChainCredentials(providers)).
			WithLogger(GetLogger()).
			//WithLogLevel(aws.LogDebugWithRequestErrors).
			WithMaxRetries(b.options.retries))
	}

	return NewInstancePlugin(ec2.New(b.Config)), nil
}

type logger struct {
	logger *log.Logger
}

func (l logger) Log(args ...interface{}) {
	l.logger.Println(args...)
}

// GetLogger gets a logger that can be used with the AWS SDK.
func GetLogger() aws.Logger {
	return &logger{
		logger: log.New(os.Stderr, "", log.LstdFlags),
	}
}
