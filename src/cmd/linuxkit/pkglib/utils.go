package pkglib

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/containerd/platforms"
	"github.com/docker/go-units"
	buildkitClient "github.com/moby/buildkit/client"
)

func printTableHeader(tw *tabwriter.Writer) {
	_, _ = fmt.Fprintln(tw, "ID\tRECLAIMABLE\tSIZE\tLAST ACCESSED")
}

func printTableRow(tw *tabwriter.Writer, di *buildkitClient.UsageInfo) {
	id := di.ID
	if di.Mutable {
		id += "*"
	}
	size := units.HumanSize(float64(di.Size))
	if di.Shared {
		size += "*"
	}
	lastAccessed := ""
	if di.LastUsedAt != nil {
		lastAccessed = units.HumanDuration(time.Since(*di.LastUsedAt)) + " ago"
	}
	_, _ = fmt.Fprintf(tw, "%-40s\t%-5v\t%-10s\t%s\n", id, !di.InUse, size, lastAccessed)
}

func printSummary(tw *tabwriter.Writer, du []*buildkitClient.UsageInfo) {
	total := int64(0)
	reclaimable := int64(0)
	shared := int64(0)

	for _, di := range du {
		if di.Size > 0 {
			total += di.Size
			if !di.InUse {
				reclaimable += di.Size
			}
		}
		if di.Shared {
			shared += di.Size
		}
	}

	if shared > 0 {
		_, _ = fmt.Fprintf(tw, "Shared:\t%s\n", units.HumanSize(float64(shared)))
		_, _ = fmt.Fprintf(tw, "Private:\t%s\n", units.HumanSize(float64(total-shared)))
	}

	_, _ = fmt.Fprintf(tw, "Reclaimable:\t%s\n", units.HumanSize(float64(reclaimable)))
	_, _ = fmt.Fprintf(tw, "Total:\t%s\n", units.HumanSize(float64(total)))
	_ = tw.Flush()
}

func printKV(w io.Writer, k string, v interface{}) {
	_, _ = fmt.Fprintf(w, "%s:\t%v\n", k, v)
}

func printVerbose(tw *tabwriter.Writer, du []*buildkitClient.UsageInfo) {
	for _, di := range du {
		printKV(tw, "ID", di.ID)
		if len(di.Parents) != 0 {
			printKV(tw, "Parent", strings.Join(di.Parents, ","))
		}
		printKV(tw, "Created at", di.CreatedAt)
		printKV(tw, "Mutable", di.Mutable)
		printKV(tw, "Reclaimable", !di.InUse)
		printKV(tw, "Shared", di.Shared)
		printKV(tw, "Size", units.HumanSize(float64(di.Size)))
		if di.Description != "" {
			printKV(tw, "Description", di.Description)
		}
		printKV(tw, "Usage count", di.UsageCount)
		if di.LastUsedAt != nil {
			printKV(tw, "Last used", units.HumanDuration(time.Since(*di.LastUsedAt))+" ago")
		}
		if di.RecordType != "" {
			printKV(tw, "Type", di.RecordType)
		}

		_, _ = fmt.Fprintf(tw, "\n")
	}

	_ = tw.Flush()
}

func getClientForPlatform(ctx context.Context, buildersMap map[string]string, builderImage, builderConfigPath, platform string) (*buildkitClient.Client, error) {
	p, err := platforms.Parse(platform)
	if err != nil {
		return nil, fmt.Errorf("failed to parse platform: %s", err)
	}
	dr := NewDockerRunner(false)
	builderName := getBuilderForPlatform(p.Architecture, buildersMap)
	client, err := dr.Builder(ctx, builderName, builderImage, builderConfigPath, platform, false)
	if err != nil {
		return nil, fmt.Errorf("unable to ensure builder container: %v", err)
	}
	return client, nil
}

// DiskUsage of builder
func DiskUsage(buildersMap map[string]string, builderImage, builderConfigPath string, platformsToClean []string, verbose bool) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, platform := range platformsToClean {
		client, err := getClientForPlatform(ctx, buildersMap, builderImage, builderConfigPath, platform)
		if err != nil {
			return fmt.Errorf("cannot get client: %s", err)
		}

		du, err := client.DiskUsage(ctx)
		if err != nil {
			_ = client.Close()
			return err
		}
		err = client.Close()
		if err != nil {
			return fmt.Errorf("cannot close client: %s", err)
		}
		tw := tabwriter.NewWriter(os.Stdout, 1, 8, 1, '\t', 0)
		if len(du) > 0 {
			if verbose {
				printVerbose(tw, du)
			} else {
				printTableHeader(tw)
				for _, di := range du {
					printTableRow(tw, di)
				}
			}
		}
		printSummary(tw, du)
	}
	return nil
}

// PruneBuilder clean build cache of builder
func PruneBuilder(buildersMap map[string]string, builderImage, builderConfigPath string, platformsToClean []string, verbose bool) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	total := int64(0)
	for _, platform := range platformsToClean {
		client, err := getClientForPlatform(ctx, buildersMap, builderImage, builderConfigPath, platform)
		if err != nil {
			return fmt.Errorf("cannot get client: %s", err)
		}

		ch := make(chan buildkitClient.UsageInfo)
		processed := make(chan struct{})

		go func() {
			defer close(processed)
			for du := range ch {
				if verbose {
					fmt.Printf("%s\t%s\tremoved\n", du.ID, units.HumanSize(float64(du.Size)))
				}
				total += du.Size
			}
		}()
		err = client.Prune(ctx, ch)
		if err != nil {
			_ = client.Close()
			close(ch)
			return err
		}
		err = client.Close()
		if err != nil {
			return fmt.Errorf("cannot close client: %s", err)
		}
		close(ch)
		<-processed
	}
	fmt.Printf("Reclaimed:\t%s\n", units.BytesSize(float64(total)))
	return nil
}
